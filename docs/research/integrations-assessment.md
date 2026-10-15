# Integrations assessment: presentations, media, NDI, SDI

Date: 2026-10-13. Companion to [language-assessment.md](language-assessment.md).
Covers four things OpenLP does or could do: PowerPoint control, media playback, NDI output, and
SDI (Blackmagic) fill-and-key.

## 1. Summary

1. **PowerPoint remoting is very doable and mostly toolkit-neutral.** OpenLP does it with
   COM automation on Windows and AppleScript on macOS. Both are reproducible from Go. Real
   PowerPoint cannot be tested in GitHub CI, so it needs a driver interface plus manual QA.
2. **VLC is the wrong dependency for 2026.** OpenLP's own master branch no longer uses it
   (verified: its media code is Qt Multimedia only). libVLC 4 still had no official release as of
   May 2026 (secondary source), and libVLC 3 is heavy to bundle.
3. **Playing video inside a webview has one verified problem.** On this Ubuntu 24.04 machine,
   WebKitGTK reported **H.264, AAC, HEVC and AV1 as unplayable**; VP8/VP9/Vorbis/Opus/MP3 are fine.
   It is fixable (dependency or transcode), but it is a real cost of the Wails/Tauri route.
4. **GStreamer is the right tool for NDI/SDI output, not for in-window playback.** It has a
   DeckLink sink with hardware keyer support (verified). Run it as an optional **sidecar
   process**, so the main app stays lean and language-neutral.
5. **Alpha frames are the crux of fill-and-key.** To key lyrics you need rendered frames *with
   alpha*. System webviews (Wails/Tauri) don't give you that; a Chromium-embedded renderer
   (CEF), OBS or CasparCG does. This is the one place the expanded scope pressures the Wails
   choice.
6. **Recommendation: keep Go + Wails v3 for the operator app, but make the output a URL-driven
   web page with pluggable renderers** (section 7). NDI/SDI become optional add-ons you can
   defer, outsource to OBS/CasparCG, or build later, without re-architecting.

Confidence in the Wails choice drops a notch (section 8). The spike gets four new tests.

Targets assumed: Ubuntu 24.04+, Windows 11, latest macOS on Apple Silicon.

## 2. Presentation files (PowerPoint and friends)

### 2.1 How OpenLP does it (read from OpenLP master on GitLab)

| Platform | Mechanism | Notable details and quirks |
| --- | --- | --- |
| Windows | Late-bound COM (`PowerPoint.Application`) | `SlideShowSettings.Run()`, `View.GotoSlide/Next/Previous`, blank via `View.State = 3`, stop via `View.Exit()`. Picks the monitor by writing the **`DisplayMonitor` registry value** before `Run()`. Slide thumbnails via `slide.Export`. |
| Windows quirks | Documented in code comments | "Ghost windows" after windowed shows (closed via `WM_CLOSE`); unblank broken in PowerPoint 2010; `Slide` unavailable on the end slide and `CurrentShowPosition` returns 0 during transitions in 2013; PowerPoint steals focus and the show pauses (`State = 2`) when it loses focus. |
| macOS | AppleScript (needs PowerPoint 15+) | Slide thumbnails via PDF export. **AppleScript can't jump to a slide**, so OpenLP types the number as keystrokes plus Return. Fullscreen uses an accessibility attribute. Keynote has its own controller. |
| Linux / other | LibreOffice Impress via UNO (`libreofficeserver.py`) | You said you may not need this. |
| Fallback | PDF controller (Ghostscript / MuPDF) | Static pages only. |

### 2.2 What it means for us

- **Go can do all of it.** `go-ole` (1,320 stars, pushed 2026-10-12) provides COM `IDispatch`
  from Go. On macOS, shelling out to `osascript` is enough; no library is needed. C# would be
  smoother for COM (first-class interop), but this is not a deal-breaker.
- **Treat it as a controller interface**, not as part of the UI:
  `Load / Start(screen) / Next / Prev / Goto / Blank / Stop / SlideInfo`.
  Implement it for PowerPoint-COM, PowerPoint-AppleScript, Keynote, and a static PDF/image
  fallback, plus a **fake for tests**.
