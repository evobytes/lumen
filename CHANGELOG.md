# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/). Versions are `x.x.x`.

## [Unreleased]

## [0.1.0]

### Added
- Production Go module `github.com/evobytes/lumen`: `cmd/lumen`, `internal/{app,api,config,host,logging,selftest,state,ui,version,win32}`.
  A control window and a frameless, always-on-top, owned output window (Wails v3 beta.29) that follow one shared state; slides,
  next/previous/show/blank; placement of the output on a chosen monitor (Win32 `SetWindowPos` with verify-and-retry, or the toolkit),
  or floating; re-application of the placement after the output page loads.
- Web API under `/api/v1` with server-sent events, problem+json errors, optional token, a strict CSP and no inline script or style;
  pages are static ES modules; `docs/api.md` is generated from the route table.
- `lumen -selftest`: places the output on every monitor repeatedly, checks owner/topmost/tool-window, runs a media probe, writes a
  JSON report; `-demo` parks the output on a monitor.
- Configuration (`LUMEN_*`, TOML, flags) with generated reference and example files.
- House tooling from `hive-nmos-capture`/`tips-go`: `make` as the only interface, `scripts/` (`build-binary`, `build-deb`,
  `build-windows-zip`, `build-windows-installer`, `build-macos-app`, `build-docs`, `selftest`, ...), `VERSION` and build stamps,
  pinned tools in `.tools/bin`, docgen with a pre-commit hook, GitHub workflows (CI, release), Dependabot.
- Documentation: intent, architecture, decisions 001-026, roadmap, Windows development guide, session log, research assessments;
  `.claude/skills` playbooks including `session-end`.
- Spikes archived under `spikes/` (Windows/Wails, Python/PySide6).

- `host.Capabilities` and a self-test that gates only on what the platform can do; unique toolkit monitor IDs; the HTTP server now shuts down
  cleanly with event streams open.
- `scripts/agent-env`: a tracked helper that keeps caches, tools and the app profile inside the repository.

### Changed
- Go is pinned to 1.26.9 (the gate flagged 21 standard-library vulnerabilities in 1.26.2).
- Licence is Apache-2.0 with copyright Evobytes Consulting Ltd (the Python prototype had MIT).
