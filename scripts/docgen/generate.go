package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/evobytes/lumen/internal/api"
	"github.com/evobytes/lumen/internal/config"
)

const modulePath = "github.com/evobytes/lumen"

// block is a marker-delimited region inside a hand-written file.
type block struct{ file, name, body string }

// regenerate renders every generated doc. With write it updates files on disk;
// either way it returns the paths whose content differs from what is on disk.
func regenerate(root string, write bool) ([]string, error) {
	whole, blocks, err := render(root)
	if err != nil {
		return nil, err
	}
	var changed []string
	touch := func(rel, content string) error {
		path := filepath.Join(root, rel)
		old, err := os.ReadFile(path) //nolint:gosec // repo-relative generated path
		if err == nil && string(old) == content {
			return nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		changed = append(changed, rel)
		if !write {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(content), 0o644) //nolint:gosec // docs are world-readable
	}

	for rel, content := range whole {
		if err := touch(rel, content); err != nil {
			return nil, err
		}
	}
	byFile := map[string][]block{}
	for _, b := range blocks {
		byFile[b.file] = append(byFile[b.file], b)
	}
	for rel, bs := range byFile {
		data, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // repo-relative
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		content := string(data)
		for _, b := range bs {
			if content, err = replaceBlock(content, b.name, b.body); err != nil {
				return nil, fmt.Errorf("%s: %w", rel, err)
			}
		}
		if err := touch(rel, content); err != nil {
			return nil, err
		}
	}
	sort.Strings(changed)
	return changed, nil
}

func markers(name string) (begin, end string) {
	return "<!-- BEGIN GENERATED: " + name + " (scripts/docgen; edit the source, run `make docs`) -->",
		"<!-- END GENERATED: " + name + " -->"
}

// replaceBlock swaps the body between the named markers.
func replaceBlock(content, name, body string) (string, error) {
	begin, end := markers(name)
	i := strings.Index(content, begin)
	j := strings.Index(content, end)
	if i < 0 || j < 0 || j < i {
		return "", fmt.Errorf("missing generated block %q (need %s … %s)", name, begin, end)
	}
	return content[:i+len(begin)] + "\n" + strings.TrimRight(body, "\n") + "\n" + content[j:], nil
}

func render(root string) (whole map[string]string, blocks []block, err error) {
	whole = map[string]string{
		shippedConfigLinux:   configFile("linux"),
		shippedConfigWindows: configFile("windows"),
		"docs/api.md":        apiDoc(),
	}
	if whole["docs/versions.md"], err = versionsDoc(root); err != nil {
		return nil, nil, err
	}
	copyright, err := loadCopyright(root)
	if err != nil {
		return nil, nil, err
	}
	whole["NOTICE"] = noticeFile(copyright)
	if whole["packaging/copyright"], err = debianCopyright(root, copyright); err != nil {
		return nil, nil, err
	}
	if whole["THIRD_PARTY_NOTICES.md"], err = thirdPartyDoc(root, copyright); err != nil {
		return nil, nil, err
	}

	targets, err := makeTargets(root)
	if err != nil {
		return nil, nil, err
	}
	skills, err := loadSkills(root)
	if err != nil {
		return nil, nil, err
	}
	pkgs, err := packageTable(root)
	if err != nil {
		return nil, nil, err
	}
	cfg := configTable()

	blocks = []block{
		{"AGENTS.md", "make-targets", targetTable(targets)},
		{"AGENTS.md", "skills-index", skillTable(skills)},
		{"README.md", "make-targets", targetTable(targets)},
		{"README.md", "copyright", copyright.String() + "."},
		{"docs/configuration.md", "config-table", cfg},
		{".claude/skills/config-logging/SKILL.md", "config-table", cfg},
		{"docs/architecture.md", "packages", pkgs},
	}
	return whole, blocks, nil
}

// The shipped example config files, generated per platform (the defaults differ).
const (
	shippedConfigLinux   = "os/usr/share/lumen/config.example.toml"
	shippedConfigWindows = "packaging/windows/config.example.toml"
)

// ---- make targets --------------------------------------------------------

type makeTarget struct{ name, desc string }

var (
	targetLine = regexp.MustCompile(`^([a-zA-Z0-9_.-]+):[^#=]*##\s*(.+)$`)
	phonyLine  = regexp.MustCompile(`^\.PHONY:\s*(.+)$`)
)

// makeTargets reads "target: deps ## description" lines in Makefile order and
// insists every .PHONY target has one: undocumented targets are a lint failure.
func makeTargets(root string) ([]makeTarget, error) {
	f, err := os.Open(filepath.Join(root, "Makefile")) //nolint:gosec // fixed path
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []makeTarget
	described := map[string]bool{}
	var phony []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if m := targetLine.FindStringSubmatch(line); m != nil {
			out = append(out, makeTarget{m[1], strings.TrimSpace(m[2])})
			described[m[1]] = true
		}
		if m := phonyLine.FindStringSubmatch(line); m != nil {
			phony = append(phony, strings.Fields(m[1])...)
		}
	}
	for _, p := range phony {
		if !described[p] {
			return nil, fmt.Errorf("target %q in the Makefile has no `## description` (add one on its rule line)", p)
		}
	}
	return out, sc.Err()
}

