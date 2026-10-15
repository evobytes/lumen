# AGENTS.md — lumen

Canonical guide for humans and AI agents. `CLAUDE.md` is a symlink to this file.
Task-specific depth lives in `.claude/skills/<name>/SKILL.md`; the product brief in `docs/intent.md`; design rationale in
`docs/architecture.md`; **every decision, with evidence, in `docs/decisions.md`**; the session narrative in `docs/session-log.md`;
what is proven vs unproven in `docs/roadmap.md`. Picking this project up cold: read this file, then those, in that order.

## What this is

`lumen`: a cross-platform presentation projector (an OpenLP replacement) for live services. One Go process: an HTTP server that
owns the state and serves web pages, plus two Wails v3 windows that load them: a **control** window and a frameless, always-on-top
**output** window that is a Win32 *owned child* of the control window and can cover a chosen monitor. Windows 11 is the **primary**
platform; Ubuntu 24.04+ and macOS (Apple Silicon) are secondary (decision 003). Not a service: a per-user desktop app.

Module and binary: `github.com/evobytes/lumen` / `lumen` (the owner/repo name is assumed: decision 015).

**Status: 0.1.0 skeleton. `make ci` passes; the Linux build was run end to end under WSLg; the Windows build compiles but has not been run in its production form.** The *spike* it was ported from was
run on a real two-monitor Windows 11 machine (`spikes/windows-wails/README.md`). Do not describe anything as working on a platform
unless it was run there; "compiled for Windows" is not "tested on Windows". `docs/roadmap.md` is the status ledger.

## Golden rules

1. **`make` is the only interface.** CI workflows only call `make <target>`. If `make ci` passes locally, CI passes. Never put logic in workflow YAML; put it in `scripts/` and call it from `make`.
2. **Run `make ci` before saying work is done.** Report failures verbatim; never claim a check passed that you did not run. Code behind `//go:build windows` is only **compiled** unless you ran it on a Windows host; say which.
3. **Write only inside this repository.** Caches, tools, clones and scratch files stay under it (`.cache/`, `.tools/`, `.ref/`, `dist/`, all git-ignored; agent sessions `source scripts/agent-env`). The one script that must write elsewhere, `scripts/windows-host-selftest`, refuses without `LUMEN_ALLOW_HOST_WRITE=1`. Decision 012.
4. **State is the single source of truth; the output is a URL.** Renderers (the Wails windows, OBS, a phone) re-read state on every event. Never give a renderer its own copy (decision 004).
5. **The windowing toolkit is behind `internal/host`.** Only `internal/host/wailshost`, `internal/app` and `cmd/lumen` may import Wails. Window facts come from the OS (`internal/win32`), not from the toolkit's opinion of itself.
6. **Idiomatic Go, stdlib first.** Add a dependency only with a stated reason (put it in the commit message). The project is Apache-2.0: dependencies must be permissively licensed (MIT, Apache-2.0, BSD, ISC, 0BSD, Unlicense); `make docs-check` fails on anything else.
7. **No JS/CSS build tooling.** The pages are static HTML, CSS and native ES modules under `internal/ui/assets`, served with a strict CSP: no inline script, no `style=""` (decision 019).
8. **Unverified is labelled.** Evidence is **verified** (run or read in source), **reported** (a secondary source) or **assumed**. Unknown is `null`/absent, never zero or "fine".
9. **Do not commit, push, tag, or release unless asked.**

## Layout

```
lumen/
├── AGENTS.md  CLAUDE.md(symlink)  README.md  CHANGELOG.md
├── VERSION                  # x.x.x — single source of truth for the version
├── COPYRIGHT                # the one line that feeds NOTICE, the deb copyright and the README
├── Makefile                 # the only entry point
├── go.mod  go.sum           # module github.com/evobytes/lumen (Go pinned to an exact patch)
├── cmd/lumen/               # main package: thin — flags, config, logging, hands off to internal/app
├── internal/
│   ├── app/             # wiring: API server + Wails windows + self-test/demo; per-OS load-finished event
│   ├── api/             # HTTP API (/api/v1), SSE, pages, problem+json, token auth; the route table docs come from
│   ├── state/           # slides, position, blank, subscribers (pure, tested)
│   ├── host/            # Host interface and types (physical px, virtual desktop); hosttest/ fake; wailshost/ Wails impl
│   ├── win32/           # monitors, DPI, window rectangles, ownership, SetWindowPos (windows) / inert twin (others)
│   ├── selftest/        # drives the real app and writes a JSON report; embedded media clips
│   ├── config/          # defaults < TOML file < LUMEN_* env < flags; toml + doc tags feed the generated docs
│   ├── logging/         # slog construction
│   ├── ui/              # embedded static pages (control, output) and ES modules
│   └── version/         # ldflags-injected version info
├── os/                      # mirrors the Linux target filesystem (desktop entry, icon, generated example config)
├── packaging/               # nfpm.yaml, windows/ (Inno Setup script, manifest, icon, generated config), macos/ (Info.plist)
├── scripts/                 # all non-trivial logic; Makefile + CI call these (docgen/ is a Go tool)
├── docs/                    # intent, architecture, decisions, roadmap, windows-dev, session-log, research/
├── spikes/                  # archived prototypes (windows-wails, python-pyside6); own go.mod or not Go; ignored by CI
├── dev/config.toml          # used by `make dev` (dev/data/ is gitignored)
├── dist/  .tools/  .cache/  .ref/   # output, tools, caches, reference clones (all gitignored)
├── .claude/skills/          # agent playbooks
├── .githooks/pre-commit     # regenerates and stages docs; run `make hooks` once per clone
└── .github/workflows/       # ci.yml, release.yml
```

