---
name: broadcast-outputs
description: How lumen reaches NDI and SDI (Blackmagic DeckLink fill and key): the outsourcing order, NDI runtime loading, GStreamer DeckLink, alpha-frame sources and licensing limits. Not built yet; read before any broadcast work. Use when adding NDI, SDI or desktop-capture output.
---

# Broadcast outputs (deferred; decision 010)

The owner wants NDI and possibly Blackmagic fill-and-key "in extremis". Build nothing in the core first.

## Order of preference
1. **Outsource to OBS**: add a browser source pointed at `http://127.0.0.1:8765/output?token=...`; OBS provides NDI (DistroAV, GPL-2.0, user installs the NDI Runtime) and DeckLink output. Zero Lumen code. OBS DeckLink keyer capability: unverified.
2. **Outsource to CasparCG** (2.5.1, GPL-3.0; HTML producer on CEF 142, DeckLink, NDI; AMCP control). **Windows and Ubuntu only: no macOS build** (verified from release assets). Heavier operations.
3. **Optional sidecars, later**: an NDI sender that loads the NDI runtime at run time with `ebitengine/purego` (no CGO), and a GStreamer process for DeckLink and desktop capture. Separate executables keep licences and size out of the core.

## Facts and limits
- **Alpha frames** are the crux of fill-and-key. System webviews give none. A Chromium off-screen renderer does: CEF (bindings: `tauri-apps/cef-rs`, `energye/energy`, `CefSharp`), OBS's browser source, CasparCG's HTML producer. GStreamer's `webview2src` (gst-plugins-bad) renders a WebView2 page to BGRA frames on Windows (D3D11/D3D12/system memory) but whether per-pixel alpha survives is **unverified**; WebView2 only supports alpha 0 or 255 for its default background colour.
- **GStreamer `decklinkvideosink`**: `keyer-mode` `off|internal|external`, `duplex-mode` (full needed for keying), `video-format=8bit-bgra`, accepts BGRA/ARGB. External mode puts the alpha as luma key on the primary port and the fill on the secondary (broadcast fill and key). Needs a duplex-capable card (the docs name Duo 2 and Quad 2).
- **DeckLink SDK**: per-OS (COM IDL on Windows; `DeckLinkAPI.h` plus a dispatch file on macOS/Linux); Desktop Video drivers required; no maintained Go binding (searched); its Windows COM interfaces are not `IDispatch`, so `go-ole` cannot call them (assumed): use GStreamer, OBS or CasparCG.
- **NDI**: do not bundle the runtime. Its distribution terms allow redistribution only if the product's licence covers the NDI EULA and Windows installers are kept current; dynamic loading is "the preferred mechanism"; the Advanced SDK page requires attribution; commercial hardware/software distribution goes through NDI sales. NDI carries alpha (BGRA/UYVA) in one stream (from SDK knowledge, not re-verified). No mature Go binding exists.
- **PowerPoint output** has no alpha: only a plain fill via desktop capture (GStreamer `d3d11screencapturesrc` / NDI Tools Screen Capture / OBS display capture).

## If you build a sidecar
Own executable, started from a configured path with an argument slice, supervised, never launched by a request, state read from the same `/api/v1` events, a fake for tests, and DeckLink/NDI behavior verified on real hardware (not in CI). Update `docs/research/integrations-assessment.md` and `docs/decisions.md`.
