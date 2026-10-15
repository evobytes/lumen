---
name: verify
description: Run the right verification for the change you made (unit, lint, docs, Windows build and self-test, packaging) and report results honestly. Use before saying any work is done.
---

# /verify

Pick the gates by what changed, run them, and **report what actually happened**: paste the tail of failing output; never claim a check passed that you did not run; say what you skipped and why.

| Changed | Run |
|---|---|
| any Go, config, docs | `make ci` (fmt, vet and lint for Windows and the host, tidy, docs-check, tests, vuln, build) |
| `internal/win32`, `wailshost`, `app`, anything `//go:build windows` | `make ci`, then say it was only **compiled** for Windows unless you ran `make selftest` on a Windows host |
| the output page, placement, DPI, owner, topmost | `make selftest` on Windows, read `dist/selftest/report.json`, and look at a `scripts/windows-screenshot` |
| `os/`, `packaging/`, `scripts/build-*` | build the package you touched (`make deb` / `win-zip` / `installer` / `macos-app`) and inspect it (`dpkg-deb -I`/`-c`, `unzip -l`) |
| a release candidate | `make release-dry` |

## Environment notes
- Agent sessions: `source scripts/agent-env` first so caches stay in the repo. After any `cd`, return to the repo root before `make`.
- On Windows `make ci` runs tests without `-race` (no cgo); the Linux CI job has the race detector. `make installer` needs Inno Setup; `make deb` needs Linux and the GTK/WebKit dev packages; `make macos-app` needs macOS.
- Passing tests prove nothing about the real windows. List placement, DPI, ownership, hot-plug, focus and video under "not verified" unless you exercised them on a real machine.
- A black or grey monitor in a screenshot can mean the page has not loaded yet (cold start 5-9 s on the first host): wait and re-check before concluding anything.
- Never kill processes by broad pattern (`pkill -f` matches your own shell); only ones you started.

## Report format
1. What you ran and the result of each (pass/fail/skipped, and why).
2. Failures verbatim, root cause, fix, re-run result.
3. What is **not** verified (Windows behavior, macOS, installer, the first CI run of a new workflow).