## Commands

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

## Versioning and artifact naming

- Version is `x.x.x` (semver, no `v` prefix, no pre-release suffix) in `VERSION`. Git tags are `vX.X.X` and must equal `VERSION`.
- **Every built package gets a build stamp appended**: `<yyyymmdd-hhmmss>-<githash>` — UTC time, 7-char short hash, `-dirty` appended to the hash if the tree is dirty. Before the first commit the hash reads `nocommit`; outside git, `nogit`.

```
dist/lumen_0.1.0_windows-amd64_20261014-120000-a1b2c3d.zip
dist/lumen_0.1.0_amd64_20261014-120000-a1b2c3d.deb
dist/lumen_0.1.0_macos-arm64_20261014-120000-a1b2c3d.zip
```

Details: `versioning-builds` skill.

## Conventions (summary — see skills for detail)

- **Go**: `gofumpt`, `goimports`, `golangci-lint`; wrap errors with `%w`; `context.Context` first param; no globals; small interfaces defined at the consumer; `log/slog`. Portable core, thin `//go:build windows` edge with an inert twin. (`go-style`)
- **Tests**: table-driven, `t.Parallel()`, `httptest`, fakes only at external boundaries (`hosttest.Fake` for the windows). (`testing`)
- **HTTP**: stdlib `ServeMux` with method+path patterns; problem+json errors; bounded strict JSON; the route table is data. (`add-feature`, `output-page`)
- **Windows specifics**: owner windows, `SetWindowPos`, DPI manifest, WebView2. (`wails-windows`, `windows-host-testing`)
- **Config precedence**: defaults < per-user `config.toml` < `LUMEN_*` env < flags. (`config-logging`)
- **Media and presentations**: HTML5 video, not VLC; PowerPoint COM behind an interface. (`media`, `presentation-control`)
- **Broadcast**: outsource first (OBS/CasparCG), sidecars later. (`broadcast-outputs`)
- **Docs are enforced**: a pre-commit hook regenerates and stages generated docs and refuses behavior changes with no doc update; `make docs-check` is the CI backstop. (`sync-docs`)

## Skills index

