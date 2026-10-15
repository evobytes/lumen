# Architecture

Lumen is one Go process: an HTTP server that owns the state and serves web pages, plus two Wails windows that
load those pages. Windows is the primary platform (decision 003).

```
                      ┌─────────────────────── one Go process ───────────────────────┐
 phone / script ─────▶│ internal/api   REST + SSE under /api/v1, pages, problem+json   │
 OBS / CasparCG ─────▶│        │                                                       │
 (later)              │        ▼                                                       │
                      │ internal/state  slides, position, blank; subscribers           │
                      │                                                                │
                      │ internal/app    wiring, config, self-test/demo modes           │
                      │        │                                                       │
                      │        ▼                                                       │
                      │ internal/host   Host interface (physical px, virtual desktop)  │
                      │   ├─ wailshost  Wails v3 windows (+ Win32 on Windows)          │
                      │   └─ hosttest   in-memory fake                                 │
                      │ internal/win32  EnumDisplayMonitors, SetWindowPos, owner (CGO-free)
                      └───────────────────────────────┬───────────────────────────────┘
                              control window ◀────────┴────────▶ output window
                              loads /control                     loads /output (frameless,
                                                                 topmost, owned by control)
```

## Principles

1. **State is the single source of truth.** `internal/state` holds the slides, the position and the blank flag.
   Every renderer re-reads it when notified (server-sent events). Nothing renders from its own copy.
2. **The output is a URL.** The Wails output window is one renderer of `/output` (decision 004).
3. **The windowing toolkit is behind an interface** (`internal/host`, decision 005). Only `wailshost` and `app` import Wails.
4. **Portable core, thin platform edge.** Everything except `wailshost`, `win32`, `app` and `cmd` compiles and is tested on any
   host without a C toolchain. Windows-only code lives behind `//go:build windows` with an inert twin.
5. **Verify with the OS, not the toolkit.** Window facts come from Win32 (rectangles, DPI, owner, topmost), because
   toolkit APIs can disagree with the OS (see the DPI note in `docs/roadmap.md`).

## Threading

The HTTP server runs on its own goroutines. Window operations that Wails requires on its main thread go through
`application.InvokeSync`. State is guarded by a mutex; subscribers get a coalescing channel (a slow reader sees at most one
pending signal and re-reads). `host.Host` methods take a `context.Context` and are safe to call from handlers.

## Window model (Windows)

- Control window: ordinary top-level window loading `/control`.
- Output window: frameless, `AlwaysOnTop`, `HiddenOnTaskbar`, Wails' frameless shadow/rounded-corner decorations
  disabled, a Win32 **owned** window of the control window. Placed by `SetWindowPos` onto a monitor rectangle in
  physical pixels (negative origins are normal), verified, retried, and re-applied when its page finishes loading.
- DPI: the executable embeds a per-monitor-v2 manifest (`packaging/windows/lumen.exe.manifest`); without it every monitor
  measurement is wrong.

## Packages

<!-- BEGIN GENERATED: packages (scripts/docgen; edit the source, run `make docs`) -->
| Package | Responsibility (its package doc) | Internal imports |
|---|---|---|
| `cmd/lumen` | is the Lumen presentation projector: a control window, a floating output window and an embedded web API | `app`, `config`, `host`, `logging`, `version` |
| `internal/api` | serves the web API (/api/v1), the control and output pages and the server-sent event stream that keeps every renderer in step with the state | `host`, `state`, `ui`, `version` |
| `internal/app` | wires Lumen together: the web API server, the control and output windows (Wails), and the optional self-test and demo modes | `api`, `config`, `host`, `host/wailshost`, `selftest`, `state` |
| `internal/config` | loads and validates Lumen configuration | — |
| `internal/host` | defines the window-management seam | — |
| `internal/host/hosttest` | provides an in-memory host.Host for tests | `host` |
| `internal/host/wailshost` | implements host.Host with Wails v3 windows | `host`, `win32` |
| `internal/logging` | builds the application's slog.Logger | — |
| `internal/selftest` | drives the real, running application through its own host seam and web page and writes a machine-readable report | `api`, `host` |
| `internal/state` | is the single source of truth for what the audience sees: the slides, the current position and the blank flag | — |
| `internal/ui` | embeds the web pages the application's windows (and any browser, OBS browser source or phone) load: the control page and the output page | — |
| `internal/version` | exposes build metadata injected with -ldflags -X | — |
| `internal/win32` | is the thin Windows edge: ground-truth monitor and window facts, plus the few Win32 operations the windowing toolkit lacks (window ownership, exact placement) | `host` |
<!-- END GENERATED: packages -->

## Data flow of a "next" request

`POST /api/v1/next` → `api` handler → `state.Next()` → `notify()` → every `GET /api/v1/events` stream emits `state` → the
output and control pages call `GET /api/v1/status` and re-render. The same path serves a phone, a script and the keyboard.

## Packaging and platforms

| Platform | Binary | Package | Notes |
|---|---|---|---|
| Windows 11 amd64 | CGO-free, GUI subsystem, cross-compiled from any host | zip, Inno Setup installer | WebView2 runtime (preinstalled on Windows 11) |
| Ubuntu 24.04 amd64/arm64 | CGO, built natively | `.deb` | GTK4, WebKitGTK 6; depends on `gstreamer1.0-libav` for video |
| macOS (Apple Silicon) | CGO, built natively | zip of an unsigned `.app` | secondary; built only by the CI macOS job |

Files installed on disk are listed under `os/` (the target filesystem tree) and `packaging/`; see the `packaging` skill.

## Security

Loopback by default; a non-loopback listen address requires `auth.token`. No inline script or style (strict CSP). Request
bodies are bounded and strict. The diagnostics store (`/api/v1/report`) accepts only bounded JSON under the token. No request
field names an executable, path or URL: window placement takes a monitor ID the OS reported. See the `security` skill.
