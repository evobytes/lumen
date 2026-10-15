# Deployment

Lumen is a desktop application, not a service: each operator runs it in their own session. There is no installer-created
service, no data directory with a special ACL and no listening port beyond loopback unless you ask for one.

## Windows (primary)
- Install with the Inno Setup installer (`make installer`, built in CI) or unzip the portable zip. Requires the **WebView2 runtime**
  (preinstalled on Windows 11; the installer warns if it is missing).
- Per-user configuration: `%APPDATA%\Lumen\config.toml` (copy `config.example.toml`, shipped beside the exe). `-config` overrides.
- The window placement setting `output.placement = win32` is the default and the most reliable.
- Unsigned: SmartScreen will warn until Authenticode signing exists (decision 026).

## Linux (secondary)
- `.deb` on Ubuntu 24.04+ (amd64, arm64). Depends on GTK4, WebKitGTK 6, `gstreamer1.0-plugins-good` and `gstreamer1.0-libav`
  (without the last, H.264/AAC video does not play). Wayland cannot position windows (a protocol rule); use X11 for exact
  multi-monitor placement.

## macOS (secondary)
- An unsigned `Lumen.app` zip for Apple Silicon, built by CI; Gatekeeper will warn. Not exercised on real hardware yet.

## Remote control
By default the API and pages listen on `127.0.0.1:8765`. To control Lumen from a phone, set `server.listen` to a LAN address **and**
`auth.token` (a non-loopback address without a token is refused at start-up). The API is plain HTTP: keep it on a trusted network.
Open `http://<host>:8765/?token=<token>` on the phone.

## Broadcast outputs
Not built in. Point OBS (browser source) or CasparCG (HTML producer) at `http://127.0.0.1:8765/output?token=...` for NDI or SDI.
See `docs/research/integrations-assessment.md` and decision 010.
