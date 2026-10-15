---
name: wails-windows
description: Wails v3 and Win32 specifics for lumen's windows: owner windows, exact placement, DPI manifest, frameless output, load-finished re-apply, build tags and known toolkit gaps. Use when touching internal/host/wailshost, internal/win32 or internal/app.
---

# Wails v3 on Windows

Pinned at `v3.0.0-beta.29` (beta: read release notes before bumping; decision 025).

## The window model
- **Control window**: normal, loads `/control`. **Output window**: `Frameless`, `AlwaysOnTop`, `Hidden` at creation, `HiddenOnTaskbar`, `DisableFramelessWindowDecorations` (no shadow or rounded corners that would clip a monitor), black background, loads `/output`.
- Wails has **no owner/parent option**. `wailshost.Attach` sets `GWLP_HWNDPARENT` with `SetWindowLongPtr` (`win32.SetOwner`) so the output stays above, minimises and dies with the control window and has no taskbar button. Verified: `outputOwnedBy: control`.
- Wails' `NativeWindow()` returns the HWND on Windows (a GTK pointer on Linux, so `win32` functions are only called when `runtime.GOOS == "windows"`).

## Placement
- Coordinates are **physical pixels** in virtual-desktop space; monitors can have negative origins (the first host's external monitor sits at (-2560,-90)). `win32.Monitors` (EnumDisplayMonitors + GetMonitorInfo + GetDpiForMonitor) is the truth.
- `output.placement = win32`: `SetWindowPos` with `HWND_TOPMOST` and `SWP_NOACTIVATE`, then `settle`: read the rectangle back after 120 ms and retry up to 4 times; failure to stick is an error, not a silent success. `toolkit`: match the Wails screen by physical bounds and `SetBounds` its logical bounds.
- `Reapply` repeats the last placement when the output page's load-finished event fires (`loadFinishedEvent()` per OS, because Wails names it differently: `Windows.WebViewNavigationCompleted`, `Mac.WebViewDidFinishNavigation`, `Linux.WindowLoadFinished`). This is the mitigation for the one 24 px first-placement offset seen in the spike; unproven.
- Window calls that Wails needs on its main thread go through `application.InvokeSync`; do not call them from inside an event handler that is itself on that thread without checking for deadlock.

## DPI
The exe **must** embed a per-monitor-v2 manifest (`packaging/windows/lumen.exe.manifest`, generated into a `.syso` by `scripts/build-binary`), or the process is DPI-unaware and rectangles are virtualised. Open mystery: Win32 and Wails report 100% while WebView2 reports `devicePixelRatio` 1.25 on the first host. Layout uses `vmin`, so the page does not depend on it, but never trust one source: compare the OS rectangle with the page's `innerWidth` times `devicePixelRatio`. Mixed-DPI placement is untested.

## Known gaps and traps
- Cold start 5-9 s before the first page request (first host); the output is blank until then: do not mistake it for a failure.
- Windows GUI-subsystem builds have no stdout: use the console variant for logs.
- Wails Linux/macOS code needs CGO; Windows does not (decision 017). Do not add a Windows CGO dependency.
- `wails3 generate syso` is the only use of the Wails CLI (icon, version info, manifest). `scripts/tools.env` pins its version with `go.mod`'s.
- Open upstream issues that touch us: wailsapp/wails #5409 (macOS mixed-DPI screen bounds), #4003 (Linux relative moves on multi-monitor), #6088 (WebView2 transparency plus click-through).
