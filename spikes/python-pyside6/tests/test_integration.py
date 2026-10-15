"""Real uvicorn server -> QtBackend -> GuiBridge -> Controller, over actual HTTP."""

import threading

import httpx
import pytest

from lumen.api import ApiServer, create_app
from lumen.bridge import GuiBridge
from lumen.controller import Controller, QtBackend
from lumen.model import Presentation, Slide


@pytest.fixture
def stack(qtbot):
    controller = Controller(Presentation(Slide(f"s{i}") for i in range(3)))
    bridge = GuiBridge()
    server = ApiServer(create_app(QtBackend(controller, bridge)), "127.0.0.1", 0)
    server.start()
    yield controller, server
    server.stop()


def request_in_thread(qtbot, fn):
    """Run a blocking HTTP call off-thread while the Qt event loop keeps spinning."""
    box = {}

    def run():
        try:
            box["result"] = fn()
        except Exception as exc:
            box["error"] = exc

    threading.Thread(target=run, daemon=True).start()
    qtbot.waitUntil(lambda: bool(box), timeout=5000)
    if "error" in box:
        raise box["error"]
    return box["result"]


def test_http_drives_gui_state(qtbot, stack):
    controller, server = stack
    r = request_in_thread(qtbot, lambda: httpx.post(f"{server.url}/api/v1/next"))
    assert r.status_code == 200 and r.json()["index"] == 1
    assert controller.presentation.index == 1

    r = request_in_thread(
        qtbot, lambda: httpx.post(f"{server.url}/api/v1/blank", json={"blank": True})
    )
    assert r.json()["blanked"] and controller.presentation.blanked


def test_screens_and_output_over_http(qtbot, stack):
    controller, server = stack
    screens = request_in_thread(qtbot, lambda: httpx.get(f"{server.url}/api/v1/screens")).json()
    assert len(screens) >= 1 and {"index", "name", "width", "height"} <= screens[0].keys()

    r = request_in_thread(
        qtbot, lambda: httpx.put(f"{server.url}/api/v1/output", json={"screen": 0})
    )
    assert r.json()["output"]["screen"] == 0 and controller.output_index == 0
    r = request_in_thread(
        qtbot, lambda: httpx.put(f"{server.url}/api/v1/output", json={"screen": 42})
    )
    assert r.status_code == 422


def test_port_conflict_raises():
    from lumen.api import ApiError

    backend_app = create_app(__import__("lumen.backend", fromlist=["x"]).PresentationBackend())
    first = ApiServer(backend_app, "127.0.0.1", 0)
    first.start()
    try:
        with pytest.raises(ApiError):
            ApiServer(backend_app, "127.0.0.1", first.port).start()
    finally:
        first.stop()
