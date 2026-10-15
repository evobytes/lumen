---
name: dev-loop
description: Fast local development for lumen: make dev, tool bootstrap, working in WSL2 with Windows interop or natively on Windows with Git Bash. Use when running or changing the dev workflow.
---

# Dev loop

Windows is the primary platform, so it is where behavior is judged; `docs/windows-dev.md` is the full guide (WSL2 interop, native Windows, Linux) and records the first dev host.

## Daily
- `make bootstrap` once per clone (pinned tools into `.tools/bin`, plus the pre-commit hook).
- `make dev` runs `go run ./cmd/lumen -config dev/config.toml` (port 18765, debug logs). Open the control page at that port; the windows open on your desktop.
- Prefer `go test ./internal/<pkg>` for the package you edit; run `make ci` before finishing and say which of its steps could not run on this host.
- `make selftest` drives the real app (placement on every monitor, media probe) and writes `dist/selftest/report.json`.
- The page assets under `internal/ui/assets` are embedded: restart to see changes (or open the page in a browser against a running instance).

## Where Windows runs when you are in WSL2
Cross-build with `scripts/build-binary windows amd64 [console]` (pure Go) and run the exe through interop; see `windows-host-testing` for the safe way to do that (it writes outside the repo, so it needs an explicit opt-in).

## Rules
- Run `make` from Git Bash on Windows, never PowerShell or cmd (the Makefile sets `SHELL := bash`).
- Agent sessions use `source scripts/agent-env` so every cache and tool stays inside the repo (decision 012).
- Never kill processes by broad pattern; only ones you started, by PID or exact name.
- After any `cd`, return to the repo root before running `make`.
