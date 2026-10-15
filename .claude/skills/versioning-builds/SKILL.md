---
name: versioning-builds
description: Version scheme (x.x.x), build stamp (yyyymmdd-hhmmss-githash), artifact filenames, ldflags injection and reproducible Go build flags for lumen. Use when changing version, build or output naming.
---

# Versioning and build naming

## Rules
- `VERSION` (repo root) contains exactly `x.x.x`: three integers, newline-terminated. No `v`, no suffix. `scripts/version` validates it.
- A release tag is `vX.X.X` and must equal `VERSION` (`make release-check`, the release workflow).
- **Every package filename ends with `<yyyymmdd-hhmmss>-<githash>`** before the extension: stamp is UTC from `date -u`, computed once per `make` run (override with `BUILD_STAMP` to reproduce a name); hash is `git rev-parse --short=7 HEAD` with `-dirty` when `git status --porcelain` is non-empty (untracked files count); `nogit` outside git; `nocommit` in a repo with no commits yet.

## Names
```
dist/lumen_<ver>_<debarch>_<stamp>-<hash>.deb          # amd64, arm64
dist/lumen_<ver>_windows-<arch>_<stamp>-<hash>.zip     # portable zip
dist/lumen_<ver>_windows-<arch>_<stamp>-<hash>.exe     # Inno Setup installer
dist/lumen_<ver>_macos-arm64_<stamp>-<hash>.zip        # unsigned Lumen.app
dist/lumen-docs_<ver>_<stamp>-<hash>.zip               # reproducible docs bundle
dist/bin/lumen_<os>_<arch>[.exe]                       # plain binaries (unstamped, intermediate)
dist/bin/lumen-console_<os>_<arch>[.exe]               # console variant used by the self-test
```
The package metadata version is `<ver>` only; the stamp is only in the file name.

## `scripts/version`
One line: `VERSION=0.1.0 GIT_HASH=a1b2c3d BUILD_STAMP=20261014-120000 ARTIFACT_SUFFIX=20261014-120000-a1b2c3d`, safe for `eval "$(scripts/version)"` and parsed by the Makefile. Never recompute elsewhere. `make version` prints it.

## Into the binary
Package `internal/version` (`Version`, `Commit`, `Built`) is set with `-ldflags "-s -w -X <module>/internal/version.Version=..."`; `-trimpath` always. `lumen -version` prints `lumen x.x.x (<hash>, <stamp>)`; `/api/v1/version` returns it as JSON. Windows GUI builds add `-H windowsgui`; the exe's file version comes from `VERSION` via `packaging/windows/info.json.in`.

## Bumping
Edit `VERSION` (maintainer or the `release` skill), update `CHANGELOG.md`, `make release-dry`. Do not bump for ordinary work: dev builds share the `x.x.x` and differ by stamp.
