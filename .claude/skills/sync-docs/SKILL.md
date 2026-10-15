---
name: sync-docs
description: Bring every kind of documentation in lumen current: regenerate generated docs, run the staleness lint, and update the hand-written docs that a change affects. Use after any behavior, config, Makefile, skill, route or dependency change, or when the pre-commit hook or make docs-check complains.
---

# /sync-docs

Read `docs/documentation.md` for the map of doc types. Some docs are **generated** (never edit them; edit the source), the rest are hand-written and guarded by lint plus the commit-time coupling rule.

## 1. Regenerate and check
```
make docs         # rewrites generated docs; prints what changed
make docs-check   # fails on stale generated docs or broken references
```
Fix every `docs:` line it prints:
| Message | Fix |
|---|---|
| `X is out of date: run make docs` | run `make docs` |
| `` `path` does not exist `` | the doc names a renamed or removed file: update the doc |
| `` names make target "x" which does not exist `` | fix the doc or add the target (with a `##` description) |
| `environment variable LUMEN_<NAME> is not a known config key` | fix the doc, or add the key in `internal/config` (tags and `envKeys`); script-only variables go in `nonConfigEnv` in `scripts/docgen/lint.go` |
| `target "x" in the Makefile has no ## description` | add `## description` to the rule line |
| `package ... has no package doc comment` | add `// Package x ...` |
| `CHANGELOG.md has no ## [VERSION] entry` | add the entry |
| `decision NNN follows MMM` | decision numbers must be sequential and unique |
| `skill ... name must equal the directory name` / short description | fix the SKILL.md frontmatter |

## 2. Update hand-written docs for what you changed
| You changed | Update |
|---|---|
| a config key | its `doc` tag, then `make docs` |
| a route | its summary in the route table in `internal/api/api.go`, then `make docs` (`docs/api.md` follows) |
| a make target | its `## description`, then `make docs` |
| a skill | its SKILL.md, then `make docs` (the index follows) |
| a package | its package doc comment, then `make docs` (the architecture table follows) |
| a pin (tool, action, Wails) | the pin; `make docs` rewrites `docs/versions.md` |
| behavior, workflow or a decision | `CHANGELOG.md`; `docs/decisions.md` (new numbered entry); `docs/roadmap.md` if status changed; the relevant skill; `AGENTS.md` if a rule or the layout changed |

## 3. Rules
Never edit between `GENERATED` markers or the generated files (`AGENTS.md` lists them). The hook counts only hand-written docs toward "docs were updated"; if a change genuinely needs none, commit with `DOCS_REVIEWED=1` and say why. Before finishing, `make docs-check` is clean.
