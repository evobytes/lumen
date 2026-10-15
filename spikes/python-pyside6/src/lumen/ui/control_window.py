"""The parent window: slide list, live preview, navigation and output-screen selection."""

from __future__ import annotations

from PySide6.QtGui import QAction, QCloseEvent, QKeySequence, QShortcut
from PySide6.QtWidgets import (
    QApplication,
    QCheckBox,
    QComboBox,
    QFileDialog,
    QFormLayout,
    QHBoxLayout,
    QLabel,
    QListWidget,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QSplitter,
    QVBoxLayout,
    QWidget,
)

from lumen import __version__
from lumen.controller import Controller
from lumen.loader import load_file
from lumen.ui.canvas import SlideCanvas
from lumen.ui.display_window import DisplayWindow


class ControlWindow(QMainWindow):
    def __init__(self, controller: Controller) -> None:
        super().__init__()
        self.controller = controller
        self.setWindowTitle("Lumen")
        self.resize(900, 560)

        self.display = DisplayWindow(controller, self)  # the floating child
        self._shown_slides: tuple | None = None

        self._list = QListWidget()
        self._list.currentRowChanged.connect(self._on_row_changed)

        self._preview = SlideCanvas(controller)
        self._prev_btn = QPushButton("◀ Previous")
        self._blank_btn = QPushButton("Blank")
        self._blank_btn.setCheckable(True)
        self._next_btn = QPushButton("Next ▶")
        self._prev_btn.clicked.connect(controller.prev)
        self._next_btn.clicked.connect(controller.next)
        self._blank_btn.clicked.connect(lambda: controller.set_blank(self._blank_btn.isChecked()))

        buttons = QHBoxLayout()
        for b in (self._prev_btn, self._blank_btn, self._next_btn):
            b.setMinimumHeight(40)
            buttons.addWidget(b)

        self._output_combo = QComboBox()
        self._output_combo.activated.connect(self._on_output_chosen)
        self._show_output = QCheckBox("Show output window")
        self._show_output.setChecked(True)
        self._show_output.toggled.connect(self._apply_output)
        form = QFormLayout()
        form.addRow("Output:", self._output_combo)
        form.addRow("", self._show_output)

        right = QWidget()
        layout = QVBoxLayout(right)
        layout.addWidget(QLabel("Live preview"))
        layout.addWidget(self._preview, 1)
        layout.addLayout(buttons)
        layout.addLayout(form)

        splitter = QSplitter()
        splitter.addWidget(self._list)
        splitter.addWidget(right)
        splitter.setStretchFactor(1, 1)
        splitter.setSizes([280, 620])
        self.setCentralWidget(splitter)

        self._api_label = QLabel("API: not running")
        self.statusBar().addPermanentWidget(self._api_label)

        self._build_menus()
        self._build_shortcuts()

        controller.changed.connect(self._sync)
        controller.output_changed.connect(self._sync_output)
        self._sync()
        self._sync_output()

    # -- public ---------------------------------------------------------------------------
    def set_api_status(self, text: str) -> None:
        self._api_label.setText(text)

    def showEvent(self, event) -> None:
        super().showEvent(event)
        self._apply_output()

    def closeEvent(self, event: QCloseEvent) -> None:
        self.display.close()
        event.accept()
        QApplication.quit()

    # -- construction helpers -------------------------------------------------------------
    def _build_menus(self) -> None:
        file_menu = self.menuBar().addMenu("&File")
        open_action = QAction("&Open slides…", self)
        open_action.setShortcut(QKeySequence.StandardKey.Open)
        open_action.triggered.connect(self._open_file)
        quit_action = QAction("&Quit", self)
        quit_action.setShortcut(QKeySequence.StandardKey.Quit)
        quit_action.triggered.connect(self.close)
        file_menu.addAction(open_action)
        file_menu.addSeparator()
        file_menu.addAction(quit_action)

        help_menu = self.menuBar().addMenu("&Help")
        about = QAction("&About", self)
        about.triggered.connect(
            lambda: QMessageBox.about(self, "About Lumen", f"Lumen {__version__}")
        )
        help_menu.addAction(about)

    def _build_shortcuts(self) -> None:
        c = self.controller
        for keys, fn in (
            (("Right", "PgDown", "Space"), c.next),
            (("Left", "PgUp", "Backspace"), c.prev),
            (("B", "."), c.set_blank),
        ):
            for key in keys:
                sc = QShortcut(QKeySequence(key), self)
                sc.activated.connect(fn)

    # -- slots ----------------------------------------------------------------------------
    def _open_file(self) -> None:
        path, _ = QFileDialog.getOpenFileName(
            self, "Open slides", "", "Text files (*.txt *.md);;All files (*)"
        )
        if not path:
            return
        try:
            slides = load_file(path)
        except (OSError, UnicodeDecodeError) as exc:
            QMessageBox.warning(self, "Open slides", f"Could not read {path}:\n{exc}")
            return
        if not slides:
            QMessageBox.information(self, "Open slides", "No slides found in that file.")
            return
        self.controller.set_slides(slides)

    def _on_row_changed(self, row: int) -> None:
        p = self.controller.presentation
        if 0 <= row < len(p.slides) and row != p.index:
            self.controller.goto(row)

    def _on_output_chosen(self, combo_index: int) -> None:
        self.controller.set_output_index(self._output_combo.itemData(combo_index))

    def _sync(self) -> None:
        p = self.controller.presentation
        self._list.blockSignals(True)
        if p.slides is not self._shown_slides:
            self._shown_slides = p.slides
            self._list.clear()
            for i, s in enumerate(p.slides):
                self._list.addItem(f"{i + 1}. {s.label}")
        self._list.setCurrentRow(p.index if p.slides else -1)
        self._list.blockSignals(False)
        self._blank_btn.setChecked(p.blanked)
        self._prev_btn.setEnabled(p.index > 0)
        self._next_btn.setEnabled(p.index + 1 < len(p.slides))

    def _sync_output(self) -> None:
        self._output_combo.blockSignals(True)
        self._output_combo.clear()
        self._output_combo.addItem("Floating window", None)
        for info in self.controller.screen_infos():
            label = f"Screen {info['index'] + 1}: {info['name']} ({info['width']}x{info['height']})"
            if info["primary"]:
                label += " (primary)"
            self._output_combo.addItem(label, info["index"])
        current = self.controller.output_index
        self._output_combo.setCurrentIndex(self._output_combo.findData(current))
        self._output_combo.blockSignals(False)
        self._apply_output()

    def _apply_output(self) -> None:
        if not self.isVisible() or not self._show_output.isChecked():
            self.display.hide()
            return
        screen = self.controller.output_screen
        if screen is None or screen not in self.controller.qscreens():
            self.display.float_over(self)
        else:
            self.display.cover(screen)
        self.display.show()
