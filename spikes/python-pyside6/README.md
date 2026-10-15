> **ARCHIVED SPIKE.** The first prototype (Python + PySide6), superseded by the Go product (decision 001); safe to delete. Not built by CI. See `docs/decisions.md` (decision 022) and the `spikes` skill.

# Lumen

A cross-platform (Linux, Windows, macOS) presentation projector built with Python and PySide6.

* **Two windows** – a *control* window (parent) with the slide list, live preview and
  navigation, and a frameless, always-on-top *output* window (floating child).
* **Multi-monitor** – pick an output screen and the output window covers that monitor. With a
  single monitor (or "Floating window" selected) it is a small draggable, resizable preview.
  Hot-plugging a monitor is handled; the chosen screen is remembered between runs.
* **Integrated web API** – a FastAPI server runs inside the app. Control it from a phone
  (`http://<host>:8765/`), scripts, or any HTTP client. Interactive docs are at `/docs`.

## Quick start

Requires Python 3.10+ and [uv](https://docs.astral.sh/uv/).

```sh
make install   # create the venv, install dependencies
make run       # launch the app
make check     # lint + tests
make dist      # build a standalone app with PyInstaller (dist/) and smoke-test it
```

`make help` lists every target. Without `make`, use `uv run lumen`, `uv run pytest`, etc.

### Command line

```
lumen [--host 127.0.0.1] [--port 8765] [--token SECRET] [--no-api]
```

The API binds to localhost by default. If you bind to another interface (e.g. `--host 0.0.0.0`
so a phone on the LAN can connect), set `--token` (or `LUMEN_TOKEN`) so only clients that send
`X-API-Token: SECRET` or `Authorization: Bearer SECRET` can control the app. The built-in
remote page accepts `/?token=SECRET`. The API is plain HTTP; do not expose it to untrusted
networks.

### Keyboard shortcuts (control window)

| Keys | Action |
| --- | --- |
| Right, PgDn, Space | Next slide |
| Left, PgUp, Backspace | Previous slide |
| B, . | Toggle blank (black) |
| Ctrl/Cmd+O | Open slides from a text file (slides separated by a line containing `---`) |

## Web API

All routes are under `/api/v1`; request/response bodies are JSON.

| Method & path | Body | Description |
| --- | --- | --- |
| `GET /status` | – | Index, count, blank flag, current slide, output screen |
| `GET /slides` | – | All slides |
| `PUT /slides` | `{"slides": [{"text": "...", "title": ""}]}` | Replace the slides |
| `POST /next`, `POST /prev` | – | Navigate (stops at the ends) |
| `POST /slides/{i}/show` | – | Jump to slide `i` (404 if out of range) |
| `POST /blank` | `{"blank": true}` (omit to toggle) | Blank the output |
| `GET /screens` | – | Connected monitors |
| `PUT /output` | `{"screen": 1}` or `{"screen": null}` | Choose output monitor / floating window |

```sh
curl -X POST localhost:8765/api/v1/next
curl -X PUT localhost:8765/api/v1/slides -H 'content-type: application/json' \
     -d '{"slides":[{"text":"Hello"},{"text":"World"}]}'
```

## Architecture

```
src/lumen/
  model.py        Slide / Presentation - pure Python, no Qt
  backend.py      Backend protocol the API depends on (+ in-memory implementation)
  controller.py   Qt state (QObject, signals, screens, settings) + QtBackend
  bridge.py       GuiBridge: run a callable on the GUI thread from the API thread
  api/            FastAPI app, uvicorn thread wrapper, phone remote page
  ui/             SlideCanvas, DisplayWindow (floating child), ControlWindow (parent)
  selftest.py     `lumen --self-test`: drives the running app through its own API
```

The API runs on a worker thread; every call is marshalled to the GUI thread through
`GuiBridge`, so Qt objects are only ever touched from the thread that owns them.

## CI / releases

* `.github/workflows/ci.yml` – ruff + pytest on Linux/Windows/macOS (Python 3.10 and 3.13),
  then a PyInstaller build per OS that is smoke-tested with `--self-test`.
* `.github/workflows/release.yml` – pushing a tag like `v0.1.0` builds all three platforms and
  publishes a GitHub release with the archives attached.

## Platform notes

* **Wayland (Linux)** compositors do not let applications position or pin windows, so the output
  window may not land on the chosen monitor or stay on top. Use X11 (`QT_QPA_PLATFORM=xcb`) for
  reliable multi-monitor placement.
* Builds are unsigned: macOS Gatekeeper and Windows SmartScreen will warn on first launch.