<!-- BEGIN GENERATED: skills-index (scripts/docgen; edit the source, run `make docs`) -->
| Skill (also a `/slash` command) | What and when |
|---|---|
| `/add-feature` | End-to-end checklist for adding a feature (API route, state change, window behavior, output-page behavior, config key) to lumen. Use first for any non-trivial change. |
| `/broadcast-outputs` | How lumen reaches NDI and SDI (Blackmagic DeckLink fill and key): the outsourcing order, NDI runtime loading, GStreamer DeckLink, alpha-frame sources and licensing limits. Not built yet; read before any broadcast work. Use when adding NDI, SDI or desktop-capture output. |
| `/ci-workflows` | GitHub Actions workflows for lumen and the make-first rule: what each workflow does, how to add a step, how to reproduce locally. Use when editing .github/ or the Makefile. |
| `/config-logging` | Configuration keys and precedence, environment variables, slog logging and the generated example config files for lumen. Use when adding config or logging. |
| `/dev-loop` | Fast local development for lumen: make dev, tool bootstrap, working in WSL2 with Windows interop or natively on Windows with Git Bash. Use when running or changing the dev workflow. |
| `/go-style` | Idiomatic Go conventions for lumen: layout, errors, interfaces, concurrency, platform code, lint and format. Use when writing or reviewing any Go code. |
| `/media` | Media playback rules for lumen: HTML5 video in the webview, codec findings per OS, no VLC, ffmpeg transcode-on-import, audio routing. Use when touching video, audio, backgrounds or media import. |
| `/output-page` | Conventions for the control and output web pages in lumen: output as a pure function of state, event-driven refresh, strict CSP, no build step, the probe and debug modes. Use when editing anything under internal/ui. |
| `/packaging` | Building and verifying lumen's packages: Linux deb (nfpm), Windows zip and Inno Setup installer, macOS app zip; the os/ tree and what each package installs. Use when touching os/, packaging/, scripts/build-* or make packages. |
| `/presentation-control` | Design and constraints for controlling PowerPoint (COM on Windows, AppleScript on macOS) and other external presentation apps from lumen. Not built yet; read before implementing. Use when adding presentation file support. |
| `/release` | Cut a tagged release of lumen: version bump, changelog, dry run, tag, publish verification. Use only when the user asks for a release. |
| `/security` | Security checklist for lumen: API token, loopback binding, CSP, request bounds, what no request may name, window and native-code safety, dependencies and packaging. Use when reviewing changes or touching the API, pages, config, native code or packaging. |
| `/session-end` | Persist everything learned in a long working session before it closes: update the decision log, session log, docs and skills in the repo, refresh the assistant's persistent memory files, and list pending decisions and next steps. Use when the user says the session is ending, or after any large chunk of work. |
| `/spikes` | What the archived spikes under spikes/ are, what they proved, and how production code was ported from them. Use when you need the evidence behind a decision or want to reuse spike code. |
| `/sync-docs` | Bring every kind of documentation in lumen current: regenerate generated docs, run the staleness lint, and update the hand-written docs that a change affects. Use after any behavior, config, Makefile, skill, route or dependency change, or when the pre-commit hook or make docs-check complains. |
| `/testing` | How to write and run tests in lumen: table-driven unit tests, handler tests, the fake window host, the self-test, Windows-only code, coverage. Use whenever adding or changing behavior. |
| `/verify` | Run the right verification for the change you made (unit, lint, docs, Windows build and self-test, packaging) and report results honestly. Use before saying any work is done. |
| `/versioning-builds` | Version scheme (x.x.x), build stamp (yyyymmdd-hhmmss-githash), artifact filenames, ldflags injection and reproducible Go build flags for lumen. Use when changing version, build or output naming. |
| `/wails-windows` | Wails v3 and Win32 specifics for lumen's windows: owner windows, exact placement, DPI manifest, frameless output, load-finished re-apply, build tags and known toolkit gaps. Use when touching internal/host/wailshost, internal/win32 or internal/app. |
| `/windows-host-testing` | How to run, screenshot and self-test lumen on a real Windows host, from WSL2 interop or natively, within the write-only-inside-the-repo rule. Use when verifying window, monitor, DPI or output behavior. |
<!-- END GENERATED: skills-index -->

## Definition of done

- `make ci` passes (paste the tail of the output when reporting).
- New behavior has tests; bug fixes have a regression test.
- User-facing or operational change: `README.md` / `docs/` / the relevant skill / `CHANGELOG.md` updated; a decision gets a numbered entry in `docs/decisions.md`.
- If anything under `os/` or `packaging/` changed: the relevant `make deb` / `win-zip` / `installer` still builds (say which you could and could not run).
- If window, DPI, placement or output-page behavior changed: `make selftest` on a Windows host (or say it was not run) and, for visual changes, a screenshot you looked at (`windows-host-testing`).
- State exactly what was compiled versus run, and on which OS.
- No new dependency without justification; `make tidy-check` clean.

## Do not

- Edit `go.sum` by hand, or the generated files (`docs/configuration.md`, `docs/api.md`, `docs/versions.md`, `NOTICE`, `THIRD_PARTY_NOTICES.md`, `packaging/copyright`, the two example config files, the blocks between `GENERATED` markers).
- Add `//nolint` without a reason comment.
- Use `panic` for control flow, `log.Fatal` outside `main`, or `fmt.Println` for logs.
- Import Wails outside `wailshost`, `app` and `cmd`; hard-code install paths; put logic in workflow YAML.
- Introduce CGO on Windows. (Linux and macOS builds need it because Wails does: decision 017.)
- Bundle NDI, ffmpeg, GStreamer, OBS or Blackmagic software, or embed secrets; optional integrations are user-installed.
- Write inline `<script>`, `style=""` or event-handler attributes in the pages (the CSP forbids them).
- Run `scripts/windows-host-selftest` or put windows on the owner's desktop without saying so first; never kill processes by broad pattern (`pkill -f` can match your own shell).
- Claim a Windows behavior works because it compiled, or a placement is right because a screenshot looked black: the page may not have loaded yet (cold start 5-9 s on the first host).
