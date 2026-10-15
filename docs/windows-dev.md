# Developing on Windows (and from WSL2)

Windows is the primary platform (decision 003). This is how the first dev host was set up and how to rebuild the setup
on another host. The first host is expected to be replaced: nothing here is specific to it except the "Host record".

## Option A (used so far): Claude Code and the repo in WSL2, Windows binaries run through interop

- Ubuntu 24.04 in WSL2, repo on the Linux filesystem (fast builds, git and `gh` over SSH).
- Go from the version in `go.mod` (`/usr/local/go`); `make bootstrap` installs the pinned tools into `.tools/bin`.
- The Windows `.exe` is a pure-Go cross-compile (`scripts/build-binary windows amd64 [console]`); WSL interop runs it on the Windows host.
- For the Linux build: `libgtk-4-dev libwebkitgtk-6.0-dev pkg-config build-essential` (apt). WSLg shows its windows.

### Check interop
```
cmd.exe /c ver            # must print a Windows version, not "Exec format error"
```
If it fails with `Exec format error`, the `WSLInterop` entry is missing from `binfmt_misc` (a systemd race on the first host:
systemd mounted a fresh `binfmt_misc` after WSL registered it). One-off fix:
`sudo sh -c 'echo ":WSLInterop:M::MZ::/init:PF" > /proc/sys/fs/binfmt_misc/register'`. Persistent fix (reboot persistence
**untested**): a systemd unit that registers it when missing.

```ini
# /etc/systemd/system/wsl-interop-fix.service   (then: systemctl daemon-reload && systemctl enable wsl-interop-fix)
[Unit]
Description=Re-register the WSL interop binfmt handler if systemd's binfmt_misc mount dropped it
After=proc-sys-fs-binfmt_misc.mount
Requires=proc-sys-fs-binfmt_misc.mount
ConditionPathExists=!/proc/sys/fs/binfmt_misc/WSLInterop

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'echo ":WSLInterop:M::MZ::/init:PF" > /proc/sys/fs/binfmt_misc/register'

[Install]
WantedBy=multi-user.target
```

### Working with the Windows host from WSL
- Run Windows programs from a `/mnt/c/...` working directory (a UNC cwd makes `cmd.exe` warn).
- PowerShell scripts: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w script.ps1)"`.
- `curl.exe` reaches the Windows loopback; WSL's own `127.0.0.1` does not.
- `scripts/windows-display-info.ps1` records the host (read-only); `scripts/windows-screenshot` captures all monitors into a PNG
  (inside the repo); `scripts/windows-host-selftest` runs the self-test on the host but **writes outside the repo**, so it needs
  `LUMEN_ALLOW_HOST_WRITE=1` (decision 012). It also puts real windows on the desktop for about half a minute.
- Never kill processes by broad pattern; stop only the ones you started.

### When to move to Option B
Interop proves unreliable; you want routine Delve debugging or hot reload on the Windows build; 9p path slowness hurts.

## Option B: native Windows (as `hive-nmos-capture` does)

Go (version from `go.mod`), Git for Windows (Git Bash is the build shell), GNU make (MSYS2 `pacman -S make` or
`winget install ezwinports.make`; GnuWin32 make 3.81 is too old), then `make bootstrap`, `make ci`. The `.vscode/settings.json` makes
Git Bash the default terminal. `make ci` runs the tests without `-race` (no cgo). Inno Setup 6.3+ is needed for `make installer`.
Consider a Defender exclusion for the Go cache and a Win11 Dev Drive for build speed. Never commit a script without the
executable bit: `make exec-check` reads the git index (run `git update-index --chmod=+x`).

## Option C: Linux only (CI parity)

The Linux CI job runs `make ci CHECK_OS=host` with the race detector; reproduce it on any Ubuntu 24.04 with the apt packages above.

## Verifying a change on Windows
1. `make ci` (compiles for Windows from any host; on Windows it also runs the tests).
2. `make selftest` on the Windows host (or `scripts/windows-host-selftest` from WSL): placement on every monitor, owner/topmost,
   media probe. Read `dist/selftest/report.json`.
3. `scripts/windows-screenshot` and look at it: a black or grey monitor can simply mean the page has not loaded yet
   (the spike's cold start took 5-9 s before the first request).

## Host record (the first dev host, 2026-10-13; will change)
Windows 11 Pro build 26200; AMD Radeon 860M; two monitors: 2256x1504 primary at (0,0) and a 2560x1600 external at
(-2560,-90) (a negative origin); both read 100% scaling from every Win32 API while WebView2 reported `devicePixelRatio` 1.25
(unexplained: `docs/roadmap.md`); WebView2 154; Microsoft 365 PowerPoint 16; NDI 6 Tools and runtime; VLC; OpenLP 3.1.5 (to
compare behavior against); ffmpeg via winget; Go 1.23 on Windows (too old for Wails v3, so build in WSL); no GStreamer, no
Blackmagic Desktop Video drivers (only DaVinci Resolve), Defender real-time scanning on.
