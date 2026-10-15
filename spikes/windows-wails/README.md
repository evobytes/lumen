> **ARCHIVED SPIKE.** Kept as provenance: the evidence behind decisions 005-008 and 017; production code was ported from it into `internal/`. Not built by CI. See `docs/decisions.md` (decision 022) and the `spikes` skill.

# Lumen Windows-first spike (Wails v3 beta.29)

Throw-away spike that tests the riskiest assumptions of the Go + Wails v3 recommendation on a
real Windows 11 machine with two monitors. It is not the product.

What it is: a Go process with an HTTP API (`net/http`, plus SSE), a **control** window and a
frameless, always-on-top **output** window. Both windows just load pages served by the Go
process (`/control`, `/output`); the output page is a pure function of server state.
Window handling sits behind the `Host` interface (`host.go`); Win32 specifics are in
`native_windows.go` (CGO-free, via `golang.org/x/sys/windows`).

## Build and run (from WSL2 or Linux; pure Go cross-compile)

```sh
./build-windows.sh          # bin/lumen-spike-console.exe   (logs visible)
./build-windows.sh gui      # bin/lumen-spike.exe           (-H windowsgui)
```

Copy the exe somewhere on the Windows side and run it:

```sh
lumen-spike-console.exe                                  # control window + floating output
lumen-spike-console.exe -selftest -port 0 -report r.json # automated checks, writes a JSON report
lumen-spike-console.exe -demo auto -mode win32 -hold 20s # park the output on the first non-primary monitor
```

Requires the WebView2 runtime (preinstalled on Windows 11). The API is on `127.0.0.1:8765` by
default: `GET /api/v1/status|screens|windows|report`, `POST /api/v1/next|prev|blank`,
`PUT /api/v1/output {"screen":"\\\\.\\DISPLAY2","mode":"win32"|"wails"}` (`screen:null` = floating).

## Results (2026-10-13; Windows 11 build 26200, WebView2 154, 2 monitors)

Monitors: `DISPLAY1` 2256x1504 at (0,0), `DISPLAY2` 2560x1600 at **(-2560,-90)** (negative origin).

| Check | Result |
| --- | --- |
| Output window frameless, always-on-top, hidden from taskbar | **Pass** (`WS_EX_TOPMOST`, `WS_EX_TOOLWINDOW`) |
| Output is a Win32 *owned* window of the control window ("floating child") | **Pass**, via `SetWindowLongPtr(GWLP_HWNDPARENT)`; Wails has no public API for this |
| Cover each monitor exactly (Win32 `SetWindowPos`, and Wails `SetBounds`) | **Pass**: 12/12 exact and stable over 2 s, including the negative-origin monitor |
| Visual check (screenshot) of output on the external monitor | **Pass**: page renders and fills the monitor |
| H.264+AAC, HEVC, AV1, VP9+Opus, VP9-with-alpha playback in WebView2 | **All pass**; alpha decoded (sample alpha 127 for a 50% clip) |
| `HTMLMediaElement.setSinkId` (audio output routing) | **Present** (function); 1 output device enumerated |
| Windows build | Pure Go, `CGO_ENABLED=0`, 11.9 MB exe, cross-compiled from WSL |

### Open issues found (not resolved)

1. **Cold start is slow: both windows took 5-9 s to request their first page** (3 launches). Not
   yet attributed to Wails vs WebView2 vs this busy machine (Defender real-time on, 58 other
   WebView2 processes). A bare-Edge baseline failed to launch, so there is no comparison yet.
2. **First placement after startup was 24 px off once** (win32 mode, primary monitor, y=-24),
   in 1 of 3 cold starts and never in the repeated 12-placement run. Likely a race with Wails'
   initial show; re-applying the position after the page loads should sidestep it. Not proven.
3. **DPI disagreement:** Win32 (`GetDpiForMonitor`, `GetScaleFactorForMonitor`) reports 100% on
   both monitors in every awareness mode, and so does Wails (`ScaleFactor` 1), but WebView2
   reports `devicePixelRatio` 1.25 on both. Cause unknown. **Mixed-DPI placement is therefore
   untested**, because both monitors are at 100% by the Win32 APIs. Needs a deliberate manual
   test with different scales.

### Not tested

Monitor hot-plug, the `-H windowsgui` build, focus behaviour of the output window, interaction
with PowerPoint's fullscreen window, NDI, GStreamer `webview2src` alpha, installers/signing,
macOS and Linux.
