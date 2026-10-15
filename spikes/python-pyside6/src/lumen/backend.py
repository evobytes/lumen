"""The interface the web API talks to, plus a Qt-free in-memory implementation.

The API runs on a worker thread, so everything it needs goes through ``Backend``. The Qt
implementation (``lumen.controller.QtBackend``) marshals calls onto the GUI thread; the
``PresentationBackend`` here is used by tests and anywhere no GUI is needed.
"""

from __future__ import annotations

import threading
from typing import Any, Protocol

from lumen.model import Presentation, Slide


class Backend(Protocol):
    def status(self) -> dict[str, Any]: ...
    def slides(self) -> list[dict[str, Any]]: ...
    def set_slides(self, slides: list[Slide]) -> dict[str, Any]: ...
    def next(self) -> dict[str, Any]: ...
    def prev(self) -> dict[str, Any]: ...
    def goto(self, index: int) -> dict[str, Any]:
        """Raises IndexError if out of range."""
        ...

    def blank(self, value: bool | None) -> dict[str, Any]: ...
    def screens(self) -> list[dict[str, Any]]: ...
    def set_output(self, screen: int | None) -> dict[str, Any]:
        """Raises IndexError for an unknown screen. ``None`` selects the floating window."""
        ...


class PresentationBackend:
    """Thread-safe in-memory backend with no screens."""

    def __init__(self, presentation: Presentation | None = None) -> None:
        self._p = presentation or Presentation()
        self._lock = threading.Lock()

    def status(self) -> dict[str, Any]:
        with self._lock:
            return {**self._p.snapshot(), "output": {"screen": None}}

    def slides(self) -> list[dict[str, Any]]:
        with self._lock:
            return [s.to_dict() for s in self._p.slides]

    def set_slides(self, slides: list[Slide]) -> dict[str, Any]:
        with self._lock:
            self._p.set_slides(slides)
        return self.status()

    def next(self) -> dict[str, Any]:
        with self._lock:
            self._p.next()
        return self.status()

    def prev(self) -> dict[str, Any]:
        with self._lock:
            self._p.prev()
        return self.status()

    def goto(self, index: int) -> dict[str, Any]:
        with self._lock:
            self._p.goto(index)
        return self.status()

    def blank(self, value: bool | None) -> dict[str, Any]:
        with self._lock:
            self._p.set_blank(value)
        return self.status()

    def screens(self) -> list[dict[str, Any]]:
        return []

    def set_output(self, screen: int | None) -> dict[str, Any]:
        if screen is not None:
            raise IndexError("no screens available")
        return self.status()
