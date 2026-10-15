# Roadmap, status and open issues

Honest status: what is proven, what is only compiled, what is unknown. Update this when something changes status.
Legend: **verified** = run on a real machine and seen; **compiled** = builds/vets but was not run; **unknown** = not tested.

## What is proven (and where)

| Claim | Status | Evidence |
|---|---|---|
| Output window: frameless, always-on-top, tool window, owned by the control window | **verified** (Windows 11) | `spikes/windows-wails/README.md`, run on the first dev host |
| Output covers each monitor exactly, including a monitor at negative coordinates, in Win32 and toolkit modes | **verified** | 12/12 exact and stable for 2 s in the spike |
| WebView2 plays H.264+AAC, HEVC, AV1, VP9+Opus; decodes VP9 alpha; has `setSinkId` | **verified** | spike media probe |
| WebKitGTK 6 on bare Ubuntu 24.04 cannot play H.264/AAC/HEVC/AV1 and lacks `setSinkId` | **verified** | `canPlayType` probe; cause `apt-cache depends` |
| Windows exe cross-compiles CGO-free from WSL (11.9 MB for the spike) | **verified** | spike build |
| The production packages' unit tests (state, config, api, selftest, logging, docgen, main) | **verified** | `go test -race ./...` on Linux |
| The production module vets for Windows (CGO off) and Linux | **verified** | `make vet` |
| The production binary *runs* on Windows | **unknown** | the spike ran; the production app has **not** been started on a Windows host (the session rule forbade writing outside the repo; run `make selftest` there: that is milestone M1) |
| The production binary runs on Linux: server, both windows, page load, media probe, clean shutdown | **verified** (WSLg, Ubuntu 24.04, 3 monitors, Wayland) | `make selftest` passed 2026-10-14; **placement is not gated there** (Wayland cannot position windows) |
| Linux deb builds and installs the right files and dependencies | **verified** (built and inspected, not installed) | `dpkg-deb -I`/`-c` |
| Windows zip builds with version info and the DPI manifest embedded | **verified** (inspected in the PE resources) | `make win-zip` |
| Docs bundle is byte-reproducible | **verified** | two bundles, one hash |
| The full gate (`make ci`) on Linux with the race detector, vulnerability scan clean | **verified** | 2026-10-14, Go 1.26.9 |
| macOS build | **unknown** | only the CI `macos` job builds it; never run |
| Windows installer (Inno Setup) | **unknown** | needs `ISCC.exe`; built only by CI |
| First GitHub Actions run | **unknown** | workflows were never run on GitHub |

## Open issues found in the spike (not resolved)

1. **Cold start 5-9 s** from launch to both windows' first request (3 launches), on a busy machine (Defender real-time on,
   58 other WebView2 processes). Not attributed to Wails vs WebView2 vs the machine; a bare-Edge baseline failed to launch,
   so there is no comparison. Needs a clean measurement; it matters because the output window shows nothing until then.
2. **First placement 24 px off, once** (1 of 3 cold starts, win32 mode, primary monitor); never in 12 repeats. Mitigation in code
   (`wailshost.Reapply` on page load, verify-and-retry); **unproven**.
3. **DPI disagreement:** `GetDpiForMonitor`, `GetScaleFactorForMonitor` and Wails all say 100% on both monitors, yet WebView2
   reports `devicePixelRatio` 1.25 on both (the page saw a 2048x1280 CSS viewport on a 2560x1600 monitor). Unexplained. **Mixed-DPI
   placement is untested** because both monitors read 100%: set the monitors to different scales and run `make selftest`.
4. Not tested at all: monitor hot-plug; focus behavior of the output (it is placed with `SWP_NOACTIVATE`); the GUI-subsystem build
   under `make selftest`; interaction with PowerPoint's fullscreen window; NDI; GStreamer `webview2src` alpha; installers and signing.

## Milestones (proposed order; the owner has not set priorities)

| # | Milestone | Notes |
|---|---|---|
| M0 | Production skeleton, tooling, CI, docs (this commit) | done; CI first run pending |
| M1 | Prove the skeleton on Windows | run `make selftest` and the screenshot on a Windows host; fix what the production app does differently from the spike; measure cold start properly; test mixed DPI |
| M2 | Slides you can author and keep | text import (`---` separated), saved service files, a preview in the control page, keyboard shortcuts |
| M3 | Presentations | the controller interface, PowerPoint COM, the output yielding, blank passthrough, a fake and manual QA checklist |
| M4 | Media | library, ffmpeg probe/transcode sidecar, video backgrounds, audio device routing |
| M5 | Themes and songs | OpenLyrics import, verse order, fonts/outline/shadow, auto-fit text |
| M6 | Hot-plug and robustness | display change events, output fallback and recovery, crash behavior, logging to a file |
| M7 | Broadcast | OBS/CasparCG recipes first; native NDI sender via `purego`; GStreamer DeckLink sidecar if still wanted |
| M8 | Signing and updates | Authenticode, macOS notarisation, an update channel |

## First CI run: things that may need fixing
- `choco install make` and `innosetup` on `windows-latest`; Git Bash as the step shell.
- `apt` packages for GTK4/WebKitGTK 6 on `ubuntu-24.04`; `xvfb` for the Linux self-test.
- GitHub's Windows runner has one virtual display, so the self-test's per-monitor placement is trivial there and may behave differently without a GPU.
- The action SHA pins were copied from `hive-nmos-capture`/`tips-go` and not re-verified.
- `golangci-lint` v2.14 must be built with a Go at least as new as `go.mod` (it is: bootstrap builds it with the local toolchain).

## First commands on a new host
```
git clone <repo> && cd <repo>
make bootstrap          # pinned tools into .tools/bin (installs the pre-commit hook on a developer machine)
make ci                 # the full gate; paste the tail when reporting
make selftest           # on the machine that has the monitors you care about
```
Read `AGENTS.md`, then `docs/decisions.md`, then `docs/session-log.md`.
