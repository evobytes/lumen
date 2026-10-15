package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const projectLicense = "Apache-2.0"

// copyrightRe matches the one line of the COPYRIGHT file: the single source of
// truth for the project's year(s) and holder. The year is YYYY or YYYY-YYYY.
var copyrightRe = regexp.MustCompile(`^Copyright (\d{4}(?:-\d{4})?) (\S.*)$`)

// projectCopyright is the parsed COPYRIGHT file.
type projectCopyright struct{ year, holder string }

// String is the statement as written in COPYRIGHT and NOTICE.
func (c projectCopyright) String() string { return "Copyright " + c.year + " " + c.holder }

// loadCopyright reads and validates COPYRIGHT.
func loadCopyright(root string) (projectCopyright, error) {
	data, err := os.ReadFile(filepath.Join(root, "COPYRIGHT")) //nolint:gosec // repo-relative
	if err != nil {
		return projectCopyright{}, err
	}
	return parseCopyright(string(data))
}

func parseCopyright(text string) (projectCopyright, error) {
	m := copyrightRe.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return projectCopyright{}, fmt.Errorf("COPYRIGHT must be the single line %q, got %q",
			"Copyright YYYY Holder Name", strings.TrimSpace(text))
	}
	return projectCopyright{year: m[1], holder: strings.TrimSpace(m[2])}, nil
}

// noticeFile renders NOTICE (Apache-2.0 section 4(d) attribution file).
func noticeFile(c projectCopyright) string {
	return "Lumen\n" + c.String() + "\n\n" +
		"This product is licensed under the Apache License, Version 2.0 (see LICENSE).\n\n" +
		"It links third-party Go modules under their own licences; the full list with\n" +
		"licence texts is in THIRD_PARTY_NOTICES.md (installed next to this file in the\n" +
		"packages).\n\n" +
		"It does not include or redistribute the NDI runtime, GStreamer, FFmpeg, OBS or any\n" +
		"Blackmagic software. Optional integrations load or run them as separate, user-installed\n" +
		"programs under their own licences.\n"
}

// module is a Go module compiled into the release binary.
type module struct{ path, version, dir string }

var licenseFileRe = regexp.MustCompile(`(?i)^(licen[cs]e|copying|unlicense|notice|patents)`)

// shippedPlatforms are the GOOS/GOARCH pairs whose builds ship; the module set is their union.
// CGO is off for listing: it does not change the module set (Wails' cgo files import
// the same Go modules), and it keeps the result identical on every host.
var shippedPlatforms = [][2]string{{"windows", "amd64"}, {"linux", "amd64"}, {"darwin", "arm64"}}

// goModules returns the third-party modules compiled into any shipped binary
// (Windows, Linux and macOS builds), sorted by path.
func goModules(root string) ([]module, error) {
	seen := map[string]module{}
	for _, plat := range shippedPlatforms {
		goos, goarch := plat[0], plat[1]
		cmd := exec.Command("go", "list", "-e", "-deps", "-json", "./cmd/lumen") //nolint:gosec,noctx // fixed arguments
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("go list (%s): %w: %s", goos, err, stderr.String())
		}
		dec := json.NewDecoder(bytes.NewReader(out))
		for dec.More() {
			var p struct {
				Module *struct {
					Path, Version, Dir string
					Main               bool
				}
			}
			if err := dec.Decode(&p); err != nil {
				return nil, err
			}
			if m := p.Module; m != nil && !m.Main {
				seen[m.Path] = module{m.Path, m.Version, m.Dir}
			}
		}
	}
	mods := make([]module, 0, len(seen))
	for _, m := range seen {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].path < mods[j].path })
	return mods, nil
}

// licenseTexts reads the licence-type files at the top of dir, primary first.
func licenseTexts(dir string) (primary string, files []licenseFile, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !licenseFileRe.MatchString(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // module cache file
		if err != nil {
			return "", nil, err
		}
		files = append(files, licenseFile{e.Name(), strings.TrimSpace(string(b))})
	}
	sort.Slice(files, func(i, j int) bool {
		pi, pj := isPrimary(files[i].name), isPrimary(files[j].name)
		if pi != pj {
			return pi
		}
		return files[i].name < files[j].name
	})
	if len(files) > 0 && isPrimary(files[0].name) {
		primary = files[0].text
	}
	return primary, files, nil
}

type licenseFile struct{ name, text string }

func isPrimary(name string) bool {
	switch strings.ToUpper(strings.TrimSuffix(strings.TrimSuffix(name, ".md"), ".txt")) {
	case "LICENSE", "LICENCE", "COPYING", "UNLICENSE":
		return true
	}
	return false
}

