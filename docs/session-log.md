# Session log (2026-10-13 to 2026-10-14)

A chronological record of the working session that created this project, written so someone with no memory of it can pick up.
It records what the owner asked, what was found, what went wrong and what changed. Decisions are in `docs/decisions.md`;
this is the narrative and the evidence trail. Status of every claim is in `docs/roadmap.md`.

## The owner
Dave Bevan, Evobytes Consulting Ltd. Good Go experience, some Python, little PySide6, excellent web skills. Prefers compiled,
statically-built languages, dislikes Python app packaging. Works in WSL2 on a Windows 11 laptop with two monitors; wants Windows
to be the primary target ("operators favour Windows"). Likes short, decisive answers with a recommendation first, honest
verified/unverified labelling, and `make` as the single interface. Maintains `tips-go` and the `hive-*` repositories under the
`evobytes` and `dB-Broadcast` GitHub organisations; this project follows their conventions (decision 013).

## Timeline

1. **Brief.** "Build a cross-platform python + pyside6 project: Linux, Windows, Mac; two windows (parent with a floating child);
   integrated web API; two monitors at least; GitHub build/CI; construct as much as you can, use make." The name Lumen was chosen
   by the assistant ("OpenLP replacement" was the directory name). Built `spikes/python-pyside6`: a control window, a frameless
   always-on-top output window, FastAPI + uvicorn in a thread bridged to the Qt thread, pytest-qt tests (33), ruff, a Makefile,
   PyInstaller packaging with a `--self-test`, GitHub CI and release workflows. Verified on Linux only. MIT licence and a personal
   copyright were placed as a stopgap (superseded by decision 014).
2. **"Python as an app feels a nightmare."** The owner asked for a language assessment (Go, Swift, other), with research and tables.
   Research (GitHub API, Go module sources, a compiled spike, search): Go + Wails v3 recommended, gated by a spike; Avalonia and
   Tauri as fallbacks; Gio, Fyne, Swift, Flutter eliminated (decision 002). A real Avalonia 12 spike was built and cross-published
   for three OSes from Linux (97/102/109 MB, untrimmed) to test the strongest alternative. Output: `docs/research/language-assessment.md`.
   An honest correction was recorded: Electron scored highest (84) but was excluded by the owner's "compiled, lean" preference, and
   the Wails/Avalonia/Tauri scores are a tie within noise.
3. **PowerPoint, media, NDI, SDI.** The owner asked about presentation remoting, VLC vs GStreamer, NDI and Blackmagic fill-and-key.
   Findings: OpenLP's code (PowerPoint COM and AppleScript with documented quirks; media moved from VLC to Qt Multimedia); a probe
   program showed WebKitGTK cannot play H.264/AAC out of the box (only *suggests* `gstreamer1.0-libav`); NDI licensing favours runtime
   loading; GStreamer's `decklinkvideosink` supports hardware keying; CasparCG has no macOS build; fill-and-key needs a Chromium renderer
   for alpha. Output: `docs/research/integrations-assessment.md` and decisions 008-010.
4. **"Windows is the primary objective."** Re-assessed: Wails gets safer (the known bugs were macOS/Linux; Windows is CGO-free; WebView2 is
   Chromium); C# + WPF + WebView2 is the real fork. Direct inspection of the Windows drive (read-only) showed WebView2 154, PowerPoint 16,
   NDI 6, no Blackmagic drivers. `webview2src` in GStreamer was found as a Windows route to BGRA frames (alpha unverified).
5. **Where to run the agent.** Recommended staying in WSL2 and enabling interop (decision 011). The assistant first said no `sudo`
   was available: **that was wrong** (passwordless `sudo` works); the real blocker had been missing GTK libs, which the owner then
   installed. Interop was broken by a `binfmt_misc` race; the owner registered the handler by hand, then a systemd unit made it
   persistent (reboot persistence untested).
