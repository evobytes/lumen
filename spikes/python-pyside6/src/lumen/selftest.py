"""``lumen --self-test``: boot the real app, drive it through its own web API, then exit.

Used by CI to verify packaged builds. Exit code 0 means the GUI, the API and the
API -> GUI thread bridge all work.
"""

from __future__ import annotations

import contextlib
import json
import sys
import threading
import urllib.request
from collections.abc import Callable
from typing import Any

from PySide6.QtWidgets import QApplication

from lumen.bridge import GuiBridge
from lumen.ui.control_window import ControlWindow


def _request(url: str, method: str = "GET", body: dict[str, Any] | None = None) -> Any:
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(
        url, data=data, method=method, headers={"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req, timeout=5) as resp:
        raw = resp.read().decode()
    return json.loads(raw) if resp.headers.get_content_type() == "application/json" else raw


def start(base_url: str, window: ControlWindow, bridge: GuiBridge, token: str | None) -> None:
    """Run the checks on a worker thread; exits the Qt app with the result code."""

    def check(cond: bool, message: str) -> None:
        if not cond:
            raise AssertionError(message)

    def work() -> int:
        api = f"{base_url}/api/v1"
        status = _request(f"{api}/status")
        check(status["count"] > 0 and status["index"] == 0, f"unexpected status {status}")
        check(_request(f"{api}/next", "POST")["index"] == 1, "next did not advance")
        check(bridge.call(lambda: window.controller.presentation.index) == 1, "GUI not updated")
        check(_request(f"{api}/blank", "POST", {"blank": True})["blanked"] is True, "blank failed")
        check(_request(f"{api}/blank", "POST", {"blank": False})["blanked"] is False, "unblank")
        check(isinstance(_request(f"{api}/screens"), list), "screens not a list")
        check("Lumen" in _request(f"{base_url}/"), "remote page missing")
        check(bridge.call(window.display.isVisible), "display window not visible")
        return 0

    def run() -> None:
        code = 0
        try:
            code = work()
        except Exception as exc:
            print(f"self-test FAILED: {exc!r}", file=sys.stderr)
            code = 1
        print("self-test OK" if code == 0 else "self-test failed", file=sys.stderr)
        _exit_gui(bridge, code)

    threading.Thread(target=run, name="lumen-selftest", daemon=True).start()


def watchdog(bridge: GuiBridge, seconds: float = 30.0) -> Callable[[], None]:
    timer = threading.Timer(seconds, lambda: _exit_gui(bridge, 2))
    timer.daemon = True
    timer.start()
    return timer.cancel


def _exit_gui(bridge: GuiBridge, code: int) -> None:
    with contextlib.suppress(Exception):
        bridge.call(lambda: QApplication.exit(code))
