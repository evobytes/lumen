# Decisions

Short ADR log. Append new entries; do not rewrite history: supersede instead. Numbers are sequential
(`make docs-check` enforces it). Dates are 2026-10 unless stated. Where a decision rests on evidence,
the evidence is named and its status is given: **verified** (run or read in source this project),
**reported** (a secondary source), or **assumed**.

## 001 — The product is not written in Python
The first prototype was Python + PySide6 (see `spikes/python-pyside6`). It worked, but the owner finds
shipping Python apps painful (PyInstaller bundles of 66 MB for Linux, per-OS packaging quirks) and prefers
compiled languages. The prototype stays as a reference: its API shape, its test ideas and its remote page
fed the Go design. It is not built by CI and can be deleted once nobody needs the reference.
Cost: PySide6/Qt has the strongest multimedia and multi-screen capability; we give that up (see 002, 008).

## 002 — Go, with Wails v3 and a web UI
Go for the core, API and window orchestration; HTML/CSS/JS (no build step) for everything visual.
Why: it matches the owner's skills (good Go, excellent web), the output is "styled text over image/video"
which a browser engine does best (OpenLP itself renders its output as HTML), and the same pages serve the
control UI, the output window and a phone remote. Wails v3 was chosen over other Go options because it is the
only one with frameless + always-on-top + screen placement + multi-window together (verified in its source).
Rejected: Gio (no window position or monitor choice, verified in v0.10.3 source), Fyne 2.8 (no general
frameless window, no video, verified), Wails v2 (single window), Swift (Apple-only in practice), Flutter
(desktop multi-window unshipped). Fallbacks if the spike assumptions fail: C# + Avalonia 12 (native; the
HTML output would need a paid WebView component) or Rust + Tauri 2 (stable webview shell, needs Rust).
Full comparison and scoring: `docs/research/language-assessment.md`. The three-way score (Wails 74, Avalonia 75,
Tauri 73) is a tie within noise: the choice rests on skills and product fit, not on the totals.
Costs: Wails v3 is **beta** (pinned at `v3.0.0-beta.29`, released 2026-10-13, about weekly), with open
multi-monitor bugs on macOS (#5409, mixed DPI) and Linux (#4003). A GUI cannot be fully static: Windows is
CGO-free but needs WebView2; Linux needs GTK4 + WebKitGTK 6; macOS needs Cocoa/WebKit (see 017).

## 003 — Windows 11 is the primary platform
The operators run Windows. Targets: Windows 11, Ubuntu 24.04+, macOS latest on Apple Silicon only. No legacy
GTK3 / WebKit2GTK 4.1 stack, no Intel or universal macOS builds. Linux and macOS are secondary: they must
build (CI), and their gaps are tracked, but Windows behavior decides design questions. Windows-only
integrations (PowerPoint COM, DeckLink, NDI runtime) are first-class; the other platforms get equivalents
later behind the same interfaces. Consequence: design for the Windows webview (WebView2 = Chromium) and treat
WebKitGTK/WKWebView limits as secondary constraints, not the lowest common denominator.

## 004 — The output is a URL-driven web page; renderers are interchangeable
The Go server owns the state (`internal/state`) and serves `/output` (the audience view), `/control` and an
event stream. The output page is a pure function of the state and re-reads it on every server-sent event.
The Wails output window is just one renderer of that URL; OBS (browser source), CasparCG (HTML producer), a
kiosk Chromium or a second machine can render the same page. This is what keeps broadcast outputs (NDI, SDI
fill-and-key) deferrable and the windowing toolkit replaceable (see 005, 010). The API is the single source of
truth; the UIs are clients of it.

## 005 — Window management sits behind the `host.Host` interface
`internal/host` defines `Host` (screens, place output on a screen, float output, report windows, quit) in
physical pixels with virtual-desktop coordinates (the origin can be negative). `internal/host/wailshost`
implements it; `internal/host/hosttest` fakes it. The API, self-test and state never import Wails. Win32 code
(`internal/win32`, CGO-free, with an inert non-Windows twin) is used only where Wails lacks a capability
(window ownership) or where ground truth is needed (monitor rectangles, DPI, window rectangles). Cost: a thin
abstraction to maintain; benefit: Wails can be swapped without touching the rest.

## 006 — The "floating child" is a Win32 owned window
The output window is frameless, always-on-top, hidden from the taskbar (tool window) and **owned** by the
control window, set with `SetWindowLongPtr(GWLP_HWNDPARENT)` because Wails v3 has no public owner/parent
option. Ownership keeps it above its owner and minimises/destroys it with it. **Verified** on Windows 11 in the
spike (owned by control, topmost, tool window). It covers a monitor by setting the native rectangle, not by OS
fullscreen (fullscreen creates a Space on macOS and can steal the main display on some Windows setups). The
output window is created with Wails' frameless decorations disabled so there is no shadow or rounded corner.

## 007 — Placement: Win32 `SetWindowPos` by default on Windows, verified and re-applied
`output.placement = win32` (default on Windows) sets the native rectangle from `EnumDisplayMonitors`, then
verifies it stuck and retries (`wailshost.settle`); `toolkit` uses Wails' `SetBounds` (default elsewhere).
Spike evidence (**verified**, Windows 11, two monitors, one at (-2560,-90)): 12/12 placements exact and stable
over 2 s in both modes. One cold start put the output 24 px high on its first placement (1 of 3 cold starts,
never in the 12 repeats); the output page's load-finished event now re-applies the last placement
(`wailshost.Reapply`). That mitigation is **unproven**. Unverified: mixed-DPI placement (the spike machine
reported 100% on both monitors by every Win32 API), hot-plug.

## 008 — Media: HTML5 `<video>` in the webview; not VLC
VLC is the wrong dependency for a new project. OpenLP's own master branch no longer uses it: its media code is
Qt Multimedia only (**verified** by reading `openlp/core/ui/media` on GitLab). libVLC 4 had no official release
as of May 2026 (**reported**), and libVLC 3 is heavy to bundle. Lyrics over video backgrounds is the product's
main use, and browser compositing makes that trivial, whereas native video widgets cause "airspace" problems.
Evidence: WebView2 154 on Windows 11 plays H.264+AAC, HEVC, AV1, VP9+Opus and decodes VP9-with-alpha, and exposes
`setSinkId` (**verified**). WebKitGTK 6 on a bare Ubuntu 24.04 plays **none** of H.264/AAC/HEVC/AV1 and has no
`setSinkId` (**verified** with a `canPlayType` probe), because the package only *suggests* `gstreamer1.0-libav`.
Therefore: the `.deb` declares `gstreamer1.0-libav` and `-plugins-good` as dependencies, and an optional
ffmpeg-based transcode-on-import to VP9/Opus WebM (royalty-free, plays in every webview) is the portable answer.
Per-device audio routing must not rely on the page off Windows. WKWebView codecs are untested.

## 009 — Presentation files: PowerPoint COM on Windows, AppleScript on macOS, behind one controller interface
OpenLP drives PowerPoint with late-bound COM on Windows and AppleScript on macOS (**verified** by reading its
source, including a list of quirks: the `DisplayMonitor` registry value to pick the monitor, ghost windows, a
PowerPoint 2010 unblank bug, focus stealing that pauses the show, AppleScript unable to jump to a slide). We follow
the same approach from Go (`go-ole` on Windows, `osascript` on macOS) behind a `Load/Start/Next/Prev/Goto/Blank/Stop`
interface with a fake for tests. LibreOffice Impress is not required (the owner said so); a PDF/image fallback
covers machines without PowerPoint. The output window must yield while an external app owns the screen (hide or
lower it; route "blank" through COM). Real PowerPoint cannot run in GitHub CI, so it needs manual QA. Not built yet.

## 010 — Broadcast outputs (NDI, SDI fill-and-key) are deferred and outsourced first
The owner wants NDI and possibly Blackmagic fill-and-key "in extremis". Decision: build nothing in the core now.
First route: point OBS (NDI via DistroAV, DeckLink) or CasparCG (HTML producer, DeckLink, NDI; **Windows and
Ubuntu only, no macOS build**, verified from its release assets) at `/output`. Later, optional sidecars: an NDI
sender that loads the NDI runtime at run time with `purego` (no CGO), and a GStreamer process for desktop capture to
`decklinkvideosink` (hardware keyer modes off/internal/external with BGRA frames, **verified** in its docs).
NDI's runtime is not bundled: its terms allow redistribution only under conditions, dynamic loading is "the
preferred mechanism", and DistroAV (GPL-2.0) sets the precedent of the user installing the runtime. System webviews
give no alpha frames; fill-and-key of the lyrics page needs a Chromium renderer (CEF, OBS or CasparCG). PowerPoint
has no alpha output, so only a plain fill feed (desktop capture) is possible for it. The DeckLink SDK has no
maintained Go binding (searched) and its Windows API is COM that `go-ole` (IDispatch) cannot call (assumed).

## 011 — Develop in WSL2 with Windows interop, not natively on Windows (for now)
Claude Code runs in WSL2 Ubuntu 24.04 with the repo on ext4. Windows binaries are cross-compiled (pure Go) and run
on the host through interop. Switch to native Windows if interop proves unreliable, routine Delve debugging or hot
reload on the Windows build is wanted, or 9p path slowness becomes a regular cost (then use a Taskfile or Git Bash,
Git for Windows and a Defender exclusion; `hive-nmos-capture` develops natively on Windows with Git Bash). Interop
was broken at the start of the project by a systemd/`binfmt_misc` race that dropped the `WSLInterop` registration;
it is fixed by `/etc/systemd/system/wsl-interop-fix.service` on that host (reboot persistence untested). See
`docs/windows-dev.md`. The first dev host is expected to change: that document is how to rebuild the setup.

## 012 — Write only inside the repository
Standing instruction from the owner (2026-10-14): only ever write to this directory and its children. In practice:
Go caches and tools are repo-local (`.cache/`, `.tools/`, set by `scripts/agent-env` during agent sessions; plain
`go` uses its usual caches for people), reference clones go under `.ref/` (git-ignored), and the one script that
must write elsewhere (`scripts/windows-host-selftest`, which copies an exe to the Windows side) refuses to run
without `LUMEN_ALLOW_HOST_WRITE=1`. The agent memory directory under `~/.claude` is the assistant's own and is
updated only by the `session-end` skill.

## 013 — House conventions follow `hive-nmos-capture`, then `tips-go`
Reference projects were chosen by recency and relevance, not by name. `reference-go-app-template` is a snapshot
of `tips-go` from 2026-10-10 and is already behind the live `tips-go` (22 skills vs 21, `docs/intent.md`, MOS). The
newest and closest is `dB-Broadcast/hive-nmos-capture` (pushed 2026-10-13): a Windows-first Go app whose CI gate
runs on Windows and Linux, with `make vet`/`lint` for both OSes, an Inno Setup installer, a script-mode check and a
`windows-service` skill. Adopted: `make` is the only interface and CI calls only `make`; logic in `scripts/`;
`os/` mirrors the target filesystem; `VERSION` plus a build stamp in every artifact name; generated docs with a
pre-commit hook and a CI backstop (`scripts/docgen`); `.claude/skills`; numbered ADRs; `AGENTS.md` with a `CLAUDE.md`
symlink; pinned tools in `.tools/bin`; Dependabot; SHA-pinned actions (the pins were taken from those repos and
not re-verified). `hive-cli-tools` and `hive-driver-evs-cerebrum` supplied the `build-deb`/`build-package` naming.

## 014 — Licence: Apache-2.0; copyright holder Evobytes Consulting Ltd
Matches `tips-go` (the owner's own consultancy) rather than the dB Broadcast Ltd copyright used by the hive
repositories. Dependencies must be permissively licensed (MIT, Apache-2.0, BSD, ISC, 0BSD, Unlicense); `make
docs-check` fails otherwise. This **supersedes** the MIT licence and personal copyright placed in the first
prototype. **Assumed, please confirm**: if the project belongs to a different entity, edit `COPYRIGHT` and run `make docs`.

## 015 — Module path `github.com/evobytes/lumen`
The binary and command are `lumen` (the owner wrote `cmd/lumen/main.go`); the directory is `openlp-replacement`.
The GitHub owner and repository name are **assumed** (`evobytes/lumen`, following `tips-go`). To change it: `go mod
edit -module <new>`, rewrite imports with `grep -rl github.com/evobytes/lumen . | xargs sed -i ...`, then update
`.golangci.yml` (`local-prefixes`), `.vscode/settings.json`, `scripts/docgen/generate.go` (`modulePath`),
`packaging/nfpm.yaml` and `scripts/docgen/license.go` (the Debian copyright `Source:`). The Makefile reads the module
path from `go.mod`.

## 016 — Build scripts are extension-less and named for what they build
`scripts/build-deb`, `scripts/build-windows-zip`, `scripts/build-windows-installer`, `scripts/build-macos-app`,
`scripts/build-binary`, `scripts/build-docs` (and `scripts/version`, `scripts/bootstrap`, ...). This follows the
owner's request for "`scripts/build-deb` and friends" and the `hive-*` repositories; `tips-go` and `hive-nmos-capture`
use `.sh`. Scripts are executable bash, run under Git Bash on Windows; `make exec-check` guards the mode bit.

## 017 — CGO policy: Windows is CGO-free; Linux and macOS need CGO
The house rule is `CGO_ENABLED=0` for shipped binaries. Wails v3 needs CGO on Linux (GTK4, WebKitGTK 6) and macOS
(Cocoa, WebKit) but not on Windows, so this is a **deliberate deviation**: the Windows exe cross-compiles from any
host with no C toolchain (verified: 11.9 MB for the spike); Linux and macOS binaries build natively only
(`scripts/build-binary` refuses to cross-build them). Consequences: `make vet`/`lint` always check Windows and add
the host platform (`CHECK_OS`); the `.deb` builds on a Linux runner with the GTK/WebKit dev packages; CI builds macOS on
a macOS runner. The race detector needs CGO too, so the Windows gate runs `test-windows` (no `-race`).

## 018 — Configuration
TOML file, `LUMEN_*` environment, flags; precedence defaults < file < environment < flags (as the references).
Per-user file (`%APPDATA%\Lumen\config.toml`, `~/.config/Lumen/config.toml`): a desktop app, not a service.
`server.listen` defaults to loopback; any non-loopback address requires `auth.token`. `-check-config` prints the
effective configuration with the token redacted. Keys carry `toml` and `doc` tags, which generate the reference and
the shipped example files; a test fails if a key lacks documentation or an environment mapping.

## 019 — Strict CSP and no JS build step
Pages are static HTML, CSS and native ES modules with no inline script or style, served with
`default-src 'self'` (plus `media-src 'self' blob:` and `img-src 'self' data:`). No npm, bundler or framework
(the house rule). The API token reaches the pages as `?token=` on their URL; the event stream alone accepts a
query token because `EventSource` cannot set headers, and the request log never records query strings.

## 020 — API shape
REST + server-sent events under `/api/v1`: status, next/prev, show slide, replace slides, blank, screens, output
placement, windows report, version, diagnostics reports, events. Errors are RFC 9457 problem+json; bodies are
bounded JSON with unknown fields rejected; the optional token is compared in constant time. The shape carries over
from the Python prototype. `docs/api.md` is generated from the route table (`api.Routes`), which the router is built from.

## 021 — The self-test is a first-class verification tool
`lumen -selftest` drives the real app through its own host seam and output page, places the output on every
monitor in every placement mode several times (sampling the rectangle over 2 s), checks owner/topmost/tool-window,
runs the media probe, and writes `report.json`. **Pass** requires that the output page loaded and reported (the app,
its server and its window really ran) and, only where the platform can do it (`host.Capabilities.ExactPlacement`,
true on Windows only), that every placement was exact and the output was topmost. Codec support is reported but never
gates (it varies by machine). Without capabilities the first Linux run failed for the wrong reason: Wayland forbids
positioning windows, so a gate there only measures the compositor. It is how packaged builds are verified; Windows CI
runs it as an experimental job (one virtual display, no Office).

## 022 — Spikes are archived, not built
`spikes/windows-wails` (the Wails spike with its README of verified results) and `spikes/python-pyside6` stay as
provenance: they have their own `go.mod` or are not Go at all, so the root module and CI ignore them. Production
code was ported from the Go spike into `internal/` with improvements (verify-and-retry placement, re-apply on page
load, owner relation, typed host seam, tests). Delete a spike once its findings are in `docs/` and nobody needs it.

## 023 — CI shape
`ci.yml`: a Windows job (the primary: the full gate, as a developer runs it), a Linux job (adds the race detector and
GTK/WebKit), a package job gated on both (Windows zip + Inno Setup installer; Linux deb), an experimental macOS
build, and an experimental Windows self-test. `release.yml` on `v*` tags repeats the gate, builds all packages, then
publishes checksums, build-provenance attestations and a GitHub release. Workflows contain no logic: they install
tools and call `make`. Not yet run on GitHub: the first run may need small fixes (see `docs/roadmap.md`).

## 024 — Documentation generation and reproducible bundles
`make docs` regenerates: config reference and example config files, the API reference, make-target tables, the
skills index, the package table, `docs/versions.md`, `NOTICE`, `THIRD_PARTY_NOTICES.md` and the Debian copyright;
`make docs-check` (in `make ci`) fails on stale output or broken references; the pre-commit hook regenerates and
stages. `make docs-bundle` writes a docs zip that is byte-reproducible (sorted entries, fixed timestamps). The
licence scan unions the module sets of the Windows, Linux and macOS builds (the CGO setting does not change the set).

## 025 — Wails is pinned and upgraded deliberately
`go.mod` pins `v3.0.0-beta.29` and `scripts/tools.env` pins the same CLI version (used only to generate the
Windows resource file: icon, version info and the per-monitor-v2 DPI manifest). Upgrade by bumping both, running
`make ci` and the self-test on a Windows host, and recording any API change. Beta releases are weekly, so expect churn;
a DPI manifest is required (without it a Windows process is DPI-unaware and every monitor test is invalid).

## 026 — Open questions the owner has not decided
Not decisions, so that no one invents them: the GitHub owner/repo name (015); the copyright holder (014); the
feature scope beyond the first slice (see `docs/intent.md`: songs, Bible, service plans and import formats are
**proposed**, not agreed); code signing (Windows Authenticode, macOS notarisation); whether to adopt a Taskfile for
native Windows development; the install scope of the Windows installer (currently per-machine); whether broadcast
outputs are ever built in. Each should become a numbered decision when made.

## 027 — Go is pinned to an exact 1.26 patch and bumped when `make vuln` says so
`go.mod` pins `go 1.26.9` (the current patch of the line `tips-go` also uses; Go 1.27 had just been released and is
not adopted yet). The first gate run on Go 1.26.2 failed `govulncheck` with 21 standard-library vulnerabilities fixed in
1.26.3 to 1.26.9, which is the gate working as designed. The `go` command downloads the pinned toolchain
automatically (`GOTOOLCHAIN=auto`; into the repo-local cache for agents). After a bump, delete `.tools/bin` and run
`make bootstrap`: `golangci-lint` must be built with a Go at least as new as the module, and the bootstrap script only
checks the tool's own version string.

## 028 — Repo-local caches for agents; formatters see only our source
Consequence of decision 012. `scripts/agent-env` (tracked; sourced) points the Go caches, tool installs and the Linux
app profile at git-ignored directories inside the repo. A bug found by doing it: `goimports -w .` and `gofumpt` with
`.` walk the module cache and the reference clone and will even rewrite cache files, so `make fmt` and `make fmt-check`
take explicit directories (`GOSRC := cmd internal scripts`). The spikes (own modules) are excluded on purpose.
`go mod verify` was run afterwards and passed.

## 029 — Platform capabilities, unique monitor IDs and clean shutdown
The first run of the production app on Linux (WSLg, 2026-10-14) found three defects, now fixed: (1) `host.Host`
gains `Capabilities()` so the self-test gates only on what a platform can do (decision 021); (2) toolkit-mode monitor
IDs are `<name>#<id>` because WSLg reports the name `rdp` for all three monitors; (3) event streams hung the HTTP
server's graceful shutdown for the whole timeout: the server's `BaseContext` is cancelled before `Shutdown`, which
ends them. The same run showed WSLg exposes **three** monitors that mirror the host layout, that Wayland ignores window
positioning (the output stayed at 0,0 at full primary size), and again that WebKitGTK plays VP9 but not H.264, HEVC or AV1.

