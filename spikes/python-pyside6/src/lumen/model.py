"""Pure-Python presentation model (no Qt), so it can be unit-tested and shared with the API."""

from __future__ import annotations

from collections.abc import Iterable
from dataclasses import dataclass
from typing import Any


@dataclass(frozen=True, slots=True)
class Slide:
    text: str
    title: str = ""

    @property
    def label(self) -> str:
        """Short name for lists: the explicit title, else the first line of text."""
        if self.title:
            return self.title
        stripped = self.text.strip()
        return stripped.splitlines()[0] if stripped else "(empty)"

    def to_dict(self) -> dict[str, Any]:
        return {"title": self.title, "text": self.text, "label": self.label}


class Presentation:
    """An ordered list of slides, a current position and a blank (black screen) flag."""

    def __init__(self, slides: Iterable[Slide] = ()) -> None:
        self._slides: tuple[Slide, ...] = tuple(slides)
        self._index = 0
        self._blanked = False

    @property
    def slides(self) -> tuple[Slide, ...]:
        return self._slides

    @property
    def index(self) -> int:
        return self._index

    @property
    def blanked(self) -> bool:
        return self._blanked

    @property
    def current(self) -> Slide | None:
        return self._slides[self._index] if self._slides else None

    def set_slides(self, slides: Iterable[Slide]) -> None:
        self._slides = tuple(slides)
        self._index = 0

    def next(self) -> bool:
        """Advance one slide; returns False (and stays put) on the last slide."""
        return self._move(self._index + 1)

    def prev(self) -> bool:
        return self._move(self._index - 1)

    def goto(self, index: int) -> None:
        if not 0 <= index < len(self._slides):
            raise IndexError(f"slide index {index} out of range (0..{len(self._slides) - 1})")
        self._index = index

    def set_blank(self, value: bool | None = None) -> bool:
        """Set the blank flag, or toggle it when ``value`` is None. Returns the new state."""
        self._blanked = (not self._blanked) if value is None else value
        return self._blanked

    def snapshot(self) -> dict[str, Any]:
        slide = self.current
        return {
            "index": self._index,
            "count": len(self._slides),
            "blanked": self._blanked,
            "slide": slide.to_dict() if slide else None,
        }

    def _move(self, index: int) -> bool:
        if 0 <= index < len(self._slides):
            self._index = index
            return True
        return False


SAMPLE_SLIDES = (
    Slide("Welcome to Lumen\nControl window on the left, floating output window on the right"),
    Slide(
        "Open this app's address in a phone browser to control the slides remotely.\n"
        "API docs are served at /docs",
        title="Web API",
    ),
    Slide("Multiple monitors\nPick an output screen and the display window will cover it"),
    Slide("Thank you"),
)
