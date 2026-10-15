# Lumen

A cross-platform presentation projector for live services: a control window, a floating output window on a second monitor,
and an integrated web API. An OpenLP replacement, **Windows first**. Go + Wails v3; the user interface is plain web pages.

Status: **early (0.1.0)**. A control window and an output window follow one shared state; the output can be placed on a chosen
monitor; the web API and pages work; Windows, Linux and macOS builds exist. Presentation files, media, songs, themes and broadcast
outputs are planned, not built. What is proven and what is not: `docs/roadmap.md`.

## Quick start

Requires Go (the version in `go.mod`). On Windows use Git Bash; on Ubuntu 24.04 also
`sudo apt install build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev`.

```sh
make bootstrap   # pinned dev tools into .tools/bin (and the pre-commit hook)
make ci          # the full gate: format, vet, lint, tidy, docs, tests, vulnerabilities, build
make dev         # run the app with dev/config.toml on 127.0.0.1:18765
make selftest    # drive the real app: place the output on every monitor, probe media
make help        # every target
```

Open `http://127.0.0.1:8765/` (or the dev port) for the control page; the API is documented in `docs/api.md`.

<!-- BEGIN GENERATED: make-targets (scripts/docgen; edit the source, run `make docs`) -->
| Command | Does |
|---|---|
| `make version` | print version, git hash, build stamp and artifact suffix |
| `make bootstrap` | install pinned dev tools into .tools/bin |
| `make dev` | run the app on this host with dev/config.toml on port 18765 (Ctrl-C stops it) |
| `make selftest` | build the console binary and drive the real app: placement on every monitor, media probe; report in dist/selftest/ |
| `make fmt` | format code (goimports + gofumpt) |
| `make fmt-check` | fail if code is not formatted |
| `make vet` | go vet for windows and the host platform (CHECK_OS=host: only the host) |
| `make lint` | golangci-lint for windows and the host platform (CHECK_OS=host: only the host) |
| `make test` | unit tests with the race detector (needs cgo: gcc on Windows); shuffled |
| `make test-windows` | unit tests without the race detector (what make ci runs on Windows) |
| `make cover` | coverage report to dist/coverage.html |
| `make vuln` | govulncheck |
| `make docs` | regenerate generated docs (config reference, API reference, make targets, skills index, packages, versions, notices) |
| `make docs-check` | fail if generated docs are stale or docs reference things that no longer exist |
| `make docs-bundle` | regenerate and check the docs, then write a reproducible docs zip to dist/ |
| `make hooks` | install the git pre-commit hook (regenerates and stages docs); run once per clone |
| `make exec-check` | fail if a tracked script is not stored as executable in git |
| `make tidy-check` | fail if go.mod/go.sum are not tidy |
| `make ci` | the full gate (fmt, vet, lint, script modes, tidy, docs, tests, vuln, build); identical on every CI job |
| `make build` | host binary to dist/bin/, plus the Windows binary |
| `make deb` | .deb for the host architecture in dist/ (Linux only; needs libgtk-4-dev and libwebkitgtk-6.0-dev) |
| `make win-zip` | portable Windows zip (exe + example config + licences) in dist/; cross-builds from any host |
| `make installer` | Windows installer (Inno Setup, needs ISCC.exe on Windows) in dist/ |
| `make macos-app` | Lumen.app zip in dist/ (macOS only; unsigned) |
| `make packages` | all packages the host can build: Linux deb + Windows zip; on Windows, zip + installer; on macOS, the app zip |
| `make release-dry` | everything a release does, minus publishing |
| `make release-check` | verify TAG=vX.X.X matches VERSION and the tree is clean |
| `make clean` | remove dist/ |
| `make help` | list targets |
<!-- END GENERATED: make-targets -->

## Packages

| Platform | Package | Command |
|---|---|---|
| Windows 11 | zip, Inno Setup installer (needs WebView2, preinstalled on Windows 11) | `make win-zip`, `make installer` |
| Ubuntu 24.04+ | `.deb` (amd64, arm64) | `make deb` |
| macOS (Apple Silicon) | unsigned `.app` zip | `make macos-app` |

Artifact names carry a build stamp: `lumen_<x.x.x>_<arch>_<yyyymmdd-hhmmss>-<githash>.<ext>`.

## Layout

`cmd/lumen` (thin main), `internal/` (state, api, host, config, selftest, ...), `scripts/` (all build and doc logic), `packaging/`
and `os/` (what gets installed), `docs/` (intent, architecture, decisions, roadmap), `.claude/skills/` (agent playbooks),
`spikes/` (archived prototypes). Start with `AGENTS.md`, then `docs/decisions.md` and `docs/session-log.md`.

## Licence

Licensed under the Apache License, Version 2.0 (Apache-2.0; see `LICENSE`, `NOTICE`, `THIRD_PARTY_NOTICES.md`).

<!-- BEGIN GENERATED: copyright (scripts/docgen; edit the source, run `make docs`) -->
Copyright 2026 Evobytes Consulting Ltd.
<!-- END GENERATED: copyright -->
