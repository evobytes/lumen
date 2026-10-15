"""Qt-side application state. Lives on the GUI thread; the web API reaches it via QtBackend."""

from __future__ import annotations

from typing import Any

from PySide6.QtCore import QObject, QSettings, Signal
from PySide6.QtGui import QGuiApplication, QScreen

from lumen.bridge import GuiBridge
from lumen.model import SAMPLE_SLIDES, Presentation, Slide

_OUTPUT_KEY = "output/screen_name"  # absent = never chosen, "" = floating window


class Controller(QObject):
    changed = Signal()  # slides, position or blank state changed
    output_changed = Signal()  # output screen selection or the set of connected screens changed

    def __init__(
        self,
        presentation: Presentation | None = None,
        settings: QSettings | None = None,
    ) -> None:
        super().__init__()
        self.presentation = presentation or Presentation(SAMPLE_SLIDES)
        self._settings = settings
        self._output: QScreen | None = self._initial_output()
        app = QGuiApplication.instance()
        if isinstance(app, QGuiApplication):
            app.screenAdded.connect(lambda _s: self.output_changed.emit())
            app.screenRemoved.connect(self._on_screen_removed)

    # -- slides ---------------------------------------------------------------------------
    def next(self) -> None:
        if self.presentation.next():
            self.changed.emit()

    def prev(self) -> None:
        if self.presentation.prev():
            self.changed.emit()

    def goto(self, index: int) -> None:
        self.presentation.goto(index)
        self.changed.emit()

    def set_blank(self, value: bool | None = None) -> None:
        self.presentation.set_blank(value)
        self.changed.emit()

    def set_slides(self, slides: list[Slide]) -> None:
        self.presentation.set_slides(slides)
        self.changed.emit()

    # -- screens --------------------------------------------------------------------------
    @staticmethod
    def qscreens() -> list[QScreen]:
        return QGuiApplication.screens()

    @property
    def output_screen(self) -> QScreen | None:
        return self._output

    @property
    def output_index(self) -> int | None:
        screens = self.qscreens()
        return screens.index(self._output) if self._output in screens else None

    def set_output_index(self, index: int | None) -> None:
        screens = self.qscreens()
        if index is None:
            self._output = None
        elif 0 <= index < len(screens):
            self._output = screens[index]
        else:
            raise IndexError(f"screen index {index} out of range (0..{len(screens) - 1})")
        if self._settings is not None:
            self._settings.setValue(_OUTPUT_KEY, self._output.name() if self._output else "")
        self.output_changed.emit()

    def screen_infos(self) -> list[dict[str, Any]]:
        primary = QGuiApplication.primaryScreen()
        infos = []
        for i, s in enumerate(self.qscreens()):
            g = s.geometry()
            infos.append(
                {
                    "index": i,
                    "name": s.name(),
                    "x": g.x(),
                    "y": g.y(),
                    "width": g.width(),
                    "height": g.height(),
                    "primary": s == primary,
                }
            )
        return infos

    def status(self) -> dict[str, Any]:
        return {**self.presentation.snapshot(), "output": {"screen": self.output_index}}

    # -- internals ------------------------------------------------------------------------
    def _initial_output(self) -> QScreen | None:
        screens = self.qscreens()
        if self._settings is not None and self._settings.contains(_OUTPUT_KEY):
            saved = str(self._settings.value(_OUTPUT_KEY, ""))
            if not saved:
                return None
            for s in screens:
                if s.name() == saved:
                    return s
        # Default: first non-primary screen if there is one, else the floating window.
        primary = QGuiApplication.primaryScreen()
        return next((s for s in screens if s != primary), None)

    def _on_screen_removed(self, screen: QScreen) -> None:
        if screen == self._output:
            self._output = None
        self.output_changed.emit()


class QtBackend:
    """Backend implementation for the web API: every call runs on the GUI thread."""

    def __init__(self, controller: Controller, bridge: GuiBridge) -> None:
        self._c = controller
        self._call = bridge.call

    def status(self) -> dict[str, Any]:
        return self._call(self._c.status)

    def slides(self) -> list[dict[str, Any]]:
        return self._call(lambda: [s.to_dict() for s in self._c.presentation.slides])

    def set_slides(self, slides: list[Slide]) -> dict[str, Any]:
        return self._call(lambda: (self._c.set_slides(slides), self._c.status())[1])

    def next(self) -> dict[str, Any]:
        return self._call(lambda: (self._c.next(), self._c.status())[1])

    def prev(self) -> dict[str, Any]:
        return self._call(lambda: (self._c.prev(), self._c.status())[1])

    def goto(self, index: int) -> dict[str, Any]:
        return self._call(lambda: (self._c.goto(index), self._c.status())[1])

    def blank(self, value: bool | None) -> dict[str, Any]:
        return self._call(lambda: (self._c.set_blank(value), self._c.status())[1])

    def screens(self) -> list[dict[str, Any]]:
        return self._call(self._c.screen_infos)

    def set_output(self, screen: int | None) -> dict[str, Any]:
        return self._call(lambda: (self._c.set_output_index(screen), self._c.status())[1])
