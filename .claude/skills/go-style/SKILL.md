---
name: go-style
description: Idiomatic Go conventions for lumen: layout, errors, interfaces, concurrency, platform code, lint and format. Use when writing or reviewing any Go code.
---

# Go style

## Layout
- Module root is the repo root. `cmd/lumen` is thin (flags, config, logging, hand-off); everything else is `internal/`. No `pkg/`.
- Package names: short, lowercase, singular, no underscores, no `util`/`common`/`helpers`. One concept per package; dependencies point inward. Every package has a doc comment starting `Package x ...` (it feeds the generated package table).
- **Portable core, thin Windows edge.** Logic compiles and tests on every host with no C toolchain. Only code that must call Windows APIs sits behind `//go:build windows` in `internal/win32`, with an inert `_other.go` twin and a shared `errors.go`. Toolkit code lives only in `internal/host/wailshost` and `internal/app`.
- `make vet` and `make lint` check Windows (CGO off) and the host platform, so Windows-only files are always compile-checked.

## Formatting and lint
`gofumpt` + `goimports` (local prefix is the module path). `make fmt` fixes, `make ci` checks. `golangci-lint` v2 per `.golangci.yml`. `//nolint:<linter> // reason`: the reason is mandatory (the Win32 calls carry `gosec` reasons for the unsafe pointers).

## Errors
Return errors wrapped with context: `fmt.Errorf("place output: %w", err)`; lowercase, no trailing punctuation. Sentinels (`state.ErrOutOfRange`, `win32.ErrUnsupported`) for conditions callers branch on, checked with `errors.Is`. `errors.Join` to report every problem at once (config validation). Handle an error once: log or return, not both. `panic` only for programmer errors at start-up (an embedded directory missing).

## Interfaces and types
Accept interfaces, return concrete types; define them at the consumer and keep them small (`host.Host`). Coordinates are physical pixels in virtual-desktop space; the origin can be negative. No package-level mutable state, no `init()` side effects.

## Context and concurrency
`ctx context.Context` first for anything doing I/O; never store it in structs. Every goroutine has an owner and a shutdown path. State uses a mutex; subscribers get coalescing channels. Wails window calls go through `application.InvokeSync`. Run tests with `-race` where cgo exists.

## Misc
`log/slog` only. Doc comments on every exported identifier, starting with its name. Inject time where it matters. `defer` cleanup immediately after acquisition. Windows builds never use CGO (decision 017).
