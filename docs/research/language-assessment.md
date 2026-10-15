# Language & toolkit assessment: Lumen (OpenLP replacement)

Date: 2026-10-13. Status: recommendation, pending a short spike (section 7).

## 1. Summary

**Recommendation: Go backend + Wails v3 (web UI in HTML/CSS/JS), gated by a 1-2 day spike.**

- It matches your skills: Go for the core, API and data; web for everything visual. You have
  excellent web and good Go, so you can read, debug and own all of it.
- It fits the product. The output window is "styled text over image/video" and a lyrics
  projector is mostly that. OpenLP itself renders its output as HTML.
- The same HTML/CSS drives the control UI, the output window and the phone remote page.
- **The risk is real:** Wails v3 is still **beta** (`v3.0.0-beta.29`, released today, weekly
  cadence) and has open multi-monitor bugs. That is why the spike comes first.
- **Fallbacks:** C# + Avalonia 12 (native rendering, verified to cross-publish from one box) if
  the webview approach fails the spike. Rust + Tauri 2 if you will accept Rust and want a
  stable webview shell.

One expectation to reset: **"static compiled binary" is only fully true for Windows and for the
non-GUI core.** Any GUI toolkit needs native libraries on macOS and Linux (section 4).

**Targets:** Ubuntu 24.04+, Windows 11, latest macOS on Apple Silicon only. No legacy GTK3 stack,
no Intel or universal macOS builds.

**Update (same day):** PowerPoint control, media playback, NDI and SDI were assessed separately
in [integrations-assessment.md](integrations-assessment.md). Net effect: the Wails lead narrows
but holds. Webview video has a verified codec gap on Linux, and alpha frames for fill-and-key need
a Chromium renderer, OBS or CasparCG. Those have sidecar or outsourcing answers that don't change
the language. That document adds four tests (9-12) to the spike in section 7.

### Eliminated early (details in section 5)

