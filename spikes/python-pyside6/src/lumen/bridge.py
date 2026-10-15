"""Run callables on the Qt GUI thread from any other thread and wait for the result."""

from __future__ import annotations

from collections.abc import Callable
from concurrent.futures import Future
from typing import Any, TypeVar

from PySide6.QtCore import QObject, QThread, Signal, Slot

T = TypeVar("T")


class GuiBridge(QObject):
    """Create this on the GUI thread. ``call`` is safe from any thread."""

    _request = Signal(object, object)

    def __init__(self) -> None:
        super().__init__()
        self._request.connect(self._run)

    @Slot(object, object)
    def _run(self, fn: Callable[[], Any], future: Future[Any]) -> None:
        if not future.set_running_or_notify_cancel():
            return
        try:
            future.set_result(fn())
        except BaseException as exc:
            future.set_exception(exc)

    def call(self, fn: Callable[[], T], timeout: float = 5.0) -> T:
        if QThread.currentThread() is self.thread():
            return fn()
        future: Future[T] = Future()
        self._request.emit(fn, future)  # queued: the receiver lives in the GUI thread
        return future.result(timeout)
