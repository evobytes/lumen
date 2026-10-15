---
name: media
description: Media playback rules for lumen: HTML5 video in the webview, codec findings per OS, no VLC, ffmpeg transcode-on-import, audio routing. Use when touching video, audio, backgrounds or media import.
---

# Media (design; partly verified; implementation is milestone M4)

## Decision (008)
Play media with HTML5 `<video>`/`<audio>` in the output page. Not VLC (OpenLP master itself moved to Qt Multimedia; libVLC 4 unreleased as of May 2026; heavy to bundle). Not a native video widget either: lyrics over a video background needs browser compositing, and a native child window over a webview has the "airspace" problem.

## What each webview plays (evidence)
| Webview | Result | Status |
|---|---|---|
| WebView2 154 (Windows 11) | H.264+AAC, HEVC, AV1, VP9+Opus all play; VP9-with-alpha decodes (alpha sample 127 for a 50% clip); `setSinkId` is a function; 1 audio output enumerated | **verified** |
| WebKitGTK 6.x (bare Ubuntu 24.04) | H.264, AAC, HEVC, AV1 all **no**; VP8, VP9, Vorbis, Opus, MP3 yes; no `setSinkId` | **verified** |
| WKWebView (macOS) | not tested | **unknown** (VP9 WebM expected on macOS 11.3+, H.264 expected) |
Cause on Linux: `libwebkitgtk-6.0-4` Depends on `gstreamer1.0-plugins-base` and `-good` but only *Suggests* `gstreamer1.0-libav`. The deb therefore depends on `gstreamer1.0-libav`. Re-run `make selftest` to see the probe on any machine.

## Rules
- Import pipeline: probe with ffprobe; play as-is when the webview supports the codec; otherwise offer **transcode to WebM (VP9 + Opus)** with ffmpeg (libvpx/libopus are royalty-free and play everywhere). ffmpeg is an optional, user-installed or optionally downloaded tool, never bundled (decision 010). Run it with an argument slice from a configured path, never from a request.
- Videos play **muted for autoplay** unless a user gesture exists; the output window has none, so unmuted playback needs `--autoplay-policy` handling in WebView2 (decide when building M4).
- Audio output routing (send sound to the projector HDMI while the house PA stays separate): `setSinkId` exists on WebView2 only. Where it is missing, route audio in the Go process with a native audio library instead of the page.
- Alpha backgrounds: VP9 with alpha works in WebView2; verify per OS before relying on it. HEVC alpha is Safari-only.
- Never put a media path in a request: files are addressed by library ID; `internal/api` serves media only from a configured library.
