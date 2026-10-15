import pytest

from lumen.loader import parse_text
from lumen.model import Presentation, Slide


def make(n: int = 3) -> Presentation:
    return Presentation(Slide(f"slide {i}") for i in range(n))


def test_navigation_stops_at_ends():
    p = make()
    assert not p.prev()
    assert p.next() and p.next()
    assert not p.next()
    assert p.index == 2


def test_goto_validates():
    p = make()
    p.goto(2)
    assert p.index == 2
    with pytest.raises(IndexError):
        p.goto(3)
    with pytest.raises(IndexError):
        p.goto(-1)


def test_blank_toggle_and_set():
    p = make()
    assert p.set_blank() is True
    assert p.set_blank() is False
    assert p.set_blank(True) is True
    assert p.set_blank(True) is True


def test_set_slides_resets_position():
    p = make()
    p.goto(2)
    p.set_slides([Slide("a")])
    assert p.index == 0 and p.snapshot()["count"] == 1


def test_empty_presentation():
    p = Presentation()
    assert p.current is None
    assert p.snapshot()["slide"] is None
    assert not p.next()


def test_label_prefers_title_then_first_line():
    assert Slide("hello\nworld").label == "hello"
    assert Slide("hello", title="T").label == "T"
    assert Slide("  ").label == "(empty)"


def test_parse_text_splits_on_separator_lines():
    slides = parse_text("one\nline\n---\n\n two \n  ---  \n\n---\nthree")
    assert [s.text for s in slides] == ["one\nline", "two", "three"]
