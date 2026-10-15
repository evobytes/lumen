package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reLink     = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
	reCodeSpan = regexp.MustCompile("`([^`\n]+)`")
	reEnv      = regexp.MustCompile(`\bLUMEN_[A-Z0-9_]+`)
	reDecision = regexp.MustCompile(`(?m)^## (\d{3}) — `)
	reLineSuf  = regexp.MustCompile(`:\d+(-\d+)?$`)
)

// repoRoots are the first path segments docs may reference; anything else in a
// code span (a path relative to some package, a system path) is not checked.
var repoRoots = []string{"internal/", "cmd/", "scripts/", "os/", "packaging/", "docs/", ".claude/", ".github/", "dev/", "spikes/"}

var rootFiles = map[string]bool{
	"AGENTS.md": true, "README.md": true, "CHANGELOG.md": true, "Makefile": true, "VERSION": true,
	"go.mod": true, "go.sum": true, ".golangci.yml": true, "COPYRIGHT": true, "NOTICE": true, "LICENSE": true,
	"THIRD_PARTY_NOTICES.md": true,
}

// generatedOrIgnored are paths docs may name that are not tracked (build output, caches).
// nonConfigEnv are LUMEN_* variables that are not config keys (they gate scripts), so docs may name them.
var nonConfigEnv = map[string]bool{"LUMEN_ALLOW_HOST_WRITE": true}

var generatedOrIgnored = []string{"dist/", ".tools/", ".cache/", ".ref/", "dev/data"}

func markdownFiles(root string) ([]string, error) {
	out := []string{"AGENTS.md", "README.md", "CHANGELOG.md"}
	for _, pattern := range []string{"docs/*.md", "docs/research/*.md", ".claude/skills/*/SKILL.md"} {
		m, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		for _, p := range m {
			rel, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(rel))
		}
	}
	return out, nil
}

// lint runs the checks generation cannot make true by construction.
func lint(root string) ([]string, error) {
	problems, err := lintReferences(root)
	if err != nil {
		return nil, err
	}
	problems = append(problems, lintLicense(root)...)
	problems = append(problems, lintSkills(root)...)
	problems = append(problems, lintVersions(root)...)
	return problems, nil
}

// lintReferences checks that what the docs name still exists: links, code-span
// paths, make targets and environment variables.
func lintReferences(root string) ([]string, error) {
	var problems []string
	add := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }

	targets, err := makeTargets(root)
	if err != nil {
		return nil, err
	}
	targetSet := map[string]bool{}
	for _, t := range targets {
		targetSet[t.name] = true
	}
	envOK := map[string]bool{}
	for _, k := range configKeys("windows") {
		envOK[k.env] = true
	}

	files, err := markdownFiles(root)
	if err != nil {
		return nil, err
	}
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // repo-relative
		if err != nil {
			add("%s: %v", rel, err)
			continue
		}
		text := string(data)

		for _, m := range reLink.FindAllStringSubmatch(text, -1) {
			t := m[1]
			if strings.Contains(t, "://") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "mailto:") {
				continue
			}
			t, _, _ = strings.Cut(t, "#")
			if _, err := os.Stat(filepath.Join(root, filepath.Dir(rel), t)); err != nil { //nolint:gosec // doc-derived path, only stat'ed inside a dev tool
				add("%s: broken link %q", rel, m[1])
			}
		}

		for _, m := range reCodeSpan.FindAllStringSubmatch(text, -1) {
			span := m[1]
			if p, ok := repoPath(span); ok {
				if _, err := os.Stat(filepath.Join(root, p)); err != nil { //nolint:gosec // doc-derived path, only stat'ed inside a dev tool
					add("%s: `%s` does not exist (renamed or removed? update the doc)", rel, span)
				}
			}
			if rest, ok := strings.CutPrefix(span, "make "); ok {
				for _, w := range strings.Fields(rest) {
					if strings.ContainsAny(w, "=-<[|") {
						break
					}
					if !targetSet[w] {
						add("%s: `%s` names make target %q which does not exist", rel, span, w)
						break
					}
				}
			}
		}

		for _, e := range reEnv.FindAllString(text, -1) {
			if strings.HasSuffix(e, "_") || envOK[e] || nonConfigEnv[e] {
				continue
			}
			add("%s: environment variable %s is not a known config key", rel, e)
		}
	}

	return problems, nil
}

