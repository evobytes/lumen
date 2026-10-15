---
name: presentation-control
description: Design and constraints for controlling PowerPoint (COM on Windows, AppleScript on macOS) and other external presentation apps from lumen. Not built yet; read before implementing. Use when adding presentation file support.
---

# Presentation control (design; not built: milestone M3)

Source of the quirks below: OpenLP's controllers (`powerpointcontroller.py`, `powerpointmaccontroller.py` on GitLab), read in full on 2026-10-13.

## Shape
A small interface in its own package (Load, Start on a screen, Next, Prev, Goto, Blank/Unblank, Stop, plus slide count/titles/notes), implemented per platform behind build tags with an inert twin, and a **fake** for tests. PowerPoint cannot run in GitHub CI (no Office on runners), so the real implementations are covered by unit tests of the pure parts and a manual QA checklist on a machine with Office.

## Windows: COM via `go-ole` (late-bound `IDispatch`)
- Discover `PowerPoint.Application` (try versioned ProgIDs `.16`, `.15`, `.14`, `.12` like OpenLP).
- `Presentations.Open`; start with `SlideShowSettings.Run()` (`ShowType` fullscreen or windowed, `ShowPresenterView = 0`); control with `SlideShowWindow.View.Next/Previous/GotoSlide`; blank with `View.State = 3` (unblank sets 1); stop with `View.Exit()`.
- **Choose the monitor** by writing the `DisplayMonitor` registry value under `HKCU\Software\Microsoft\Office\<ver>\PowerPoint\Options` before `Run()` and restoring it after (OpenLP does this; the device name comes from `EnumDisplayMonitors`).
- Quirks to design around: "ghost" windows after windowed shows (close with `WM_CLOSE`); `Slide` is unavailable on the end slide and `CurrentShowPosition` can be 0 during transitions in 2013; unblank is broken in 2010 (re-navigate); the show **pauses (`State = 2`) when it loses focus**, so re-`Activate()`; hidden slides must be skipped consistently (keep an index map); cloud-hosted files compare by name.
- Thumbnails via `slide.Export`.

## macOS: AppleScript via `osascript`
Needs PowerPoint 15+. AppleScript **cannot jump to a slide**: OpenLP types the slide number as keystrokes plus Return (needs Accessibility permission, to be requested). Thumbnails via PDF export. Keynote has its own scripting.

## The output window must yield
PowerPoint opens its own native fullscreen window on the output monitor. Our output is always-on-top, so while an external app owns the screen: hide or lower the output, pass **blank** through the controller (not the page), and restore on stop. Model it as an explicit "output owner" state in `internal/state` (lumen | external app). Capturing PowerPoint for NDI/SDI is a desktop capture, fill only (no alpha): `broadcast-outputs`.

## Fallbacks
A PDF/image path (render pages) for machines without PowerPoint; LibreOffice is **not** required (owner's decision).

## Security
Never launch what a request names: the file comes from the library by ID; PowerPoint is found via COM, not a path from a request. Macros and external links in decks are a risk: open read-only, and say so.
