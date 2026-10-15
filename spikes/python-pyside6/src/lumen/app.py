"""Application entry point."""

from __future__ import annotations

import argparse
import os
import sys

from PySide6.QtCore import QSettings
from PySide6.QtWidgets import QApplication

from lumen import __version__, selftest
from lumen.api import ApiError, ApiServer, create_app
from lumen.bridge import GuiBridge
from lumen.controller import Controller, QtBackend
from lumen.ui.control_window import ControlWindow


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(prog="lumen", description="Lumen presentation projector")
    p.add_argument("--version", action="version", version=f"%(prog)s {__version__}")
    p.add_argument("--host", default="127.0.0.1", help="API bind address (default: %(default)s)")
    p.add_argument(
        "--port", type=int, default=8765, help="API port, 0 = any (default: %(default)s)"
    )
    p.add_argument(
        "--token",
        default=os.environ.get("LUMEN_TOKEN"),
        help="require this token on /api/v1 (or set LUMEN_TOKEN); "
        "recommended when binding beyond localhost",
    )
    p.add_argument("--no-api", action="store_true", help="do not start the web API")
    p.add_argument("--self-test", action="store_true", help=argparse.SUPPRESS)
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    if args.self_test:
        args.port = 0
        args.no_api = False

    app = QApplication(sys.argv[:1])
    app.setApplicationName("Lumen")
    app.setOrganizationName("Lumen")
    app.setApplicationVersion(__version__)

    bridge = GuiBridge()
    settings = None if args.self_test else QSettings()  # self-test must not touch user prefs
    controller = Controller(settings=settings)
    window = ControlWindow(controller)

    server: ApiServer | None = None
    if not args.no_api:
        server = ApiServer(
            create_app(QtBackend(controller, bridge), args.token), args.host, args.port
        )
        try:
            server.start()
            window.set_api_status(f"API: {server.url}  (docs: {server.url}/docs)")
        except ApiError as exc:
            window.set_api_status("API: unavailable")
            print(f"warning: {exc}", file=sys.stderr)
            server = None

    window.show()

    cancel_watchdog = None
    if args.self_test:
        if server is None:
            print("self-test FAILED: API did not start", file=sys.stderr)
            return 1
        cancel_watchdog = selftest.watchdog(bridge)
        selftest.start(server.url, window, bridge, args.token)

    try:
        return app.exec()
    finally:
        if cancel_watchdog:
            cancel_watchdog()
        if server is not None:
            server.stop()


if __name__ == "__main__":
    sys.exit(main())