func targetTable(ts []makeTarget) string {
	var b strings.Builder
	b.WriteString("| Command | Does |\n|---|---|\n")
	for _, t := range ts {
		if t.name == "help" {
			continue
		}
		fmt.Fprintf(&b, "| `make %s` | %s |\n", t.name, t.desc)
	}
	b.WriteString("| `make help` | list targets |\n")
	return b.String()
}

// ---- skills --------------------------------------------------------------

type skill struct{ name, desc, dir string }

func loadSkills(root string) ([]skill, error) {
	dirs, err := filepath.Glob(filepath.Join(root, ".claude", "skills", "*", "SKILL.md"))
	if err != nil {
		return nil, err
	}
	var out []skill
	for _, p := range dirs {
		data, err := os.ReadFile(p) //nolint:gosec // globbed repo path
		if err != nil {
			return nil, err
		}
		name, desc := frontmatter(string(data))
		out = append(out, skill{name: name, desc: desc, dir: filepath.Base(filepath.Dir(p))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].dir < out[j].dir })
	return out, nil
}

// frontmatter returns name and description from a leading --- block (single-line values).
func frontmatter(s string) (name, desc string) {
	if !strings.HasPrefix(s, "---\n") {
		return "", ""
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return "", ""
	}
	for _, l := range strings.Split(s[4:4+end], "\n") {
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "name":
			name = strings.TrimSpace(v)
		case "description":
			desc = strings.TrimSpace(v)
		}
	}
	return name, desc
}

func skillTable(ss []skill) string {
	var b strings.Builder
	b.WriteString("| Skill (also a `/slash` command) | What and when |\n|---|---|\n")
	for _, s := range ss {
		fmt.Fprintf(&b, "| `/%s` | %s |\n", s.dir, strings.ReplaceAll(s.desc, "|", "\\|"))
	}
	return b.String()
}

// ---- config --------------------------------------------------------------

type keyInfo struct{ section, key, def, doc, env string }

// configKeys lists every config key with its default for goos. Windows is the
// primary platform, so its defaults are what the reference docs show.
func configKeys(goos string) []keyInfo {
	def := config.DefaultFor(goos)
	rv := reflect.ValueOf(def)
	rt := rv.Type()
	var out []keyInfo
	for i := 0; i < rt.NumField(); i++ {
		sec := rt.Field(i)
		section := sec.Tag.Get("toml")
		sv := rv.Field(i)
		for j := 0; j < sv.NumField(); j++ {
			f := sv.Type().Field(j)
			key := f.Tag.Get("toml")
			out = append(out, keyInfo{
				section: section, key: key,
				def: tomlValue(sv.Field(j)),
				doc: f.Tag.Get("doc"),
				env: config.EnvPrefix + strings.ToUpper(section) + "_" + strings.ToUpper(key),
			})
		}
	}
	return out
}

