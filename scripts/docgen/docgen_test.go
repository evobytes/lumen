package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evobytes/lumen/internal/api"
)

func TestReplaceBlock(t *testing.T) {
	t.Parallel()
	begin, end := markers("x")
	in := "head\n" + begin + "\nold\n" + end + "\ntail\n"

	got, err := replaceBlock(in, "x", "new line\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "head\n" + begin + "\nnew line\n" + end + "\ntail\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	again, _ := replaceBlock(got, "x", "new line")
	if again != got {
		t.Error("replaceBlock must be idempotent")
	}
	if _, err := replaceBlock("no markers here", "x", "b"); err == nil {
		t.Error("missing markers must be an error")
	}
	if _, err := replaceBlock(end+"\n"+begin, "x", "b"); err == nil {
		t.Error("reversed markers must be an error")
	}
}

func TestFrontmatter(t *testing.T) {
	t.Parallel()
	name, desc := frontmatter("---\nname: foo\ndescription: Does a thing. Use when x.\n---\n# Body\nname: not this\n")
	if name != "foo" || desc != "Does a thing. Use when x." {
		t.Errorf("got %q %q", name, desc)
	}
	if n, d := frontmatter("# no frontmatter"); n != "" || d != "" {
		t.Errorf("got %q %q for missing frontmatter", n, d)
	}
}

func TestCompactDuration(t *testing.T) {
	t.Parallel()
	for in, want := range map[time.Duration]string{
		5 * time.Second: "5s", 15 * time.Second: "15s", time.Minute: "1m",
		2 * time.Minute: "2m", 90 * time.Second: "1m30s", time.Hour: "1h",
	} {
		if got := compactDuration(in); got != want {
			t.Errorf("compactDuration(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestWrap(t *testing.T) {
	t.Parallel()
	lines := wrap("one two three four five", 10)
	for _, l := range lines {
		if len(l) > 10 {
			t.Errorf("line %q exceeds width", l)
		}
	}
	if strings.Join(lines, " ") != "one two three four five" {
		t.Errorf("wrap lost words: %v", lines)
	}
}

func TestRepoPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		span string
		want bool
	}{
		{"internal/config/config.go", true},
		{"internal/config/config.go:42", true},
		{"scripts/version", true},
		{"os/usr/share/lumen/", true},
		{"Makefile", true},
		{"AGENTS.md", true},
		{"dist/bin/lumen", false}, // build output
		{".tools/bin", false},     // ignored
		{"dev/data/x", false},     // ignored
		{"internal/**/*.go", false},
		{`C:\Users\me\AppData\Roaming\Lumen\config.toml`, false}, // target-host path
		{"static/css/app.css", false},                            // package-relative, not a repo root
		{"make ci", false},
		{"internal/<pkg>/x.go", false},
	}
	for _, tc := range tests {
		if _, ok := repoPath(tc.span); ok != tc.want {
			t.Errorf("repoPath(%q) = %v, want %v", tc.span, ok, tc.want)
		}
	}
}

func TestCouplingProblems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		staged   []string
		reviewed bool
		wantSub  string // empty = no problem
	}{
		{"docs only", []string{"README.md"}, false, ""},
		{"tests only", []string{"internal/auth/auth_test.go", "internal/bpf/bpf_test.go"}, false, ""},
		{"code with hand doc", []string{"internal/auth/auth.go", "CHANGELOG.md"}, false, ""},
		{"code with skill", []string{"internal/auth/auth.go", ".claude/skills/security/SKILL.md"}, false, ""},
		{"code without docs", []string{"internal/auth/auth.go"}, false, "no hand-written doc is staged"},
		{"makefile without docs", []string{"Makefile"}, false, "no hand-written doc is staged"},
		{"generated docs do not count", []string{"internal/auth/auth.go", "docs/versions.md", "docs/configuration.md"}, false, "no hand-written doc is staged"},
		{"hand-written docs/ page counts", []string{"internal/auth/auth.go", "docs/deployment.md"}, false, ""},
		{"reviewed bypass", []string{"internal/auth/auth.go"}, true, ""},
		{"windows packaging counts as behavior", []string{"packaging/windows/lumen.iss"}, false, "no hand-written doc is staged"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := strings.Join(couplingProblems(tc.staged, tc.reviewed), "\n")
			switch {
			case tc.wantSub == "" && got != "":
				t.Errorf("unexpected problems: %s", got)
			case tc.wantSub != "" && !strings.Contains(got, tc.wantSub):
				t.Errorf("problems %q missing %q", got, tc.wantSub)
			}
		})
	}
}

