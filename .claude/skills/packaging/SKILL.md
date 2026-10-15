---
name: packaging
description: Building and verifying lumen's packages: Linux deb (nfpm), Windows zip and Inno Setup installer, macOS app zip; the os/ tree and what each package installs. Use when touching os/, packaging/, scripts/build-* or make packages.
---

# Packaging

## The `os/` tree
Mirrors the Linux target filesystem; `scripts/build-deb` appends every file under it to the nfpm spec, so adding a file under `os/` is all it takes to ship it.
```
os/usr/share/applications/lumen.desktop            # desktop entry
os/usr/share/icons/hicolor/scalable/apps/lumen.svg # icon (placeholder artwork)
os/usr/share/lumen/config.example.toml             # GENERATED from internal/config (make docs)
```
The binary goes to `/usr/bin/lumen`; `packaging/copyright` (generated Debian DEP-5), `LICENSE`, `NOTICE`, `THIRD_PARTY_NOTICES.md` go to `/usr/share/doc/lumen`. Lumen is per-user configured, so nothing under `/etc`. Never hard-code install paths in Go.

## Linux deb (`make deb`, `scripts/build-deb`)
Built natively (Wails needs CGO) on Linux of the target architecture with `libgtk-4-dev` and `libwebkitgtk-6.0-dev`; CI uses `ubuntu-24.04` and `ubuntu-24.04-arm`. Version and arch are passed to `packaging/nfpm.yaml` (`${VERSION}`, `${GOARCH}`); the file name carries the build stamp. **Depends** on `libgtk-4-1`, `libwebkitgtk-6.0-4`, `gstreamer1.0-plugins-good` and `gstreamer1.0-libav`: WebKitGTK only suggests the last, and without it H.264/AAC/HEVC/AV1 do not play (verified). Check a built package with `dpkg-deb -I` and `dpkg-deb -c`.

## Windows (`make win-zip`, `make installer`)
- `scripts/build-binary windows amd64` is pure Go and embeds the icon, version info (from `VERSION`) and the per-monitor-v2 DPI manifest via `wails3 generate syso` (the `.syso` is generated next to `cmd/lumen/main.go` and removed after). `packaging/windows/lumen.exe.manifest` **must** declare per-monitor-v2 DPI awareness or every monitor measurement is invalid.
- `scripts/build-windows-zip` works from any host (zip, or PowerShell's `Compress-Archive` under Git Bash).
- `scripts/build-windows-installer` needs Windows, Git Bash and Inno Setup 6.3+ (`ISCC.exe`, or set `ISCC`); script `packaging/windows/lumen.iss`; `COPYRIGHT` supplies the publisher. It warns when the WebView2 runtime is missing. `AppId` identifies the product across upgrades: never change it. Per-machine install; unsigned.

## macOS (`make macos-app`)
`scripts/build-macos-app` builds `Lumen.app` (Info.plist from `packaging/macos/Info.plist.in`) and zips it with `ditto`. Unsigned and not notarised; built only by the CI macOS job.

## Verification status
2026-10-14: Linux deb built and inspected (`dpkg-deb -I`/`-c`: dependencies, files and modes correct; **not installed**). Windows zip built; the exe's version info and per-monitor DPI manifest were confirmed in its resource section. Docs bundle reproducible. **Installer: never built** (no `ISCC` on the first host). macOS app: never built. Say so when you report.