- **The output window must yield.** PowerPoint opens its own native fullscreen window on the
  output monitor, and our always-on-top output window would sit on top of it. Model it as
  "output owner = Lumen | external app": hide or lower our window while a slideshow runs, and
  route *blank* through COM (`View.State`). OpenLP has the same dance.
- **macOS needs user permissions.** Automating PowerPoint triggers the Automation prompt; the
  keystroke workaround and fullscreen handling very likely need Accessibility permission (my
  inference from OpenLP's code; not tested).
- **Not testable in CI.** GitHub runners don't have Office installed (my understanding; not
  checked this session), so PowerPoint code needs manual QA on a real machine and good unit tests
  against the fake.
- **Fidelity vs independence trade-off:** driving real PowerPoint gives full animations, video and
  transitions but requires PowerPoint. Converting to images (LibreOffice headless to PDF, then
  render) works anywhere but drops animations and embedded media. Offer both.

## 3. Media playback

### 3.1 Is VLC still appropriate?

**No, for a new project.**

- **Verified:** OpenLP master's media package contains only `mediaplayer.py` and `audioplayer.py`;
  the player uses `QMediaPlayer` / `QVideoWidget` (Qt Multimedia). No VLC.
- libVLC 4.0: a May 2026 porting diary says there is "still no official 4.x release in sight";
  4.x changes the API (async stop/pause, new track list). libVLC 3.x is what you can embed today.
- `libvlc-go` (510 stars, pushed 2026-07-16) uses CGO. libVLC must be bundled (large) or
  installed by the user, and its plugin tree is awkward to package on Windows and macOS.
- VLC's strength is playing almost anything. A transcode-on-import step (below) gives you most
  of that benefit without shipping it.

### 3.2 Options compared