func TestMakeTargets(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mk := ".PHONY: a b\na: ## does a\n\t@true\n\n.PHONY: b\nb: a ## does b\n\t@true\nplain:\n\t@true\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mk), 0o600); err != nil {
		t.Fatal(err)
	}
	ts, err := makeTargets(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 || ts[0] != (makeTarget{"a", "does a"}) || ts[1].name != "b" {
		t.Errorf("targets = %+v", ts)
	}

	// A .PHONY target without a description must be rejected.
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(".PHONY: a c\na: ## does a\n\t@true\nc:\n\t@true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := makeTargets(dir); err == nil || !strings.Contains(err.Error(), `"c"`) {
		t.Errorf("want undocumented-target error naming c, got %v", err)
	}
}

func TestConfigFileRendersEveryKey(t *testing.T) {
	t.Parallel()
	out := configFile("windows")
	for _, k := range configKeys("windows") {
		if k.doc == "" {
			t.Errorf("%s.%s has no doc tag", k.section, k.key)
		}
		if !strings.Contains(out, "# "+k.key+" = ") {
			t.Errorf("shipped config missing key %s.%s", k.section, k.key)
		}
		if !strings.HasPrefix(k.env, "LUMEN_") {
			t.Errorf("%s.%s env %q lacks the LUMEN_ prefix", k.section, k.key, k.env)
		}
	}
	if !strings.Contains(out, `# placement = "win32"`) {
		t.Error("the Windows example config must carry the Windows placement default")
	}
	if lin := configFile("linux"); !strings.Contains(lin, `# placement = "toolkit"`) {
		t.Error("the Linux example config must carry the toolkit placement default")
	}
}

func TestLintFlagsBrokenReferences(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("Makefile", ".PHONY: ci\nci: ## gate\n\t@true\n")
	write("VERSION", "1.2.3\n")
	write("CHANGELOG.md", "## [1.2.3]\n")
	write("docs/decisions.md", "## 001 — a\n## 002 — b\n")
	write("AGENTS.md", "See [ok](README.md), [bad](missing.md), `internal/gone/file.go`, `make ci`, `make nope`, LUMEN_SERVER_LISTEN, LUMEN_BOGUS_KEY.\n")
	write("README.md", "ok\n")

	problems, err := lintReferences(root)
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(problems, "\n")
	for _, want := range []string{`broken link "missing.md"`, "internal/gone/file.go", `make target "nope"`, "LUMEN_BOGUS_KEY"} {
		if !strings.Contains(all, want) {
			t.Errorf("lint missed %q; got:\n%s", want, all)
		}
	}
	for _, bad := range []string{"README.md", "`make ci`", "LUMEN_SERVER_LISTEN"} {
		if strings.Contains(all, "broken link \"README.md\"") || strings.Contains(all, bad+" does not") {
			t.Errorf("lint wrongly flagged %q:\n%s", bad, all)
		}
	}
}

func TestLintVersionsAndDecisions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("VERSION", "2.0.0\n")
	write("CHANGELOG.md", "## [1.0.0]\n")                     // no 2.0.0
	write("docs/decisions.md", "## 001 — a\n## 003 — skip\n") // gap

	all := strings.Join(lintVersions(root), "\n")
	for _, want := range []string{"no `## [2.0.0]` entry", "decision 003 follows 001"} {
		if !strings.Contains(all, want) {
			t.Errorf("lintVersions missed %q; got:\n%s", want, all)
		}
	}
}

func TestClassify(t *testing.T) {
	t.Parallel()
	tests := []struct{ text, want string }{
		{"Permission is hereby granted, free of charge, to any person obtaining a copy", "MIT"},
		{"Redistribution and use in source and binary forms, with or without\n Neither the name of the copyright holder", "BSD-3-Clause"},
		{"Redistribution and use in source and binary forms, with or without modification, are permitted", "BSD-2-Clause"},
		{"Zero-Clause BSD\n=====\nPermission to use, copy, modify, and/or distribute this software", "0BSD"},
		{"Permission to use, copy, modify, and/or distribute this software for any purpose", "ISC"},
		{"Apache License\n Version 2.0, January 2004", "Apache-2.0"},
		{"GNU GENERAL PUBLIC LICENSE Version 3", "GPL"},
		{"GNU Affero General Public License", "AGPL"},
		{"GNU Lesser General Public License", "LGPL"},
		{"Mozilla Public License Version 2.0", "MPL-2.0 (review)"},
		{"This is free and unencumbered software released into the public domain.", "Unlicense"},
		{"some custom terms nobody has heard of", "unknown"},
	}
	for _, tc := range tests {
		if got := classify(tc.text); got != tc.want {
			t.Errorf("classify(%.40q...) = %q, want %q", tc.text, got, tc.want)
		}
	}
	for _, ok := range []string{"MIT", "Apache-2.0", "BSD-3-Clause", "BSD-2-Clause", "ISC", "0BSD", "Unlicense"} {
		if !permissive(ok) {
			t.Errorf("%s must be allowed", ok)
		}
	}
	for _, bad := range []string{"GPL", "AGPL", "LGPL", "MPL-2.0 (review)", "unknown"} {
		if permissive(bad) {
			t.Errorf("%s must not be on the allow-list", bad)
		}
	}
}