// tomlValue renders a config value as a TOML literal.
func tomlValue(v reflect.Value) string {
	switch x := v.Interface().(type) {
	case config.Duration:
		return strconv.Quote(compactDuration(x.Std()))
	case string:
		return strconv.Quote(x)
	case bool:
		return strconv.FormatBool(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case []string:
		q := make([]string, len(x))
		for i, s := range x {
			q[i] = strconv.Quote(s)
		}
		return "[" + strings.Join(q, ", ") + "]"
	}
	panic(fmt.Sprintf("docgen: unsupported config field type %s: teach tomlValue about it", v.Type()))
}

func compactDuration(d time.Duration) string {
	s := d.String() // 1m0s -> 1m, 1h0m0s -> 1h
	if strings.HasSuffix(s, "m0s") {
		s = strings.TrimSuffix(s, "0s")
	}
	if strings.HasSuffix(s, "h0m") {
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}

func configTable() string {
	var b strings.Builder
	b.WriteString("| Key | Environment | Default (Windows) | Description |\n|---|---|---|---|\n")
	for _, k := range configKeys("windows") {
		def := "`" + k.def + "`"
		if k.def == `""` {
			def = "_(empty)_"
		}
		fmt.Fprintf(&b, "| `%s.%s` | `%s` | %s | %s |\n", k.section, k.key, k.env, def, k.doc)
	}
	return b.String()
}

// configFile renders the shipped example config for goos: every key commented
// out at its default, preceded by its description.
func configFile(goos string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `# Lumen configuration example (%s defaults).
# GENERATED by scripts/docgen from internal/config. Do not edit by hand:
# change the struct tags in internal/config/config.go and run "make docs".
#
# Every value shown is the built-in default; uncomment a line to change it.
# Copy this file to the per-user location (the -config flag overrides it):
#   Windows  %%APPDATA%%\\Lumen\\config.toml
#   Linux    ~/.config/Lumen/config.toml
# Precedence: defaults < this file < %s* environment < flags.
# Each key has an environment variable, %s<SECTION>_<KEY> (%sSERVER_LISTEN).
# Keep secrets out of this file; prefer the environment for anything sensitive.
`, goos, config.EnvPrefix, config.EnvPrefix, config.EnvPrefix)
	section := ""
	for _, k := range configKeys(goos) {
		if k.section != section {
			section = k.section
			fmt.Fprintf(&b, "\n[%s]\n", section)
		}
		for _, l := range wrap(k.doc, 74) {
			fmt.Fprintf(&b, "# %s\n", l)
		}
		fmt.Fprintf(&b, "# %s = %s\n", k.key, k.def)
	}
	return b.String()
}

// apiDoc renders docs/api.md from the route table the router is built from.
func apiDoc() string {
	var b strings.Builder
	b.WriteString("# Web API reference\n\n" +
		"GENERATED by `scripts/docgen` from the route table in `internal/api` (`api.Routes`). Do not\n" +
		"edit by hand: change the route's summary in the code and run `make docs`.\n\n" +
		"The server listens on `server.listen` (default `127.0.0.1:8765`). Routes under `/api/v1` need the\n" +
		"`auth.token` secret as `X-API-Token` or `Authorization: Bearer` when one is configured (the event\n" +
		"stream also accepts `?token=`, because browsers cannot set headers on `EventSource`). Errors are\n" +
		"`application/problem+json`. Request bodies are bounded JSON and unknown fields are rejected.\n\n" +
		"| Method | Path | Auth | Description |\n|---|---|---|---|\n")
	for _, r := range api.Routes() {
		auth := "no"
		if r.Auth {
			auth = "token"
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n", r.Method, r.Path, auth, strings.ReplaceAll(r.Summary, "|", "\\|"))
	}
	b.WriteString("\n```sh\n" +
		"curl -X POST http://127.0.0.1:8765/api/v1/next\n" +
		"curl -X PUT  http://127.0.0.1:8765/api/v1/slides -H 'content-type: application/json' \\\n" +
		"     -d '{\"slides\":[{\"text\":\"Hello\"},{\"text\":\"World\"}]}'\n" +
		"curl -X PUT  http://127.0.0.1:8765/api/v1/output -d '{\"screen\":\"\\\\\\\\.\\\\DISPLAY2\"}'\n" +
		"```\n")
	return b.String()
}

func wrap(s string, width int) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		if cur != "" && len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
			continue
		}
		if cur != "" {
			cur += " "
		}
		cur += w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// ---- packages ------------------------------------------------------------

type goPkg struct {
	ImportPath string
	Doc        string
	Imports    []string
}

func packageTable(root string) (string, error) {
	cmd := exec.Command("go", "list", "-json", "./cmd/...", "./internal/...") //nolint:gosec,noctx // fixed arguments
	cmd.Dir = root
	// Always list for the shipped platform: build tags change the package set
	// (internal/proc/stopspike is windows-only), and the table must not depend
	// on which host ran `make docs`.
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list: %w: %s", err, stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var pkgs []goPkg
	for dec.More() {
		var p goPkg
		if err := dec.Decode(&p); err != nil {
			return "", err
		}
		pkgs = append(pkgs, p)
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ImportPath < pkgs[j].ImportPath })

	var b strings.Builder
	b.WriteString("| Package | Responsibility (its package doc) | Internal imports |\n|---|---|---|\n")
	for _, p := range pkgs {
		rel := strings.TrimPrefix(p.ImportPath, modulePath+"/")
		doc := p.Doc
		if doc == "" {
			return "", fmt.Errorf("package %s has no package doc comment (it feeds the generated package table)", rel)
		}
		for _, pre := range []string{"Package ", "Command "} {
			if after, ok := strings.CutPrefix(doc, pre); ok {
				_, rest, _ := strings.Cut(after, " ")
				doc = rest
			}
		}
		var deps []string
		for _, imp := range p.Imports {
			if strings.HasPrefix(imp, modulePath+"/internal/") {
				deps = append(deps, "`"+strings.TrimPrefix(imp, modulePath+"/internal/")+"`")
			}
		}
		d := "—"
		if len(deps) > 0 {
			d = strings.Join(deps, ", ")
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", rel, strings.TrimSuffix(doc, "."), d)
	}
	return b.String(), nil
}

// ---- versions ------------------------------------------------------------

var (
	reGoMod   = regexp.MustCompile(`(?m)^go\s+(\S+)`)
	reEnvLine = regexp.MustCompile(`(?m)^([A-Z][A-Z0-9_]*)=(\S+)`)
	reUses    = regexp.MustCompile(`uses:\s+([\w./-]+)@([0-9a-f]{40})\s+#\s+(\S+)`)
)

func versionsDoc(root string) (string, error) {
	read := func(rel string) (string, error) {
		b, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // repo-relative
		return string(b), err
	}
	var b strings.Builder
	b.WriteString("# Versions and pins\n\n" +
		"GENERATED by `scripts/docgen` from the files named below. Do not edit by hand: change the\n" +
		"source, run `make docs`. Dependabot does not track the tool pins; upgrade them deliberately.\n\n")

	gm, err := read("go.mod")
	if err != nil {
		return "", err
	}
	if m := reGoMod.FindStringSubmatch(gm); m != nil {
		fmt.Fprintf(&b, "## Go toolchain\n\n| Item | Version | Source |\n|---|---|---|\n| Go | %s | `go.mod` |\n\n", m[1])
	}

	tools, err := read("scripts/tools.env")
	if err != nil {
		return "", err
	}
	b.WriteString("## Dev tools (`make bootstrap`)\n\n| Tool | Pin | Source |\n|---|---|---|\n")
	for _, m := range reEnvLine.FindAllStringSubmatch(tools, -1) {
		fmt.Fprintf(&b, "| %s | `%s` | `scripts/tools.env` |\n", m[1], m[2])
	}

	wfs, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		return "", err
	}
	seen := map[string]string{}
	for _, w := range wfs {
		data, err := os.ReadFile(w) //nolint:gosec // globbed repo path
		if err != nil {
			return "", err
		}
		for _, m := range reUses.FindAllStringSubmatch(string(data), -1) {
			seen[m[1]] = m[3] + " (`" + m[2][:12] + "…`)"
		}
	}
	acts := make([]string, 0, len(seen))
	for a := range seen {
		acts = append(acts, a)
	}
	sort.Strings(acts)
	b.WriteString("\n## GitHub Actions (pinned by commit SHA)\n\n| Action | Version | Source |\n|---|---|---|\n")
	for _, a := range acts {
		fmt.Fprintf(&b, "| %s | %s | `.github/workflows/` |\n", a, seen[a])
	}
	return b.String(), nil
}