| Option | Plays what | Composites lyrics over video | Packaging | Fits Go + Wails? | Notes |
| --- | --- | --- | --- | --- | --- |
| **HTML5 `<video>` in the webview** | Whatever the OS webview supports | **Yes, trivially** (video is a layer under text) | Nothing to bundle | Native fit | See 3.3: Linux codec gap; no audio-device routing on WebKitGTK |
| libmpv | Nearly everything, great quality | Hard: native child window vs webview ("airspace") | Bundle libmpv per OS | Poor | `gen2brain/go-mpv` 61 stars |
| libVLC 3 | Nearly everything | Same airspace problem | Large | Poor | Section 3.1 |
| GStreamer | Everything via plugins | Same airspace problem in-window | Official installers exist but are big (not verified here) | Poor in-window; excellent as sidecar | `go-gst` 270 stars, active |
| Qt Multimedia (OpenLP's choice) | FFmpeg backend | Yes (QVideoSink / QML) | Part of Qt | Only if we stay on Qt | That's the PySide6 prototype |
| Native video via Avalonia | LibVLCSharp / FFmpeg | Needs frame callbacks or an overlay window | ~100 MB base | n/a | Airspace issues similar to above |

The deciding factor for a lyrics app is **lyrics over video backgrounds** (themes with video
backgrounds are common). Browser compositing makes that trivial; native video widgets make it
painful. That favours the web renderer, if its codec limits can be handled.

### 3.3 Verified: webview video capabilities on this machine

I wrote a minimal GTK4 + WebKitGTK 6 program (Ubuntu 24.04, WebKitGTK 2.52.6) and asked the
engine what it can play:

| Format (`canPlayType`) | Result |
| --- | --- |
| H.264 (`avc1`) in MP4, with and without AAC | **NO** |
| HEVC (`hvc1`) | **NO** |
| AV1 | **NO** |
| AAC audio in MP4 | **NO** |
| VP8, VP9 in WebM | probably (yes) |
| Vorbis (Ogg), MP3, FLAC | yes / maybe |
| `HTMLMediaElement.setSinkId` | **undefined** (no per-element audio output selection) |

Cause (verified from apt): `libwebkitgtk-6.0-4` **Depends** on `gstreamer1.0-plugins-base` and
`-good`, but only **Suggests** `gstreamer1.0-libav` and `-plugins-bad`. Only `plugins-good` is
installed here. A normal Ubuntu desktop may have more installed, so this is a worst case, not a
guarantee.

Options for Linux, none of them hard:

1. Declare `gstreamer1.0-libav` as a package dependency (works for `.deb`; not for tarballs).
2. **Transcode on import** to **WebM (VP9 + Opus)** with ffmpeg. libvpx and libopus are
   royalty-free, WebM plays in all three webviews (Linux verified above; WebView2 and WKWebView
   expected, not tested), and the app never depends on proprietary codec support. The cost is
   import time and a bundled or required `ffmpeg`.
3. Both: play what the system can, offer one-click conversion for the rest.

Audio output device selection (for example, send audio to the projector's HDMI while the house
PA stays separate) is not available through `setSinkId` on WebKitGTK. It may work on WebView2
(Chromium) and Safari 18+, but I did not test either. If per-device audio routing matters,
route audio in the Go process (native audio library) instead of the page.

**Update, Windows verified (2026-10-13, spike in `spikes/windows-wails`):** WebView2 154 on Windows 11
played **H.264+AAC, HEVC, AV1 and VP9+Opus**, decoded VP9-with-alpha correctly, and exposes
`setSinkId`. So the Linux codec gap does not exist on Windows, and Windows needs no transcoding.
Not tested: WKWebView (macOS).

## 4. NDI

- **What it needs:** the NDI runtime library (`libndi`) plus frames in a supported pixel format.
  NDI supports alpha, so fill and key can travel in a single stream (from my knowledge of the SDK;
  not re-verified).
- **Licensing (verified from NDI's docs):** you may bundle the NDI binaries if your EULA
  covers the NDI SDK EULA terms; Windows redistributable installers may be shipped but you must
  keep them reasonably current; headers are MIT; **dynamic loading is "the preferred mechanism"**.
  A separate NDI page says commercial hardware/software distribution should go through NDI sales
  (the Advanced SDK page; check which SDK applies). Platform-specific notes for macOS, Linux and
  ARM were not found on that page.
- **The precedent to copy:** DistroAV (the OBS NDI plugin; GPL-2.0, 4,571 stars, active, installable
  via winget, Homebrew and Flatpak) **requires the user to install the NDI Runtime (v6.3+)**
  rather than bundling it. That sidesteps redistribution entirely.
- **From Go without CGO:** `ebitengine/purego` (4,024 stars, Apache-2.0, pushed 2026-10-09)
  calls shared libraries via `dlopen` with no C compiler. It lists Linux amd64/arm64, and it
  supports Windows and macOS. So `libndi` can be loaded at runtime from pure Go. I did not
  prototype it.
- **No mature Go NDI binding exists** (searched GitHub; nothing relevant). You would write a thin
  `purego` wrapper (~a few hundred lines for send-only video).
- **Where do frames come from?** That is the hard part (section 6).

## 5. SDI / DeckLink fill-and-key

- **GStreamer's `decklinkvideosink` supports hardware keying (verified from its docs and source
  mirrors):** `keyer-mode` (`off`/`internal`/`external`), `duplex-mode` (`full` needed for
  keying), `video-format=8bit-bgra`, accepting BGRA and ARGB frames. In **external** mode the
  primary port carries the alpha as a luma **key** and the secondary port carries the **fill**,
  which is the broadcast fill-and-key arrangement. Documentation says this needs a card with
  duplex support (the docs name Duo 2 and Quad 2).
- **The DeckLink SDK is per-OS:** a COM `.idl` on Windows, `DeckLinkAPI.h` plus a dispatch file
  on macOS and Linux (verified from Blackmagic's docs). Desktop Video drivers are required. I
  could not confirm native Apple Silicon support in the SDK, and I found no maintained Go binding
  (only a handful of tiny repos). Calling it from Go would mean a C++ shim over CGO: avoid.
- **"Stream a desktop to SDI":** a GStreamer pipeline of screen capture into `decklinkvideosink`
  does this (capture elements per OS: `d3d11screencapturesrc`, `avfvideosrc`, `ximagesrc` or
  PipeWire; from my knowledge, not verified this session). It is also the answer for **PowerPoint
  output**, which is a native window your own renderer can't see.
- **Zero-code alternatives** (from my knowledge, not verified): NDI Tools' free Screen Capture
  for desktop to NDI, then any NDI-to-SDI converter; or OBS display capture.
- **CasparCG** (2.5.1, 2026-09-16, GPL-3.0): a broadcast playout server with an HTML producer
  (CEF 142), DeckLink and NDI outputs, controlled over its AMCP protocol. Release assets are
  **Windows and Ubuntu only: there is no macOS build** (verified from the release assets).
- **Hardware can't be tested in CI.** Only pipeline construction can be unit-tested.

## 6. The crux: where do rendered frames come from?

For NDI and fill-and-key you need **BGRA frames with alpha** of whatever the audience sees.

| Source of frames | Alpha? | Works for | Cost |
| --- | --- | --- | --- |
| System webview (WebView2 / WKWebView / WebKitGTK) | No practical route I could verify | n/a | Not designed for it |
| **CEF off-screen rendering** (what OBS browser source and CasparCG's HTML producer use) | Yes | Lyrics/themes/web output | Chromium ~150+ MB; extra process. Bindings: `tauri-apps/cef-rs` (478 stars), `energye/energy` (Go, 618 stars), `CefSharp` (C#, 10k stars) |
| **OBS browser source** pointed at our output URL | Yes | Lyrics/themes; also NDI and DeckLink out via OBS | Outsourced entirely; user installs OBS |
| **CasparCG HTML producer** | Yes | Broadcast-grade fill/key; Windows and Linux only | Outsourced; heavier ops |
| Native compositor (Avalonia/Skia `RenderTargetBitmap`) | Yes | Our own rendering | We'd render text/video ourselves |
| Desktop / window capture | No | PowerPoint and other external apps (no key, fill only) | Easy; no alpha |
| Electron OSR | Yes | Whole stack | Not compiled; ~100-250 MB (see language-assessment.md) |

Fill-and-key of **PowerPoint** is not possible in general (PowerPoint has no alpha output); only
a plain fill feed via desktop capture. Say this plainly in the product docs.

## 7. Recommended architecture

**Output as a pure function of server state, with interchangeable renderers.**

```
                   ┌──────────────── Lumen (Go process) ───────────────┐
 phone / script ──▶│ HTTP + WebSocket API: state, slides, media, themes │
                   │ presentation controllers (COM / AppleScript / PDF) │
                   │ media library + ffmpeg sidecar (probe, transcode)  │
                   └──────────┬─────────────────────────────────────────┘
                              │ /output (HTML page, subscribes to state)
        ┌─────────────────────┼──────────────────────────┬────────────────┐
        ▼                     ▼                          ▼                ▼
  Wails output window   OBS browser source        CasparCG HTML      kiosk Chromium /
  (default, on-screen)  → NDI / DeckLink          producer → SDI     second PC / Pi
                                                  (Win/Linux only)
                         optional sidecars (later): lumen-ndi (purego), lumen-gst
                         (GStreamer: desktop capture → DeckLink fill/key)
```

- **Day 1:** Wails output window renders `/output`. Lyrics, images, audio, WebM/H.264 video,
  PowerPoint control, the web API.
- **Broadcast, deferred and optional:** point OBS (NDI + DeckLink) or CasparCG at the same URL.
  No Lumen code is needed, because the page already works anywhere. This is the cheapest way to
  get fill-and-key.
- **If you later want it built in:** a small sidecar. Either a CEF off-screen renderer sending
  to NDI via `purego`, or a GStreamer process for DeckLink. Both are separate executables, so
  their licensing and size don't touch the core app.
- **Presentation sources** (PowerPoint) bypass the page: native window on the output monitor,
  plus optional desktop capture for NDI/SDI fill.
- Ship `ffmpeg` as an optional download or detect the system one, rather than bundling it.

## 8. What this does to the language/toolkit decision

| Criterion | Go + Wails v3 | C# + Avalonia 12 | Rust + Tauri 2 | Electron |
| --- | --- | --- | --- | --- |
| PowerPoint control ergonomics | OK (`go-ole`, `osascript`) | **Best** (COM interop) | OK (`windows` crate) | OK (node-ffi/edge) |
| Lyrics over video | **Excellent** (HTML) | Hard (airspace) | **Excellent** | **Excellent** |
| Video codec consistency | Weak (system webviews; Linux gap verified) | Good (we ship decoders, but extra work) | Same as Wails | **Best** (bundled Chromium) |
| Audio device routing | Weak in page; do it in Go | Native | Weak in page | Good (`setSinkId`) |
| Alpha frames for NDI/SDI | Via sidecar/OBS | Native (`RenderTargetBitmap`) | Via sidecar/OBS; `cef-rs` | **Built in** (OSR) |
| NDI/DeckLink wrappers | `purego` + sidecar | P/Invoke; DeckLink COM on Windows | FFI | npm modules, native addons |

Verdict: the expanded scope **narrows the lead** of Wails, but doesn't overturn it.

- It strengthens Electron-class options (consistent codecs, alpha frames, audio routing), but
  these remain excluded by your compiled-and-lean preference.
- It strengthens C#/Avalonia for Windows broadcast integration (COM everywhere), but it makes
  lyrics-over-video harder, which is the product's main use.
- Wails keeps the best fit for your skills and for the core product. The weak points (codecs,
  alpha frames, audio routing) all have **sidecar or outsource answers** that don't change the
  language.
- If the spike shows webview video is unreliable on Windows or macOS too, move the output
  renderer to CEF (Tauri's `cef-rs` or a sidecar) rather than abandoning Go.

## 9. Spike additions (to the 8 tests in language-assessment.md)

| # | Test | Why |
| --- | --- | --- |
| 9 | Play H.264 MP4 *and* VP9 WebM, with audio, in WebView2, WKWebView and WebKitGTK (with and without `gstreamer1.0-libav`) | Section 3.3 verified a gap on Linux only |
| 10 | Drive PowerPoint end-to-end from Go via `go-ole` on a Windows machine with Office; AppleScript on a Mac | Can't run in CI; needs a manual checklist |
| 11 | Send a BGRA frame with alpha as NDI from Go via `purego` on all three OSes | Proves the CGO-free NDI path |
| 12 | Open `/output` in OBS as a browser source and receive it as NDI with alpha | Validates the outsource route before building anything |

DeckLink and hardware tests need real cards; plan manual QA, not CI.

## 10. Sources and confidence

**Verified directly this session:**

- OpenLP master (GitLab): `powerpointcontroller.py`, `powerpointmaccontroller.py`,
  `mediaplayer.py`, `mediacontroller.py`, and the directory listings for presentations and media.
- **Experiment:** GTK4 + WebKitGTK 6 `canPlayType` and `setSinkId` probe on Ubuntu 24.04
  (results in 3.3); `apt-cache depends`/`policy` for the WebKitGTK and GStreamer packages.
- NDI software-distribution rules (docs.ndi.video).
- GStreamer `decklinkvideosink` documentation (keyer modes, BGRA/ARGB caps).
- GitHub metadata for `go-ole`, `libvlc-go`, `go-gst`, `go-mpv`, `purego`, `cef-rs`, `energy`,
  `CefSharp`, DistroAV, OBS, CasparCG (stars, licences, recent activity, release assets).

**From web search only (secondary):** libVLC 4 having no official release as of May 2026;
`keyer-level` and Duo 2 / Quad 2 duplex details; Blackmagic SDK per-OS headers; DeckLink licence
wording (old headers; check the current SDK); CasparCG feature list; Electron OSR behaviour.

**From my general knowledge, not checked this session:** NDI alpha support, GStreamer screen
capture element names, NDI Tools Screen Capture, WebView2 / WKWebView codec support, Qt 6's FFmpeg
backend, OBS DeckLink output capabilities.

**Not tested at all:** anything on Windows or macOS; PowerPoint or Keynote automation; NDI or
DeckLink with real software or hardware; DeckLink SDK on Apple Silicon; CEF bindings; the GStreamer
NDI plugin (its README is old and says it needs the NDI SDK library; I could not confirm whether
current GStreamer builds ship it).

Links:

- OpenLP: <https://gitlab.com/openlp/openlp>
- NDI software distribution: <https://docs.ndi.video/all/developing-with-ndi/sdk/software-distribution>
- GStreamer DeckLink sink: <https://gstreamer.freedesktop.org/documentation/decklink/decklinkvideosink.html>
- DeckLink SDK docs: <https://sdk-doc.blackmagicdesign.com/decklink-sdk/decklinkapi.html>
- CasparCG: <https://github.com/CasparCG/server/releases>
- DistroAV: <https://github.com/DistroAV/DistroAV>
- purego: <https://github.com/ebitengine/purego> · go-ole: <https://github.com/go-ole/go-ole>
- cef-rs: <https://github.com/tauri-apps/cef-rs> · CefSharp: <https://github.com/cefsharp/CefSharp>
