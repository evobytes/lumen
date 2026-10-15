---
name: config-logging
description: Configuration keys and precedence, environment variables, slog logging and the generated example config files for lumen. Use when adding config or logging.
---

# Configuration and logging

## Precedence
Defaults < per-user `config.toml` < `LUMEN_*` environment < flags. The file is `%APPDATA%\Lumen\config.toml` (Windows) or `~/.config/Lumen/config.toml`; `-config <path>` overrides and then a missing file is an error. Unknown keys are rejected. `lumen -check-config` prints the effective configuration (token redacted).

## Adding a key
1. Add the field to the right section struct in `internal/config/config.go` with `toml` and `doc` tags (the doc becomes the reference and the example files).
2. Add its `envKeys` entry (`LUMEN_<SECTION>_<KEY>`), its default in `DefaultFor`, and validation in `Validate` (report every problem; `errors.Join`).
3. `make docs`. The test in `internal/config/docs_test.go` fails if a key lacks a doc tag or an environment mapping.

## Defaults per platform
`DefaultFor(goos)` must not read the environment (docs generation uses it). Today only `output.placement` differs: `win32` on Windows, `toolkit` elsewhere.

## Secrets
`auth.token` is a shared secret: prefer the environment variable to the file; `-check-config` redacts it; the request log never records query strings (the event stream accepts `?token=`).

## Keys
<!-- BEGIN GENERATED: config-table (scripts/docgen; edit the source, run `make docs`) -->
| Key | Environment | Default (Windows) | Description |
|---|---|---|---|
| `server.listen` | `LUMEN_SERVER_LISTEN` | `"127.0.0.1:8765"` | Listen address as host:port for the web API, the control page and the output page. Keep it on loopback unless auth.token is set; use port 0 for any free port. |
| `server.read_header_timeout` | `LUMEN_SERVER_READ_HEADER_TIMEOUT` | `"5s"` | Maximum time to read the request headers. |
| `server.shutdown_timeout` | `LUMEN_SERVER_SHUTDOWN_TIMEOUT` | `"5s"` | Grace period to drain in-flight requests when quitting. |
| `auth.token` | `LUMEN_AUTH_TOKEN` | _(empty)_ | Shared secret required on /api/v1 as the X-API-Token header or an Authorization: Bearer header. Empty disables the check, which is allowed only while server.listen is on loopback. Prefer the LUMEN_AUTH_TOKEN environment variable to writing it in this file. |
| `output.screen` | `LUMEN_OUTPUT_SCREEN` | `"auto"` | Where the output window starts: auto (the first non-primary monitor, else a floating window), floating, or a monitor id such as \\.\DISPLAY2 as GET /api/v1/screens shows it. |
| `output.placement` | `LUMEN_OUTPUT_PLACEMENT` | `"win32"` | How the output window is moved onto a monitor: win32 (set the native window rectangle directly; Windows only; the default there) or toolkit (the windowing toolkit's own API; the default elsewhere). |
| `log.level` | `LUMEN_LOG_LEVEL` | `"info"` | Minimum level: debug, info, warn or error. |
| `log.format` | `LUMEN_LOG_FORMAT` | `"text"` | Output format on stderr: json or text. |
<!-- END GENERATED: config-table -->

## Logging
`log/slog` only, built by `internal/logging` (`json` or `text` to stderr). Under the Windows GUI-subsystem build stderr goes nowhere: use the console build (`scripts/build-binary windows amd64 console`, or `make selftest`) when you need logs. A rotating log file is not built yet (milestone M6).
