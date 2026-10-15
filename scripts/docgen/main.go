// Command docgen keeps Lumen's documentation current.
//
//	docgen generate   rewrite generated docs from their sources; prints changed paths
//	docgen check      fail if generated docs are stale, or lint finds problems (CI)
//	docgen lint       link/path/target/env/frontmatter/version consistency checks
//	docgen staged     commit-time rule: code staged without any hand-written doc
//	docgen bundle F   write a reproducible zip of the documentation to F
//
// Generated: config reference (from internal/config struct tags), the shipped example
// config files, the API reference (docs/api.md, from the route table in internal/api),
// make targets (Makefile "##" comments), the skills index (SKILL.md frontmatter), the
// package table (go doc comments), docs/versions.md (pins) and the copyright statement
// (COPYRIGHT -> NOTICE, THIRD_PARTY_NOTICES.md, packaging/copyright, README).
// Wired into .githooks/pre-commit (regenerate and stage) and `make docs-check` (CI).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: docgen generate|check|lint|staged|bundle <out.zip>")
		os.Exit(2)
	}
	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}
	switch os.Args[1] {
	case "generate":
		changed, err := regenerate(root, true)
		if err != nil {
			fatal(err)
		}
		for _, p := range changed {
			fmt.Println(p) // stdout is the list the pre-commit hook stages
			fmt.Fprintln(os.Stderr, "docgen: updated", p)
		}
	case "check":
		stale, err := regenerate(root, false)
		if err != nil {
			fatal(err)
		}
		problems, err := lint(root)
		if err != nil {
			fatal(err)
		}
		for _, p := range stale {
			problems = append(problems, p+" is out of date: run `make docs`")
		}
		report(problems)
	case "lint":
		problems, err := lint(root)
		if err != nil {
			fatal(err)
		}
		report(problems)
	case "bundle":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: docgen bundle <out.zip>")
			os.Exit(2)
		}
		if err := bundle(root, os.Args[2]); err != nil {
			fatal(err)
		}
	case "staged":
		files, err := stagedFiles(root)
		if err != nil {
			fatal(err)
		}
		report(couplingProblems(files, os.Getenv("DOCS_REVIEWED") == "1"))
	default:
		fmt.Fprintln(os.Stderr, "usage: docgen generate|check|lint|staged|bundle <out.zip>")
		os.Exit(2)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "docgen:", err)
	os.Exit(1)
}

func report(problems []string) {
	if len(problems) == 0 {
		return
	}
	sort.Strings(problems)
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "docs:", p)
	}
	fmt.Fprintf(os.Stderr, "docs: %d problem(s)\n", len(problems))
	os.Exit(1)
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

func stagedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACMRD") //nolint:gosec,noctx // fixed arguments
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached: %w", err)
	}
	var files []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}
