#!/usr/bin/env bash
# Cross-compile the spike for Windows from Linux/WSL. Pure Go: no CGO, no Windows toolchain.
#   ./build-windows.sh            -> bin/lumen-spike-console.exe   (shows logs; used for self-test)
#   ./build-windows.sh gui        -> bin/lumen-spike.exe           (-H windowsgui, no console)
set -euo pipefail
cd "$(dirname "$0")"
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
variant="${1:-console}"
mkdir -p bin

# Embeds the icon, version info and the per-monitor-v2 DPI manifest.
( cd build && wails3 generate syso -arch amd64 -icon windows/icon.ico \
    -manifest windows/wails.exe.manifest -info windows/info.json -out ../wails_windows_amd64.syso ) >/dev/null
trap 'rm -f wails_windows_amd64.syso' EXIT

if [ "$variant" = gui ]; then
  ldflags="-w -s -H windowsgui"; out=bin/lumen-spike.exe
else
  ldflags="-w -s"; out=bin/lumen-spike-console.exe
fi
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags production -trimpath -buildvcs=false \
  -ldflags="$ldflags" -o "$out" .
ls -l "$out" | awk '{printf "built %s (%.1f MB)\n", $9, $5/1e6}'
