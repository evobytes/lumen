"""Load slides from plain text: slides are separated by a line containing only ``---``."""

from __future__ import annotations

import re
from pathlib import Path

from lumen.model import Slide

_SEPARATOR = re.compile(r"^\s*---\s*$", re.MULTILINE)


def parse_text(text: str) -> list[Slide]:
    return [Slide(chunk.strip()) for chunk in _SEPARATOR.split(text) if chunk.strip()]


def load_file(path: str | Path) -> list[Slide]:
    return parse_text(Path(path).read_text(encoding="utf-8"))