| Option | Why it's out |
| --- | --- |
| Swift | Apple-only in practice. Cross-platform Swift GUI is immature and reportedly lacks multi-window. |
| Go + Gio | Verified in source (v0.10.3): no way to position a window or pick a monitor. No video. |
| Go + Fyne | Fyne 2.8 gained position / always-on-top / "fullscreen on secondary", but there is no general frameless window, no video, and limited text styling. |
| Wails v2 | Stable, but single window only. |
| Flutter | Desktop multi-window is still an open umbrella issue (#30701), so it is not shipped. |

**Windows spike result (same day):** on a real Windows 11 two-monitor machine, a Wails v3 beta.29
spike (CGO-free, 11.9 MB) passed the window and monitor checks (frameless, always-on-top, owned
child window, exact placement on a negative-origin monitor) and the video checks (all codecs plus
alpha, `setSinkId`). Open items: 5-9 s cold start, one intermittent 24 px offset on first
placement, an unexplained DPI disagreement, and untested mixed DPI. Details in
`spikes/windows-wails/README.md`.

## 2. What the project actually needs

These requirements drive the scoring. Taken from your brief plus what an OpenLP replacement
must do.

1. Two windows: a control window (parent) plus a borderless, always-on-top output window
   (floating child).
2. Two or more monitors: enumerate screens, place the output on a chosen one, handle mixed DPI
   and hot-plug.
3. An integrated web API (also serving a phone remote page).
4. GitHub CI that builds Linux, Windows and macOS.
5. **Rich output rendering:** large auto-fit text, outlines and shadows, transitions, image and
   **video backgrounds**. This is the hard part of the product and is easy to underestimate.
6. Later: song database (SQLite), PDF/PowerPoint import, audio, a stage/confidence display.
7. Maintainable by you. Compiled, easy to package, no "install this runtime first".

## 3. Candidates compared

Scores are my judgement, 1 (poor) to 5 (excellent). Evidence behind each is in section 5.

| Criterion (weight) | Go + Wails v3 | Rust + Tauri 2 | C# + Avalonia 12 | Python + PySide6 (current) | Go + Fyne 2.8 | Go + Gio | Electron |
| --- | :-: | :-: | :-: | :-: | :-: | :-: | :-: |
| Fit with your skills (x3) | 5 | 2 | 2 | 3 | 5 | 5 | 4 |
| Floating/frameless/on-top window (x3) | 4 | 4 | 5 | 4 | 2 | 3 | 5 |
| Multi-monitor placement (x3) | 3 | 4 | 5 | 4 | 2 | 1 | 5 |
| Rich text + video output (x3) | 4 | 4 | 3 | 4 | 1 | 1 | 5 |
| Embedded web API (x2) | 5 | 4 | 5 | 4 | 5 | 5 | 4 |
| Cross-platform CI / packaging (x2) | 4 | 4 | 5 | 2 | 3 | 3 | 4 |
| Binary size / "compiled" feel (x1) | 4 | 5 | 2 | 1 | 4 | 4 | 1 |
| Maturity / stability (x2) | 2 | 5 | 4 | 5 | 4 | 3 | 5 |
| **Weighted total (max 95)** | **74** | 73 | 75 | 68 | 58 | 56 | 84 |

Reading the table:

- **Electron scores highest (84)** on raw capability. It is excluded by your stated preference
  (compiled, lean, no JS runtime; it ships ~100-250 MB), which the single "size" row
  under-weights. If you ever drop that preference, Electron is the lowest-risk way to get
  identical rendering and mature multi-monitor control on every OS.
- **Wails (74), Avalonia (75) and Tauri (73) are a statistical tie.** The scores are my
  judgement and the gaps are within noise, so the totals do not pick a winner. The recommendation
  rests on the qualitative points: Wails is the only one that uses Go *and* web, your two
  strongest skills, and it matches the product shape.
- That choice is sensitive to how much you value owning the code. If "fit with your skills"
  were weighted x1 instead of x3, Avalonia would lead (71) over Tauri (69) and Wails (64).
- PySide6 scores lower mainly on packaging and size, the pain you described, not on capability.

## 4. "Compiled and static", precisely

| Stack | Windows | macOS | Linux | Notes |
| --- | --- | --- | --- | --- |
| Go core only (`CGO_ENABLED=0`) | static | static | static | Verified: a Go HTTP server cross-compiles to 3 OSes in seconds, ~6 MB each, Linux binary "statically linked". |
| Wails v3 | pure Go, needs WebView2 (preinstalled on nearly all Win 10/11) | CGO + WKWebView (system) | CGO + **GTK4 + WebKitGTK 6** on the user's machine (GTK3 stack still supported) | Per Wails docs. Needs native build or Docker per OS for mac/Linux. |
| Tauri 2 | WebView2 | WKWebView | WebKitGTK | Same webview dependencies; Rust compile. |
| Avalonia 12 | self-contained | self-contained | self-contained | Verified below. Bundles .NET runtime and Skia, so big. |
| PySide6 (current) | PyInstaller bundle | PyInstaller bundle | PyInstaller bundle | 66 MB tar.gz on Linux, plus the packaging fragility you dislike. |

Verified here: an Avalonia 12.1.4 spike built on this Linux machine and cross-published
(`dotnet publish -r <rid> --self-contained -p:PublishSingleFile=true`) to **linux-x64 (97 MB),
win-x64 (102 MB), osx-arm64 (109 MB)** with no per-OS toolchain. These are untrimmed; trimming
or NativeAOT (supported by Avalonia) would reduce them, but I did not measure that.

## 5. Option notes

### 5.1 Go + Wails v3 (recommended, conditionally)

Wails embeds a system webview in native windows and gives Go a bindings layer to the UI.

Verified in the Wails v3 source on `master`:

- `WebviewWindowOptions` has `AlwaysOnTop`, `Frameless`, `X`, `Y`, `InitialPosition`, `Hidden`
  and `Screen *Screen` (target screen for initial placement).
- A `ScreenManager` with `GetAll`, `GetPrimary`, `GetByID`, `GetByIndex`, with `ScaleFactor`,
  `Bounds` and `WorkArea` per screen.
- Multiple windows can be created at runtime (documented as a v3 core feature).
- Windows builds are pure Go (no CGO). macOS and Linux need CGO.
- Linux code notes `gdk_monitor_is_primary` is unreliable on Wayland (always false), so primary
  detection has Wayland workarounds.

Pros:

- One language (Go) for server, API, DB and window orchestration, and one UI skill (web).
- The output window, control UI and phone remote can share components and a design system.
- Video, CSS typography, text shadow and transitions come free from the browser engine.
- Trivial to embed an HTTP API (`net/http`, same process).
- Modern Go toolchain; pure-Go SQLite (`modernc.org/sqlite`) keeps Windows CGO-free.

Cons / risks:

- **Beta.** `v3.0.0-beta.29` (2026-10-13), 379 open issues. Expect API churn; pin the version.
- **Multi-monitor bugs are open right now:** #5409 (macOS: `Screen.Bounds` X/Y wrong on
  non-primary screens with mixed DPI) and #4003 (Linux: moving a window relative to its
  position on multi-monitor setups). These touch our core requirement.