func TestIsPrimaryLicenseFile(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]bool{
		"LICENSE": true, "LICENSE.md": true, "LICENSE.txt": true, "COPYING": true, "LICENCE": true,
		"PATENTS": false, "LICENSE-3RD-PARTY.md": false, "NOTICE": false, "LICENSE-SQLITE": false,
	} {
		if got := isPrimary(name); got != want {
			t.Errorf("isPrimary(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestFence(t *testing.T) {
	t.Parallel()
	if got, want := fence("a\nb"), "~~~~\na\nb\n~~~~\n"; got != want {
		t.Errorf("fence = %q, want %q", got, want)
	}
}

func TestParseCopyright(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    projectCopyright
		wantErr bool
	}{
		{"Copyright 2026 Evobytes Consulting Ltd\n", projectCopyright{"2026", "Evobytes Consulting Ltd"}, false},
		{"Copyright 2026-2028 Evobytes Consulting Ltd", projectCopyright{"2026-2028", "Evobytes Consulting Ltd"}, false},
		{"", projectCopyright{}, true},
		{"Copyright Evobytes Consulting Ltd", projectCopyright{}, true},
		{"Copyright 26 Evobytes", projectCopyright{}, true},
		{"(c) 2026 Evobytes", projectCopyright{}, true},
		{"Copyright 2026 Evobytes\nCopyright 2026 Other", projectCopyright{}, true},
	}
	for _, tc := range tests {
		got, err := parseCopyright(tc.in)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("parseCopyright(%q) = %+v, %v; want %+v, err=%v", tc.in, got, err, tc.want, tc.wantErr)
		}
	}
}

func TestNoticeFileCarriesStatement(t *testing.T) {
	t.Parallel()
	c := projectCopyright{"2026-2027", "Acme Ltd"}
	n := noticeFile(c)
	if !strings.HasPrefix(n, "Lumen\nCopyright 2026-2027 Acme Ltd\n") {
		t.Errorf("NOTICE does not lead with the statement:\n%s", n)
	}
	if got := c.String(); got != "Copyright 2026-2027 Acme Ltd" {
		t.Errorf("String() = %q", got)
	}
	if !strings.Contains(n, "does not include or redistribute the NDI runtime") {
		t.Errorf("NOTICE must say the optional integrations are not redistributed:\n%s", n)
	}
}

func TestAPIDocListsEveryRoute(t *testing.T) {
	t.Parallel()
	out := apiDoc()
	for _, r := range api.Routes() {
		if !strings.Contains(out, "`"+r.Path+"`") {
			t.Errorf("docs/api.md lacks route %s %s", r.Method, r.Path)
		}
	}
	if !strings.Contains(out, "| `GET` | `/api/v1/status` | token |") {
		t.Errorf("auth column missing:\n%s", out)
	}
}

func TestDebianCopyright(t *testing.T) {
	t.Parallel()
	out, err := debianCopyright(t.TempDir(), projectCopyright{"2026", "Acme Ltd"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Upstream-Name: lumen", "Copyright: 2026 Acme Ltd", "License: Apache-2.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("copyright file lacks %q:\n%s", want, out)
		}
	}
}

func TestBundleIsReproducible(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"README.md", "CHANGELOG.md", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md"} {
		write(f, f)
	}
	write("docs/b.md", "b")
	write("docs/research/a.md", "a")
	write("docs/not-markdown.txt", "x")
	a, b := filepath.Join(t.TempDir(), "a.zip"), filepath.Join(t.TempDir(), "b.zip")
	if err := bundle(root, a); err != nil {
		t.Fatal(err)
	}
	if err := bundle(root, b); err != nil {
		t.Fatal(err)
	}
	da, _ := os.ReadFile(a)
	db, _ := os.ReadFile(b)
	if !bytes.Equal(da, db) || len(da) == 0 {
		t.Fatal("two bundles of the same sources must be byte-identical")
	}
	files, err := bundleFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(files, ","); !strings.Contains(got, "docs/research/a.md") || strings.Contains(got, ".txt") {
		t.Fatalf("bundle files = %s", got)
	}
}
