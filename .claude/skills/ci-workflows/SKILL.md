---
name: ci-workflows
description: GitHub Actions workflows for lumen and the make-first rule: what each workflow does, how to add a step, how to reproduce locally. Use when editing .github/ or the Makefile.
---

# CI workflows

## Rule zero
Workflows contain **no logic**: checkout, set up Go, install tools, call `make <target>`, upload artifacts. Every step must be runnable locally through the same target. To add a check: add a `make` target (calling a `scripts/` file), add it to `make ci` if it is a gate, then call it from the workflow.

## Workflows
| File | Trigger | Jobs |
|---|---|---|
| `ci.yml` | PR, push `main` | `windows` (the primary): `make bootstrap`, `make ci CHECK_OS=host` (no race detector: no cgo). `linux`: apt GTK4/WebKitGTK 6, `make ci CHECK_OS=host` (race detector). `package` (needs both): Windows zip + Inno Setup installer, Linux deb, docs bundle, uploaded as artifacts. `macos` (experimental, `continue-on-error`): `make macos-app`. `selftest-windows` (experimental): `make selftest` on `windows-latest`, report uploaded |
| `release.yml` | tag `v*` | `check` (the gate on Windows and Linux, plus `make release-check TAG=...`), `build` (all packages), `publish` (SHA256SUMS, build-provenance attestation, `gh release create`) |
| `dependabot.yml` | schedule | gomod and github-actions, weekly |

## Conventions
- Pin third-party actions to a full commit SHA with a `# vX.Y.Z` comment. The pins were copied from `hive-nmos-capture`/`tips-go` and not re-verified; `docs/versions.md` lists them. `permissions:` minimal at the top (`contents: read`), elevated per job only.
- Go version from `go.mod` (`actions/setup-go` with `go-version-file`).
- Windows steps run with `shell: bash` (Git Bash); `choco install make` is needed on `windows-latest`; `choco install innosetup` for the installer job.
- Tools come from `make bootstrap` (the same as local), not ad hoc `go install` lines.
- Concurrency group per ref with `cancel-in-progress` for PRs. No secrets are needed; release uses `GITHUB_TOKEN` only.
- Artifact names carry the `<stamp>-<hash>` suffix from `scripts/version`.

## Known gaps (the workflows have never run on GitHub)
First run may need fixes: Git Bash path quirks, `xvfb` for a Linux self-test, the virtual single display on runners, `macos-latest` Xcode/CGO. Fix forward and record it in `docs/roadmap.md`.

## Reproducing CI locally
`make ci` is the `windows`/`linux` jobs. `make release-dry` is `release.yml` minus publishing. `act` is optional and never required.

## Makefile conventions
`.PHONY` all non-file targets; every target has a `## description` (the generated tables read it); `SHELL := bash` with `-euo pipefail`; variables come from `scripts/version` and `scripts/tools.env`; no duplication.
