---
name: windows-host-testing
description: How to run, screenshot and self-test lumen on a real Windows host, from WSL2 interop or natively, within the write-only-inside-the-repo rule. Use when verifying window, monitor, DPI or output behavior.
---

# Testing on a real Windows host

Unit tests prove nothing about windows. A Windows host is the only place placement, ownership, DPI and codecs are real. `docs/windows-dev.md` has the setup; this is the procedure.

## Check you can reach it
```
cmd.exe /c ver            # a Windows version, not "Exec format error" (see docs/windows-dev.md to fix interop)
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w scripts/windows-display-info.ps1)"
```
The second is read-only: monitors with per-monitor DPI, WebView2, Office, NDI, tools. Run it first and note how many monitors exist: placement results are only meaningful with two or more.

## The rule
Session rule (decision 012): write only inside the repo. Running an exe from a Windows path, and the WebView2 profile the app creates, write outside it. So `scripts/windows-host-selftest` refuses without `LUMEN_ALLOW_HOST_WRITE=1`, and you set that only after the owner has agreed. It also puts real windows on their desktop for about half a minute: say so first.
- On a Windows host itself (Git Bash) none of this applies: `make selftest` is the command.

## Procedure
1. `make ci` (cross-compiles the Windows exe).
2. `make selftest` on Windows, or from WSL: `LUMEN_ALLOW_HOST_WRITE=1 scripts/windows-host-selftest`. Report: `dist/selftest/report.json` (or `windows-host-report.json`). Read `summary.pass`, `placementsExact`, each placement's `samples`, `outputOwnedBy`, `topmost`, `toolWindow`, and the `media` block (codecs, `setSinkId`, `dpr`).
3. `scripts/windows-screenshot dist/selftest/desktop.png` and **look at it**. To see the output on a monitor: `lumen-console -demo auto -placement win32 -hold 25s` (debug overlay shows size, dpr, visibility), then screenshot after the page loaded (wait about 12 s).
4. For DPI: set the monitors to different scales (a person does this), re-run, compare the OS rectangle with the page's `inner` and `dpr`.

## Reading results honestly
- A black or grey monitor can mean the page has not loaded (cold start). Check the app's request log or wait.
- `exact: false` with a small offset on the first placement is the known race (roadmap issue 2); repeated runs matter more than one.
- Never claim "works on Windows" from a compile; say what ran, where, with how many monitors.
- Clean up only processes you started (`taskkill.exe /PID <pid>`), and delete nothing outside the repo.
