"""Widget that renders the current slide, auto-sizing the text to fit."""

from __future__ import annotations

from PySide6.QtCore import QRect, Qt
from PySide6.QtGui import QColor, QFont, QFontMetrics, QPainter, QPaintEvent
from PySide6.QtWidgets import QWidget

from lumen.controller import Controller

_FLAGS = int(Qt.AlignmentFlag.AlignCenter | Qt.TextFlag.TextWordWrap)


def fit_font(base: QFont, text: str, area: QRect, max_px: int = 400) -> QFont:
    """Largest font (by pixel size) for which ``text`` fits inside ``area``."""
    font = QFont(base)
    lo, hi = 6, max(6, min(max_px, area.height()))
    while lo < hi:
        mid = (lo + hi + 1) // 2
        font.setPixelSize(mid)
        r = QFontMetrics(font).boundingRect(area, _FLAGS, text)
        if r.height() <= area.height() and r.width() <= area.width():
            lo = mid
        else:
            hi = mid - 1
    font.setPixelSize(lo)
    return font


class SlideCanvas(QWidget):
    def __init__(self, controller: Controller, parent: QWidget | None = None) -> None:
        super().__init__(parent)
        self._controller = controller
        self.setMinimumSize(160, 90)
        controller.changed.connect(self.update)

    def paintEvent(self, event: QPaintEvent) -> None:
        painter = QPainter(self)
        painter.fillRect(self.rect(), QColor("black"))
        p = self._controller.presentation
        slide = p.current
        if p.blanked or slide is None:
            return
        margin = round(min(self.width(), self.height()) * 0.06)
        area = self.rect().adjusted(margin, margin, -margin, -margin)
        painter.setFont(fit_font(self.font(), slide.text, area))
        painter.setPen(QColor("white"))
        painter.drawText(area, _FLAGS, slide.text)