// repoPath reports whether a code span is a repo-relative path worth checking.
func repoPath(span string) (string, bool) {
	if strings.ContainsAny(span, " *<>{}$|") || strings.Contains(span, "...") || strings.Contains(span, "…") {
		return "", false
	}
	span = reLineSuf.ReplaceAllString(span, "")
	span = strings.TrimSuffix(span, "/")
	if rootFiles[span] {
		return span, true
	}
	for _, g := range generatedOrIgnored {
		if strings.HasPrefix(span, g) {
			return "", false
		}
	}
	for _, r := range repoRoots {
		if strings.HasPrefix(span, r) {
			return span, true
		}
	}
	return "", false
}

func lintSkills(root string) []string {
	var problems []string
	skills, err := loadSkills(root)
	if err != nil {
		return []string{err.Error()}
	}
	for _, s := range skills {
		switch {
		case s.name == "":
			problems = append(problems, fmt.Sprintf(".claude/skills/%s/SKILL.md: missing frontmatter name", s.dir))
		case s.name != s.dir:
			problems = append(problems, fmt.Sprintf(".claude/skills/%s/SKILL.md: name %q must equal the directory name", s.dir, s.name))
		}
		if len(s.desc) < 20 {
			problems = append(problems, fmt.Sprintf(".claude/skills/%s/SKILL.md: description is missing or too short to guide when to use it", s.dir))
		}
	}
	return problems
}

// lintVersions checks the changelog and decision log are sane.
func lintVersions(root string) []string {
	var problems []string
	add := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // repo-relative
		if err != nil {
			add("%s: %v", rel, err)
		}
		return string(b)
	}

	version := strings.TrimSpace(read("VERSION"))
	if !strings.Contains(read("CHANGELOG.md"), "## ["+version+"]") {
		add("CHANGELOG.md has no `## [%s]` entry for the current VERSION", version)
	}

	prev := 0
	for _, m := range reDecision.FindAllStringSubmatch(read("docs/decisions.md"), -1) {
		var n int
		_, _ = fmt.Sscanf(m[1], "%d", &n)
		if n != prev+1 {
			add("docs/decisions.md: decision %03d follows %03d (numbers must be sequential and unique)", n, prev)
		}
		prev = n
	}
	return problems
}

// ---- commit-time coupling ---------------------------------------------------

// handDocs are the documents people maintain; generated docs are excluded on
// purpose (the hook regenerates them, so they would always look "touched").
func isHandDoc(f string) bool {
	switch f {
	case "README.md", "AGENTS.md", "CHANGELOG.md":
		return true
	case "docs/configuration.md", "docs/versions.md", "docs/api.md":
		return false // generated
	}
	if strings.HasPrefix(f, "docs/") && strings.HasSuffix(f, ".md") {
		return true
	}
	return strings.HasPrefix(f, ".claude/skills/") && strings.HasSuffix(f, "/SKILL.md")
}

// isBehavior reports files whose change should be reflected in documentation.
func isBehavior(f string) bool {
	if strings.HasSuffix(f, "_test.go") {
		return false
	}
	switch f {
	case "Makefile", "VERSION", "go.mod":
		return true
	}
	for _, p := range []string{"internal/", "cmd/", "scripts/", "os/", "packaging/", ".github/"} {
		if strings.HasPrefix(f, p) {
			return true
		}
	}
	return false
}

func couplingProblems(staged []string, reviewed bool) []string {
	var behavior []string
	docs := map[string]bool{}
	for _, f := range staged {
		if isHandDoc(f) {
			docs[f] = true
		}
		if isBehavior(f) {
			behavior = append(behavior, f)
		}
	}
	if reviewed {
		return nil
	}
	var problems []string
	if len(behavior) > 0 && len(docs) == 0 {
		shown := behavior
		if len(shown) > 5 {
			shown = append(append([]string{}, shown[:5]...), fmt.Sprintf("… and %d more", len(behavior)-5))
		}
		problems = append(problems, fmt.Sprintf(
			"behavior changed (%s) but no hand-written doc is staged. Update the docs that apply "+
				"(CHANGELOG.md, AGENTS.md, README.md, docs/decisions.md, docs/architecture.md, or a .claude/skills/*/SKILL.md), "+
				"or, if nothing needs to change, commit with DOCS_REVIEWED=1",
			strings.Join(shown, ", ")))
	}
	return problems
}
