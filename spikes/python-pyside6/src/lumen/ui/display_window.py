"""The output window: a frameless, always-on-top child of the control window.

Two modes:
* floating - a small draggable/resizable preview that hovers over the control window;
* on a screen - covers a whole (second) monitor, without OS fullscreen. Real fullscreen
  creates a separate Space on macOS and steals the main display on some Windows setups.
"""

from __future__ import annotations

from PySide6.QtCore import QPoint, QRect, QSize, Qt
from PySide6.QtGui import QMouseEvent, QPainter, QPen, QResizeEvent, QScreen
from PySide6.QtWidgets import QSizeGrip, QWidget

from lumen.controller import Controller
from lumen.ui.canvas import SlideCanvas

_FLOATING_SIZE = QSize(480, 270)


class DisplayWindow(QWidget):
    def __init__(self, controller: Controller, parent: QWidget) -> None:
        super().__init__(
            parent,
            Qt.WindowType.Window  # a real top-level window, but owned by (floats over) parent
            | Qt.WindowType.Tool
            | Qt.WindowType.FramelessWindowHint
            | Qt.WindowType.WindowStaysOnTopHint,
        )
        self.setWindowTitle("Lumen output")
        self.setAttribute(Qt.WidgetAttribute.WA_ShowWithoutActivating)
        self.setAttribute(
            Qt.WidgetAttribute.WA_MacAlwaysShowToolWindow
        )  # stay up when app inactive
        self._canvas = SlideCanvas(controller, self)
        self._grip = QSizeGrip(self)
        self._floating = True
        self._float_geometry: QRect | None = None
        self.resize(_FLOATING_SIZE)

    @property
    def is_floating(self) -> bool:
        return self._floating

    def float_over(self, anchor: QWidget) -> None:
        """Switch to the small floating mode; restores the last floating position."""
        if not self._floating and self._float_geometry is not None:
            self._float_geometry = QRect(self._float_geometry)
        self._floating = True
        self._grip.show()
        self.unsetCursor()
        if self._float_geometry is None:
            top_right = anchor.frameGeometry().topRight()
            self._float_geometry = QRect(QPoint(top_right.x() + 12, top_right.y()), _FLOATING_SIZE)
        self.setScreen(anchor.screen())
        self.setGeometry(self._float_geometry)

    def cover(self, screen: QScreen) -> None:
        """Cover ``screen`` entirely."""
        if self._floating:
            self._float_geometry = self.geometry()
        self._floating = False
        self._grip.hide()
        self.setCursor(Qt.CursorShape.BlankCursor)
        self.setScreen(screen)
        self.setGeometry(screen.geometry())

    # -- floating-mode interaction ----------------------------------------------------------
    def mousePressEvent(self, event: QMouseEvent) -> None:
        if self._floating and event.button() == Qt.MouseButton.LeftButton:
            handle = self.windowHandle()
            if handle is not None:
                handle.startSystemMove()  # native move: also works on Wayland
                return
        super().mousePressEvent(event)

    def moveEvent(self, event) -> None:
        self._remember_floating()
        super().moveEvent(event)

    def resizeEvent(self, event: QResizeEvent) -> None:
        self._canvas.setGeometry(self.rect())
        self._grip.move(self.width() - self._grip.width(), self.height() - self._grip.height())
        self._grip.raise_()
        self._remember_floating()
        super().resizeEvent(event)

    def paintEvent(self, event) -> None:
        if self._floating:
            painter = QPainter(self)
            painter.setPen(QPen(Qt.GlobalColor.darkGray, 2))
            painter.drawRect(self.rect().adjusted(1, 1, -1, -1))

    def _remember_floating(self) -> None:
        if self._floating and self.isVisible():
            self._float_geometry = self.geometry()
