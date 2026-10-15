import threading

import pytest
from PySide6.QtWidgets import QApplication

from lumen.bridge import GuiBridge
from lumen.controller import Controller
from lumen.model import Presentation, Slide
from lumen.ui.canvas import fit_font
from lumen.ui.control_window import ControlWindow


@pytest.fixture
def controller(qtbot):
    return Controller(Presentation(Slide(f"slide {i}") for i in range(4)))


@pytest.fixture
def window(qtbot, controller):
    w = ControlWindow(controller)
    qtbot.addWidget(w)
    w.show()
    return w


def test_list_follows_controller(window, controller):
    assert window._list.count() == 4
    controller.next()
    assert window._list.currentRow() == 1
    controller.set_slides([Slide("only")])
    assert window._list.count() == 1


def test_clicking_list_navigates(window, controller):
    window._list.setCurrentRow(3)
    assert controller.presentation.index == 3
    assert not window._next_btn.isEnabled()


def test_blank_button_and_shortcut_state(window, controller):
    window._blank_btn.click()
    assert controller.presentation.blanked
    controller.set_blank(False)
    assert not window._blank_btn.isChecked()


def test_display_is_child_and_toplevel(window):
    d = window.display
    assert d.parent() is window
    assert d.isWindow() and d.isVisible()


def test_floating_then_cover_screen(window, controller):
    screen = QApplication.screens()[0]
    d = window.display
    window._show_output.setChecked(True)
    controller.set_output_index(None)
    assert d.is_floating
    controller.set_output_index(0)
    assert not d.is_floating
    assert d.geometry() == screen.geometry()
    controller.set_output_index(None)
    assert d.is_floating and d.width() == 480


def test_hide_output(window):
    window._show_output.setChecked(False)
    assert not window.display.isVisible()
    window._show_output.setChecked(True)
    assert window.display.isVisible()


def test_invalid_output_index(controller):
    with pytest.raises(IndexError):
        controller.set_output_index(99)


def test_output_combo_lists_screens(window):
    n = len(QApplication.screens())
    assert window._output_combo.count() == n + 1


def test_fit_font_shrinks_for_long_text(window):
    area = window.rect().adjusted(0, 0, -400, -300)
    short = fit_font(window.font(), "hi", area).pixelSize()
    long = fit_font(window.font(), "word " * 200, area).pixelSize()
    assert long < short


def test_bridge_runs_on_gui_thread(qtbot):
    bridge = GuiBridge()
    main = threading.current_thread()
    result = {}

    def worker():
        result["thread"] = bridge.call(threading.current_thread)

    t = threading.Thread(target=worker)
    t.start()
    qtbot.waitUntil(lambda: "thread" in result, timeout=3000)
    t.join()
    assert result["thread"] is main


def test_bridge_propagates_exceptions(qtbot):
    bridge = GuiBridge()
    result = {}

    def worker():
        try:
            bridge.call(lambda: 1 / 0)
        except ZeroDivisionError as exc:
            result["exc"] = exc

    t = threading.Thread(target=worker)
    t.start()
    qtbot.waitUntil(lambda: "exc" in result, timeout=3000)
    t.join()
