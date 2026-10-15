---
name: testing
description: How to write and run tests in lumen: table-driven unit tests, handler tests, the fake window host, the self-test, Windows-only code, coverage. Use whenever adding or changing behavior.
---

# Testing

## Commands
| Command | Scope |
|---|---|
| `make test` | every package, `-race -shuffle=on -count=1` (needs cgo: gcc on Windows); `make ci` uses it on Linux and macOS |
| `make test-windows` | the same suite without `-race`; `make ci` uses it on Windows |
| `make cover` | coverage profile to `dist/coverage.out` plus HTML |
| `go test ./internal/api -run TestAuth -v` | a single test |
| `make selftest` | the real app on this machine: placement, owner/topmost, media probe (opens windows) |

## Principles
- Stdlib `testing` and `net/http/httptest`; no assertion frameworks, no mock generators. Table-driven with named subtests; `t.Parallel()` in tests and subtests.
- **Test behavior through public APIs** (`package x_test`); white-box only when needed (`internal/config/docs_test.go` reads `envKeys`).
- **Don't mock what we own.** Fake only external boundaries: the windows (`Fake` in `internal/host/hosttest`), time where it matters, the toolkit.
- `t.Helper()`, `t.Cleanup()`, `t.TempDir()`. No `time.Sleep` for synchronisation (the self-test's settle delays are the exception: it measures settling). Hermetic and order-independent (`-shuffle=on`). Never read the developer's real per-user config: pass an explicit temp config file (see `cmd/lumen/main_test.go`).

## Handler tests
Build a real `api.Server` with a real `state.State` and a `hosttest.Fake`, and call `Handler()` with `httptest`. Always cover: no token, wrong token, header token, bearer token; a query token on a non-event route (must fail); the host-not-ready 503; bounds (oversized body, too many slides); unknown JSON fields; SSE delivery with a real `httptest.Server`.

## What cannot be unit-tested
`internal/host/wailshost`, `internal/app` and `internal/win32` need real windows. The self-test gates placement only where `host.Capabilities` says the platform can place windows (Windows today); everywhere it requires that the output page loaded. They are covered by `make vet`/`lint` (compile) and by the self-test on a real machine. Windows-only code is **compiled** by `make ci` everywhere and **run** only on a Windows host: state which in reports. PowerPoint/COM, NDI and DeckLink cannot run in CI at all: put the logic behind an interface with a fake, and keep a manual QA checklist in the relevant skill.

## Quality bar
New code: tests in the same change. Bug fix: failing test first. No flaky tests: fix or delete, never retry. Coverage is a signal, not a gate.
