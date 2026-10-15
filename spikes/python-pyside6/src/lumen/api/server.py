"""Run the FastAPI app with uvicorn on a background thread."""

from __future__ import annotations

import socket
import threading
import time

import uvicorn
from fastapi import FastAPI


class ApiError(RuntimeError):
    pass


def _free_port(host: str) -> int:
    with socket.socket() as s:
        s.bind((host, 0))
        return int(s.getsockname()[1])


class ApiServer:
    def __init__(self, app: FastAPI, host: str = "127.0.0.1", port: int = 8765) -> None:
        self._app = app
        self.host = host
        self.port = port
        self._server: uvicorn.Server | None = None
        self._thread: threading.Thread | None = None
        self._error: BaseException | None = None

    @property
    def url(self) -> str:
        shown = "127.0.0.1" if self.host in ("0.0.0.0", "::") else self.host
        return f"http://{shown}:{self.port}"

    def start(self, timeout: float = 10.0) -> None:
        if self.port == 0:
            self.port = _free_port(self.host)
        # log_config=None: uvicorn's default logging breaks in windowed (no-stderr) bundles.
        config = uvicorn.Config(
            self._app,
            host=self.host,
            port=self.port,
            log_config=None,
            access_log=False,
            lifespan="off",
            ws="none",
        )
        self._server = uvicorn.Server(config)
        self._thread = threading.Thread(target=self._run, name="lumen-api", daemon=True)
        self._thread.start()

        deadline = time.monotonic() + timeout
        while not self._server.started:
            if not self._thread.is_alive():
                raise ApiError(f"API server failed to start on {self.host}:{self.port}") from (
                    self._error
                )
            if time.monotonic() > deadline:
                raise ApiError("timed out starting API server")
            time.sleep(0.02)

    def stop(self) -> None:
        if self._server is not None:
            self._server.should_exit = True
        if self._thread is not None:
            self._thread.join(timeout=5)

    def _run(self) -> None:
        assert self._server is not None
        try:
            self._server.run()
        except BaseException as exc:
            self._error = exc
