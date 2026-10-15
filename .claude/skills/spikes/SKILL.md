---
name: spikes
description: What the archived spikes under spikes/ are, what they proved, and how production code was ported from them. Use when you need the evidence behind a decision or want to reuse spike code.
---

# Spikes

Both spikes are archived: provenance, not product. They are ignored by CI (the Go one has its own `go.mod`; the Python one is not Go).

## `spikes/windows-wails` (Go, Wails v3 beta.29)
A flat `main` package that proved the Windows window model on a real two-monitor Windows 11 machine. Its `README.md` has the build and run commands and the **verified results table** (owner/topmost/tool window, 12/12 exact placements incl. a negative-origin monitor, all video codecs in WebView2, `setSinkId`) and the three open issues (cold start 5-9 s, a 24 px first-placement offset, the DPI disagreement). Build it with its own `build-windows.sh` (cross-compile from WSL, needs `wails3` on PATH for the resource file).
Ported into production: state and SSE (`internal/state`, `internal/api`), the `Host` seam (`internal/host`), Win32 code (`internal/win32`), the placement logic with verify-and-retry (`wailshost`), the pages (`internal/ui`), the self-test (`internal/selftest`). Not ported: the throwaway `Reports` plumbing details and the spike's flat layout.

## `spikes/python-pyside6` (Python, PySide6 + FastAPI)
The first prototype (control window, floating output window, web API in a thread bridged to Qt, 33 tests, PyInstaller packaging with `--self-test`, CI). Superseded (decision 001). Useful as a reference for the API shape and for how a Qt-native version would differ. Safe to delete once nobody needs it.

## Using a spike
Read its README first. Do not import from it. If you extend a spike to answer a question, put the answer in `docs/` (the roadmap or a decision) and port code into `internal/`, with tests.
