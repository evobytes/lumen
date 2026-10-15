# Intent

What Lumen is for: the product brief (the "what"). `docs/architecture.md` and `docs/decisions.md` hold the
"how" and "why". Features trace back to this file (see the `add-feature` skill). When a feature changes the
intent, change this file in the same commit.

Status: **early.** Section 1 is what the owner has asked for. Everything marked *proposed* (and all of section 5)
is my reading of what an OpenLP replacement needs and is **not agreed**.

## 1. What the owner asked for (verbatim intent)

- An **OpenLP replacement**: software that projects lyrics, slides and media for a live service.
- Cross-platform, **Windows first** (operators use Windows): Windows 11, Ubuntu 24.04+, macOS latest on Apple Silicon.
- **Two windows**: a parent *control* window and a floating child *output* window.
- **At least two monitors**: the output goes on a chosen monitor.
- An **integrated web API** (remote control; also served to a phone).
- GitHub build/CI; reproducible local builds (`scripts/build-deb` and friends); generated documentation.
- Compiled, easy to ship: not a Python application (decision 001).
- **Presentation files**: "play" PowerPoint (remoting PowerPoint, at least on Windows). LibreOffice support is not required.
- **Media playback**, traditionally via VLC in OpenLP: VLC is not mandated; GStreamer or other choices are open (decision 008).
- Possible later: **NDI** playout and **Blackmagic I/O** (fill and key), "in extremis", perhaps outsourced to
  GStreamer, perhaps streaming a desktop to HD-SDI (decision 010).

## 2. Users

- **Operator**: runs the projection during a service, usually on Windows, one or two monitors plus a projector
  or screen. Needs speed and no surprises: next/previous, blank, jump to an item, nothing stealing focus.
- **Remote operator** (*proposed*): drives the same session from a phone or tablet through the web UI.
- **Preparer** (*proposed*): builds the order of service ahead of time.
- **Integrator** (*proposed*): drives Lumen from scripts, a stream deck or a vision mixer through the API.

## 3. Scenarios

1. The operator starts Lumen with a projector connected: the output window covers the projector, the control window
   stays on the operator's screen, and the audience sees the first slide.
2. The projector is unplugged or a monitor is added mid-service: the output falls back to a floating window and can
   be re-placed without restarting (hot-plug handling is **not yet built**).
3. A phone on the LAN opens the control page, with a token, and moves the slides.
4. The operator runs a PowerPoint deck: PowerPoint owns the output monitor, Lumen's output yields, and Next/Previous/Blank
   reach PowerPoint (**not yet built**).
5. A song plays over a video background with the words laid on top (**not yet built**).

## 4. First slice (built)

A control window and an output window that follow one shared state; slides, next/previous/show/blank; placing the
output on a chosen monitor or leaving it floating; the web API and pages; the self-test; Windows, Linux and macOS
builds and packages. Slides are plain text; the sample deck is built in. See `docs/roadmap.md` for what is proven.

## 5. Proposed scope (not agreed): parity with OpenLP

Listed so the owner can accept, cut or reorder. None of it is committed.
- Songs: a database, search, OpenLyrics import, verse order, themes, CCLI-style reporting.
- Scripture (Bible) lookup and display.
- Service plans: ordered items, saved and reloaded.
- Presentations (PowerPoint COM / AppleScript; a PDF and image fallback).
- Media: video, audio and images with transcode-on-import, video backgrounds, alpha clips.
- Themes: fonts, outline, shadow, backgrounds, auto-fit text.
- Alerts and a stage (confidence) display.
- Broadcast outputs through OBS/CasparCG first, native NDI and SDI later.

## 6. Non-goals (so far)

- Not a service: Lumen is a desktop app configured per user (unlike the `hive-*` services).
- No in-app TLS or user accounts: the API token is a shared secret for a trusted LAN, not an identity system.
- No LibreOffice integration.
- No code signing or auto-update yet.

## 7. Open questions

See decision 026. The ones that most change the design: which of section 5 comes first; whether the control UI stays
a web page in a webview or needs native widgets; how slides are authored (text, files, a database); whether the
operator needs a "preview" of the output on the control window (likely yes).