6. **The Windows spike** (`spikes/windows-wails`). Cross-compiled a pure-Go Wails v3 app and ran it on the owner's desktop through
   interop. Results in `docs/roadmap.md`: window model works, 12/12 exact placements incl. a negative-origin monitor, all codecs play in
   WebView2 incl. alpha, setSinkId present. Open issues: 5-9 s cold start, one 24 px offset, DPI disagreement, mixed DPI untested. A
   black screenshot early on was the page not having loaded yet (a bad assumption caught by adding request logging and a debug overlay).
7. **Productionising (this phase).** The owner: persist all intent for a host switch; write `AGENTS.md`, skills, GitHub pipelines,
   reproducible local builds (`scripts/build-deb` and friends) and documentation generation; learn from `hive-nmos-capture` and `tips-go`;
   judge reference repositories **by age and content, not by name** (the owner's correction: `reference-go-app-template` is an older
   snapshot of `tips-go`); idiomatic Go (`cmd/lumen/main.go`, `internal/`); decide whether the spikes are worth keeping; **write only
   inside this directory** (decision 012). Result: both spikes archived under `spikes/`; the Go module, scripts, packaging, CI, docs,
   skills and tests in this repository.

8. **Verifying the production skeleton.** `make bootstrap` (one `go install` hung on a stalled network fetch and was restarted by PID; the
   cache made the retry quick), `make docs`, then `make ci`. The gate found real things: formatters walking the repo-local module cache
   (decision 028); seven lint findings in the tests and Win32 code; and 21 standard-library vulnerabilities on Go 1.26.2 (decision 027).
   After `make ci` passed, the Linux build of the production app was **run** under WSLg through `make selftest`, which found three
   defects (decision 029). A `.deb`, a Windows zip and a docs bundle were built and inspected. The production app was **not** run on
   Windows (that would write outside the repo): the spike ran there; `make selftest` on a Windows host is the first task for the next host.

## Instructions and preferences stated by the owner (collected)
- Python + PySide6 first; later: prefers compiled/static (Go experience); wants research and a written assessment with tables.
- Windows is the primary platform; Ubuntu 24+, Windows 11, latest macOS (Apple Silicon) only; no "legacy support".
- "Construct as much as you can on your own, use make." Build pipelines aimed at GitHub CI; reproducible local builds.
- Production code is idiomatic Go: `cmd/lumen/main.go`, `internal/`.
- Review `hive-nmos-capture` (and `tips-go`, which is Linux-only so less transferable) for build/CI ideas; judge references by timestamps.
- Only ever write to this directory and its children.
- 100% persistence of intent and every decision for an independent pick-up: the owner will commit everything and change hosts.
- "Have a ponder, make a plan, and execute."

## Mistakes made and corrected (so they are not repeated)
- Said `sudo` was unavailable (it was available). Cause of the misdiagnosis: assuming a silent command meant failure.
- First Avalonia API use was stale (`SystemDecorations` renamed `WindowDecorations` in v12): compile the spike before trusting memory.
- Treated a black screenshot as a rendering fault; the page had not loaded (5-9 s cold start). Log requests before theorising.
- Chose the "template" reference by its name; the owner pointed out age matters (decision 013).
- Treated the first Linux self-test failure as an app fault: placement is impossible on Wayland, and screen IDs were not unique. Run on a real
  platform early, then separate platform limits from defects (decision 029).
- Weighted-total arithmetic in the first assessment was wrong and was recomputed and corrected before the owner saw the claim relied on.

## Environment facts at the time
WSL2 Ubuntu 24.04 (kernel 6.18), Go 1.26.2 on the host and 1.26.9 pinned in go.mod, passwordless `sudo`, `gh` logged in over SSH, GTK4 + WebKitGTK 6 dev packages installed, `dotnet`
10 and Node 24 present, Windows interop fixed by a systemd unit, VS Code Remote-WSL. See `docs/windows-dev.md` for the host record.

## State at the end of the session
Nothing is committed (the git repository has no commits). The first commit should be made by the owner; the pre-commit hook
(`make hooks`) then regenerates and stages the generated docs. Next steps: `docs/roadmap.md`, "First commands on a new host".