// classify names a licence from its text (best effort; "unknown" must be reviewed).
func classify(text string) string {
	t := strings.ToLower(text)
	has := func(s string) bool { return strings.Contains(t, s) }
	switch {
	case has("zero-clause bsd"):
		return "0BSD"
	case has("gnu affero"):
		return "AGPL"
	case has("gnu lesser general public"):
		return "LGPL"
	case has("gnu general public"):
		return "GPL"
	case has("mozilla public license"):
		return "MPL-2.0 (review)"
	case has("apache license") && has("version 2.0"):
		return "Apache-2.0"
	case has("permission is hereby granted, free of charge"):
		return "MIT"
	case has("redistribution and use in source and binary forms"):
		if has("neither the name") || has("may be used to endorse") {
			return "BSD-3-Clause"
		}
		return "BSD-2-Clause"
	case has("permission to use, copy, modify, and/or distribute"):
		return "ISC"
	case has("free and unencumbered software released into the public domain"):
		return "Unlicense"
	}
	return "unknown"
}

func permissive(class string) bool {
	switch class {
	case "MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC", "0BSD", "Unlicense":
		return true
	}
	return false
}

func fence(text string) string { return "~~~~\n" + text + "\n~~~~\n" }

// thirdPartyDoc renders THIRD_PARTY_NOTICES.md.
func thirdPartyDoc(root string, c projectCopyright) (string, error) {
	var b strings.Builder
	b.WriteString("# Third-party notices\n\n" +
		"GENERATED by `scripts/docgen` from the licence files of the Go modules compiled into the\n" +
		"Windows, Linux and macOS binaries (`go list -deps`). Do not edit by hand; run `make docs`.\n\n" +
		"Lumen itself is " + c.String() + ", licensed under the " + projectLicense + " (see `LICENSE` and\n" +
		"`NOTICE`). It includes the following third-party software under their own licences.\n\n## Go modules (compiled into the binary)\n\n")

	mods, err := goModules(root)
	if err != nil {
		return "", err
	}
	for _, m := range mods {
		primary, files, err := licenseTexts(m.dir)
		if err != nil {
			return "", err
		}
		if len(files) == 0 {
			return "", fmt.Errorf("module %s %s has no licence file: review its licence before shipping", m.path, m.version)
		}
		fmt.Fprintf(&b, "### %s %s (%s)\n\n", m.path, m.version, classify(primary))
		for _, f := range files {
			if len(files) > 1 {
				fmt.Fprintf(&b, "`%s`:\n\n", f.name)
			}
			b.WriteString(fence(f.text))
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

// debianCopyright renders packaging/copyright (DEP-5) for the deb package.
func debianCopyright(root string, c projectCopyright) (string, error) {
	_ = root // the third-party licence texts live in THIRD_PARTY_NOTICES.md
	return `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: lumen
Source: https://github.com/evobytes/lumen
Comment: GENERATED by scripts/docgen (do not edit; run "make docs"). Go modules compiled into
 the binary are listed with their licences in /usr/share/doc/lumen/THIRD_PARTY_NOTICES.md.

Files: *
Copyright: ` + c.year + ` ` + c.holder + `
License: ` + projectLicense + `

License: ` + projectLicense + `
 On Debian systems, the full text of the Apache License, Version 2.0 can be found
 in /usr/share/common-licenses/Apache-2.0.
`, nil
}

// lintLicense checks the project's own licensing is in place and every
// dependency is permissively licensed.
func lintLicense(root string) []string {
	var problems []string
	add := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // repo-relative
		if err != nil {
			add("%s: %v", rel, err)
		}
		return string(b)
	}

	if lic := read("LICENSE"); !strings.Contains(lic, "Apache License") || !strings.Contains(lic, "Version 2.0, January 2004") {
		add("LICENSE is not the Apache License, Version 2.0")
	}
	// NOTICE and the README block are generated from COPYRIGHT (the stale check
	// covers them); here we only require that COPYRIGHT itself parses.
	if _, err := loadCopyright(root); err != nil {
		add("%v", err)
	}
	if !strings.Contains(read("README.md"), projectLicense) {
		add("README.md does not state the licence (%s)", projectLicense)
	}

	mods, err := goModules(root)
	if err != nil {
		return append(problems, err.Error())
	}
	for _, m := range mods {
		primary, files, err := licenseTexts(m.dir)
		switch {
		case err != nil:
			add("%s: %v", m.path, err)
		case len(files) == 0 || primary == "":
			add("dependency %s %s has no recognizable LICENSE file: review before shipping", m.path, m.version)
		default:
			if c := classify(primary); !permissive(c) {
				add("dependency %s %s is licensed %s: not on the permissive allow-list (MIT, Apache-2.0, BSD, ISC, 0BSD, Unlicense); review it", m.path, m.version, c)
			}
		}
	}
	return problems
}