- Webviews differ per OS, so rendering and **video codec support vary** (notably WebKitGTK
  depends on the system's GStreamer plugins). This is a testing burden, and it is unverified
  here.
- Linux users need GTK4 and WebKitGTK 6 installed. Wayland cannot position windows (a protocol
  restriction affecting every toolkit, including Qt).
- Builds are unsigned unless CI signs them.

Mitigation: hide window management behind our own small Go interface so Wails could be
swapped, and build the output as a **plain web page served by our own Go server** (section 6).

### 5.2 Rust + Tauri 2

Most mature webview shell (111k stars, `tauri-v2.12.3` on 2026-10-10, stable 2.x). Mature
multi-window, monitor and always-on-top APIs (I did not re-verify these in source this session).
Same webview caveats as Wails. The cost is Rust: I can write it, but you would be maintaining a
language you have not used. Pick this if you want a stable shell and will accept Rust. A hybrid
(Tauri shell + Go sidecar API) adds complexity for little gain.

### 5.3 C# + Avalonia 12

Verified with a working spike (`WindowDecorations.None`, `Topmost`, `ShowInTaskbar`, `Owner` via
`Show(parent)`, `Screens.All`, `Screens.Primary`, `Screens.Changed`, per-screen `Bounds` and
`Scaling`). It compiled and the Linux build ran under WSLg.

Pros: genuinely native windowing and screen API, one consistent renderer (Skia), no webview
differences, stable (12.1.4 on 2026-10-09), easiest cross-publishing of the lot. Embedded API via
ASP.NET Core Minimal APIs is first-class.

Cons: new language and XAML for you; rich text effects (outline, shadow, auto-fit) and **video**
must be built (LibVLCSharp is the usual route, and it bundles libvlc); binaries are ~100 MB; an
API rename already bit me in the spike (`SystemDecorations` -> `WindowDecorations`), which hints
at the cost of a recent major version. Your web skills would only be reused for the remote page.

### 5.4 Python + PySide6 (what exists now)

Technically the most capable (QtMultimedia, QtWebEngine, mature multi-screen), and the prototype
already works. The problem you named is distribution: PyInstaller bundles, per-OS quirks, a
heavy dependency tree. If the Go/web route fails the spike, staying on Qt is a rational choice,
and **Qt for Go** (`miqt`, 791 stars) is a poor trade (CGO-heavy, slow builds, small community).

### 5.5 Pure-Go native toolkits

- **Fyne 2.8.2** (verified in source): `desktop.Window` offers `RequestPosition`,
  `RequestAlwaysOnTop` and `RequestFullScreenSecondary` (both documented as best-effort and
  ignored on Wayland). But no general frameless window (only a splash window), no video, no
  way to choose a *specific* monitor, and limited text styling.
- **Gio v0.10.3** (verified in source): multiple windows work on desktop; `TopMost` and
  `Decorated` exist; **no position or monitor option**, no video, and an immediate-mode
  paradigm with a smaller widget set.

Both are fine for forms and tools, but weak for a projection output window.

### 5.6 Swift

SwiftUI/AppKit are macOS-only. The cross-platform options are community projects
(SwiftCrossUI ~1.8k stars; a search summary says its notes list no multi-window support, which I
did not verify in source). You would be writing separate UIs per OS or accepting gaps. Swift is
the right choice only for a **macOS-only** product. Not recommended here.

### 5.7 Others, briefly

| Option | Verdict |
| --- | --- |
| Flutter | Desktop multi-window still an open umbrella (#30701, #142845). Wait. |
| Electron | Best capability and consistency, but heavy and not compiled. Reasonable only if consistency beats your preferences. |
| Qt/C++ | What OpenLP-class apps use. Most capable, hardest toolchain and licensing. |
| Rust + Slint/egui/iced | Rust learning curve and a young ecosystem for rich text/video. |
| Kotlin Compose Desktop | JVM runtime, jpackage. No advantage over Avalonia for you. |
| Go + system browser (kiosk) | See section 6. A good *fallback mode*, a poor primary. |

## 6. Architecture that limits the risk (applies to the Go path)

```
┌────────────────────────── one Go process ──────────────────────────┐
│  core: model, SQLite, import, media    HTTP server (net/http)       │
│        │                                  ├─ /api/v1/*  (JSON)      │
│        │                                  ├─ /output    (HTML page) │
│        ▼                                  ├─ /control   (HTML UI)   │
│  window manager (Wails v3, thin interface)└─ /remote    (phone UI)  │
│   ├─ control window  → loads /control                               │
│   └─ output window   → loads /output  (frameless, on-top, on screen N)
└─────────────────────────────────────────────────────────────────────┘
```

- The output is a URL, not a Wails-specific component. The same page works in OBS "browser
  source", a browser on a second PC or a Raspberry Pi, or a kiosk Chromium. If Wails
  disappoints, only the thin window layer changes.
- The API is the single source of truth, and the UIs are just clients of it. That also gives
  you good tests without a GUI (plain Go `httptest`).
- The core stays CGO-free, so tests and the API run anywhere, quickly.

## 7. The spike (do this before committing)

Time-boxed to 1-2 days. Pass/fail criteria, each tested on **Windows, macOS and Linux (X11 and
Wayland)**, driven by a GitHub Actions matrix where possible:

| # | Test | Why |
| --- | --- | --- |
| 1 | Two windows: control + borderless, always-on-top output | Core requirement |
| 2 | Output placed on monitor 2 and covering it, on every OS | Core requirement |
| 3 | Mixed-DPI second monitor places correctly (cf. #5409) | Known bug area |
| 4 | Unplug/replug a monitor: output falls back and recovers | Real-world use |
| 5 | H.264 video background plays smoothly in each webview | Biggest hidden risk |
| 6 | Go HTTP API + remote page served in-process, calling into windows | Architecture |
| 7 | `wails3 build` on all three runners, artifacts uploaded | CI requirement |
| 8 | Window code behind our own interface (can it be swapped?) | Exit strategy |

**Decision rule:** if 2, 3 or 5 fail on any OS with no workaround, switch to Avalonia (native)
or Tauri (stable webview, if Rust is acceptable). If Linux/Wayland alone fails, document X11
and move on, since every toolkit has the same limitation.

I could not run the Wails spike here: this WSL machine lacks the WebKitGTK dev libraries and
sudo. CI runners can install them, so the spike should run in GitHub Actions.

## 8. What carries over from the Python prototype

- The REST API shape (`/api/v1/status`, `next`, `prev`, `slides/{i}/show`, `blank`, `screens`,
  `output`) and the token scheme port directly.
- The phone remote page (plain HTML/JS) is reusable as-is.
- The `Slide`/`Presentation` model and its tests are small and easy to re-express in Go.
- The `--self-test` idea (boot the real app, drive it via its API) is worth keeping; it is how
  the packaged builds get verified in CI.
- CI layout (lint, test matrix, build matrix, tag-driven release) carries over with different
  commands.

I would keep the Python prototype in the repo until the spike passes, then remove it.

## 9. Sources and confidence

**Verified directly this session:**

- Wails v3 source (`webview_window_options.go`, `screenmanager.go`, `screen_linux.go`) and
  docs (cross-platform build, installation) via the GitHub API; release list
  (`v3.0.0-beta.29`, 2026-10-13); open issues #5409, #4003.
- Fyne v2.8.2 and Gio v0.10.3 module sources (window interfaces and options).
- Avalonia 12.1.4: compiled, ran (Linux, WSLg) and cross-published a multi-window,
  multi-monitor spike; release dates via GitHub.
- A Go 1.26 `CGO_ENABLED=0` HTTP server cross-compiled to 3 OSes.
- Repository activity (stars, latest releases) via the GitHub API.

**From web search only (not verified in source):** Wails v3 beta announcement and status;
Tauri 2 feature details beyond the release list; SwiftCrossUI's gaps; Flutter desktop
multi-window status (confirmed as open issues, not as a feature statement).

**Not tested at all:** anything running on Windows or macOS; video codec behaviour in any
webview; real dual-monitor placement on any OS; Wayland behaviour of Wails; trimmed or
NativeAOT Avalonia sizes; Wails binary size.

Primary links:

- Wails v3: <https://v3.wails.io/> and <https://github.com/wailsapp/wails>
- Tauri: <https://v2.tauri.app/> and <https://github.com/tauri-apps/tauri>
- Avalonia: <https://docs.avaloniaui.net/> and <https://github.com/AvaloniaUI/Avalonia>
- Fyne: <https://docs.fyne.io/> · Gio: <https://gioui.org/>
- SwiftCrossUI: <https://github.com/stackotter/swift-cross-ui>
