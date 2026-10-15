# How documentation is kept current

Documentation drifts unless something makes drift fail. What exists, what is generated, and what enforces it.

## Kinds of docs

| Doc | Kind | Source of truth |
|---|---|---|
| `AGENTS.md` (rules, layout), `README.md` (prose) | hand-written | people |
| `AGENTS.md` / `README.md` make-target tables | **generated** | `##` comments in the `Makefile` |
| `AGENTS.md` skills index | **generated** | `.claude/skills/*/SKILL.md` frontmatter |
| `docs/configuration.md`, config table in the `config-logging` skill | **generated** | `toml` / `doc` tags in `internal/config/config.go` |
| `os/usr/share/lumen/config.example.toml`, `packaging/windows/config.example.toml` | **generated** | the same struct tags (per-OS defaults) |
| `docs/api.md` | **generated** | the route table (`api.Routes`) that the router is built from |
| package table in `docs/architecture.md` | **generated** | each package's doc comment |
| `COPYRIGHT` | hand-written | the one line `Copyright YYYY Holder` that is the **single source** for year and holder |
| `NOTICE`, `packaging/copyright` (Debian DEP-5), `THIRD_PARTY_NOTICES.md`, copyright block in `README.md` | **generated** | `COPYRIGHT` plus the licence files of the modules in the Windows, Linux and macOS builds |
| `LICENSE` | hand-written | the canonical Apache-2.0 text; lint checks it is Apache-2.0 |
| `docs/versions.md` | **generated** | `go.mod`, `scripts/tools.env`, workflow action SHA pins |
| `docs/architecture.md` prose, `docs/intent.md`, `docs/roadmap.md`, `docs/windows-dev.md`, `docs/deployment.md`, `docs/decisions.md`, `docs/session-log.md`, `docs/research/*.md` | hand-written | people (decisions: append only) |
| `CHANGELOG.md` | hand-written | people (an entry for the current `VERSION` is required) |
| `.claude/skills/*/SKILL.md` | hand-written | people; also the slash-command playbooks |
| Go doc comments | hand-written | lint (`revive: exported`) requires them on exported names |
| assistant memory (outside the repo) | hand-maintained | `/session-end`; never the only home of a fact |

Generated regions sit between `<!-- BEGIN GENERATED: name ... -->` and `<!-- END GENERATED: name -->`, or are whole files that say
so in their first lines. Never edit them: change the source and run `make docs`.

## What enforces it

1. **pre-commit hook** (`.githooks/pre-commit`; install once per clone with `make hooks`; `make bootstrap` does it too):
   *coupling* (behavior files staged with no hand-written doc staged stops the commit; `DOCS_REVIEWED=1` overrides, say why in the
   message), *generate* (regenerate and stage generated docs), *lint* (broken links, code-span paths that no longer exist, `make`
   targets that do not exist, `LUMEN_*` variables that are not config keys, skill frontmatter, a `CHANGELOG.md` entry for `VERSION`,
   sequential decision numbers, licences).
2. **CI backstop**: `make ci` runs `make docs-check`. `git commit --no-verify` cannot get past CI.
3. **Tests**: `internal/config` fails if a key lacks a `doc` tag or an environment mapping; `scripts/docgen` tests cover the
   generators and the lint.

## Commands

| Command | Does |
|---|---|
| `make docs` | regenerate generated docs |
| `make docs-check` | stale-docs check plus lint (what CI runs) |
| `make docs-bundle` | regenerate, check, then write a byte-reproducible docs zip to `dist/` (`scripts/build-docs`) |
| `make hooks` | install the pre-commit hook |
| `/sync-docs` | agent playbook: regenerate, lint, update the hand-written docs a change affects |
| `/session-end` | agent playbook: persist decisions, docs and memory before a session closes |

## Extending

A new generated doc: add a renderer and, for a block, a marker in `scripts/docgen/generate.go`; a test in
`scripts/docgen/docgen_test.go`; a row in the table above. A new lint rule: `scripts/docgen/lint.go` plus a test. The hook follows
the convention of `hive-nmos-capture`, `tips-go` and `hive-cli-tools`.
