---
name: security
description: Security checklist for lumen: API token, loopback binding, CSP, request bounds, what no request may name, window and native-code safety, dependencies and packaging. Use when reviewing changes or touching the API, pages, config, native code or packaging.
---

# Security

Lumen controls what an audience sees, on a machine in a room. The threats are a hostile device on the LAN, a malicious page or file, and the operator's own mistakes.

## API and pages
- Loopback by default. A non-loopback `server.listen` is refused unless `auth.token` is set (`Config.Validate`). The API is plain HTTP: trusted networks only.
- Token: `X-API-Token` or `Authorization: Bearer`, compared in constant time. The event stream alone also accepts `?token=` (browsers cannot set headers on `EventSource`); never accept a query token elsewhere, and never log query strings.
- **No request names an executable, path, URL, or device by anything but an OS-reported ID.** Placement takes a monitor ID that must match one the OS listed.
- Bodies: `http.MaxBytesReader`, `DisallowUnknownFields`, counts and sizes bounded (slides, report names). Add bounds with every new input.
- CSP `default-src 'self'` (`media-src 'self' blob:`, `img-src 'self' data:`), `nosniff`, no referrer, no framing. No inline script or style, no `eval`. Treat slide text as text (`textContent`), never HTML.
- The diagnostics store (`/api/v1/report`) accepts bounded JSON from any token holder: do not put secrets in it or trust its contents.

## Native code
- Win32 calls take handles from the toolkit and rectangles from the OS; keep `gosec` `//nolint` reasons on the unsafe pointer lines. A bad handle must return an error, not crash.
- Never launch a program named by a caller. Future presentation/ffmpeg/sidecar launches take executables from configuration, arguments as slices, never a shell (`go-style`).
- Keep `CGO_ENABLED=0` on Windows: a smaller native surface.

## Dependencies and packaging
Permissive licences only (`make docs-check`); `make vuln` (govulncheck) is in `make ci`. Pinned Wails beta: read its release notes before bumping. Do not bundle NDI, ffmpeg, GStreamer or Blackmagic software (licences and attack surface). No secrets in `os/`, `packaging/` or example configs. Unsigned binaries: say so; do not suggest users disable SmartScreen or Gatekeeper beyond what the docs state.

## Review prompts
Can a request reach exec, a path, a fetch or an unbounded allocation? Does a new page need a CSP change (it almost never should)? Does a log line carry a token or a query string?
