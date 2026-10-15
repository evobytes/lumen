# Configuration reference

Precedence (low to high): built-in defaults, the config file, `LUMEN_*` environment variables, command-line flags. The file is
per-user (`%APPDATA%\Lumen\config.toml` on Windows, `~/.config/Lumen/config.toml` on Linux); `-config <path>` overrides it and
then a missing file is an error. Unknown keys are rejected. `lumen -check-config` prints the effective configuration (token
redacted) after validation.

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

Flags: `-config`, `-listen`, `-log-level`, `-version`, `-check-config`, and the developer and CI modes `-selftest`, `-report`,
`-demo`, `-placement`, `-hold`. Example files with every key at its default: `os/usr/share/lumen/config.example.toml` (Linux) and
`packaging/windows/config.example.toml` (Windows); both are generated.
