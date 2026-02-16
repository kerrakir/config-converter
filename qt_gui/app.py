import shutil
import sys
from pathlib import Path

from PySide6.QtCore import QProcess, Qt
from PySide6.QtGui import QColor, QFont, QPainter, QPen, QPixmap
from PySide6.QtWidgets import (
    QApplication,
    QCheckBox,
    QComboBox,
    QFileDialog,
    QFormLayout,
    QFrame,
    QGridLayout,
    QGroupBox,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QSplitter,
    QTextEdit,
    QVBoxLayout,
    QWidget,
)


class MainWindow(QMainWindow):
    def __init__(self) -> None:
        super().__init__()
        self.setWindowTitle("Конвертер конфигураций")
        self.resize(980, 700)

        self.repo_root = Path(__file__).resolve().parent.parent
        self.is_frozen = getattr(sys, "frozen", False)
        self.bundle_dir = Path(getattr(sys, "_MEIPASS", self.repo_root))

        self.process = QProcess(self)
        self.process.setProcessChannelMode(QProcess.MergedChannels)
        self.process.readyReadStandardOutput.connect(self._read_process_output)
        self.process.finished.connect(self._process_finished)

        self.input_edit = QLineEdit()
        self.output_edit = QLineEdit()
        self.from_combo = QComboBox()
        self.to_combo = QComboBox()
        self.index_style_combo = QComboBox()
        self.index_prefix_edit = QLineEdit("1")
        self.log = QTextEdit()
        self.log.setReadOnly(True)
        self.log.setLineWrapMode(QTextEdit.NoWrap)
        self.source_preview = QTextEdit()
        self.source_preview.setReadOnly(True)
        self.source_preview.setLineWrapMode(QTextEdit.NoWrap)
        self.output_preview = QTextEdit()
        self.output_preview.setReadOnly(True)
        self.output_preview.setLineWrapMode(QTextEdit.NoWrap)

        self.run_btn = QPushButton("Конвертировать")
        self.stop_btn = QPushButton("Остановить")
        self.add_mapping_btn = QPushButton("Добавить правило")
        self.use_mapping_checkbox = QCheckBox("Использовать сопоставление типов интерфейсов")
        self.use_mapping_checkbox.setChecked(False)
        self.status_label = QLabel("Готово")
        self.run_btn.setObjectName("runButton")
        self.stop_btn.setObjectName("stopButton")

        self.interface_types = [
            "FastEthernet",
            "GigabitEthernet",
            "TenGigabitEthernet",
            "10GE",
            "GE",
            "Ethernet",
        ]
        self.mapping_rows: list[tuple[QWidget, QComboBox, QComboBox]] = []

        self._build_ui()
        self._apply_style()
        self._set_running_state(False)
        self._set_status("Готово", "idle")

    def _build_ui(self) -> None:
        self.from_combo.addItems(["cisco", "huawei", "json"])
        self.to_combo.addItems(["huawei", "cisco", "json"])
        self.index_style_combo.addItems(
            [
                "Оставить исходный",
                "2-частный (0/1)",
                "3-частный (1/0/1)",
            ]
        )

        self.input_edit.setPlaceholderText("Выберите исходный файл конфигурации")
        self.output_edit.setPlaceholderText("Укажите путь для готового файла")
        self.index_prefix_edit.setPlaceholderText("По умолчанию: 1")
        self.log.setPlaceholderText("Здесь будет вывод процесса")
        self.source_preview.setPlaceholderText("Тут появится исходная конфигурация")
        self.output_preview.setPlaceholderText("Тут появится готовая конфигурация после конвертации")

        browse_input_btn = QPushButton("Обзор")
        browse_output_btn = QPushButton("Обзор")

        browse_input_btn.clicked.connect(self._browse_input)
        browse_output_btn.clicked.connect(self._browse_output)
        self.run_btn.clicked.connect(self._run_conversion)
        self.stop_btn.clicked.connect(self._stop_conversion)
        self.add_mapping_btn.clicked.connect(self._add_mapping_row)
        self.use_mapping_checkbox.toggled.connect(self._toggle_mapping_controls)

        paths_box = QGroupBox("Файлы")
        paths_layout = QGridLayout()
        paths_layout.addWidget(QLabel("Исходный"), 0, 0)
        paths_layout.addWidget(self.input_edit, 0, 1)
        paths_layout.addWidget(browse_input_btn, 0, 2)
        paths_layout.addWidget(QLabel("Готовый"), 1, 0)
        paths_layout.addWidget(self.output_edit, 1, 1)
        paths_layout.addWidget(browse_output_btn, 1, 2)
        paths_box.setLayout(paths_layout)

        self.mapping_container = QWidget()
        self.mapping_layout = QVBoxLayout()
        self.mapping_layout.setContentsMargins(0, 0, 0, 0)
        self.mapping_layout.setSpacing(8)
        self.mapping_container.setLayout(self.mapping_layout)

        options_box = QGroupBox("Параметры конвертации")
        options_form = QFormLayout()
        options_form.setLabelAlignment(Qt.AlignLeft)
        options_form.addRow("Из формата", self.from_combo)
        options_form.addRow("В формат", self.to_combo)
        options_form.addRow("Формат индекса", self.index_style_combo)
        options_form.addRow("Префикс для 3-частного", self.index_prefix_edit)
        options_form.addRow("", self.use_mapping_checkbox)
        options_form.addRow(
            "",
            QLabel("Сопоставление нужно только для переименования типов интерфейсов. Обычно не требуется."),
        )
        options_form.addRow("Сопоставление", self.add_mapping_btn)
        options_form.addRow("", self.mapping_container)
        options_box.setLayout(options_form)
        self._add_mapping_row()
        self._toggle_mapping_controls(False)

        controls_layout = QHBoxLayout()
        controls_layout.addWidget(self.run_btn)
        controls_layout.addWidget(self.stop_btn)
        controls_layout.addStretch()
        controls_layout.addWidget(QLabel("Статус:"))
        controls_layout.addWidget(self.status_label)

        preview_box = QGroupBox("Предпросмотр конфигураций")
        preview_layout = QVBoxLayout()
        splitter = QSplitter(Qt.Horizontal)

        source_panel = QWidget()
        source_layout = QVBoxLayout()
        source_layout.setContentsMargins(0, 0, 0, 0)
        source_layout.addWidget(QLabel("Исходный файл"))
        source_layout.addWidget(self.source_preview)
        source_panel.setLayout(source_layout)

        output_panel = QWidget()
        output_layout = QVBoxLayout()
        output_layout.setContentsMargins(0, 0, 0, 0)
        output_layout.addWidget(QLabel("Готовый файл"))
        output_layout.addWidget(self.output_preview)
        output_panel.setLayout(output_layout)

        splitter.addWidget(source_panel)
        splitter.addWidget(output_panel)
        splitter.setSizes([480, 480])
        preview_layout.addWidget(splitter)
        preview_box.setLayout(preview_layout)

        log_box = QGroupBox("Журнал")
        log_layout = QVBoxLayout()
        log_layout.addWidget(self.log)
        log_box.setLayout(log_layout)

        content_layout = QVBoxLayout()
        content_layout.setContentsMargins(18, 18, 18, 18)
        content_layout.setSpacing(14)
        content_layout.addWidget(self._build_hero_card())
        content_layout.addWidget(paths_box)
        content_layout.addWidget(options_box)
        content_layout.addLayout(controls_layout)
        content_layout.addWidget(preview_box)
        content_layout.addWidget(log_box)

        root = QWidget()
        root.setLayout(content_layout)
        self.setCentralWidget(root)

    def _build_hero_card(self) -> QFrame:
        card = QFrame()
        card.setObjectName("heroCard")

        left_col = QVBoxLayout()
        title = QLabel("Конвертер сетевых конфигураций")
        title_font = QFont("Segoe UI", 15)
        title_font.setBold(True)
        title.setFont(title_font)

        subtitle = QLabel(
            "Быстро конвертируйте конфиги Cisco, Huawei и JSON с удобным предпросмотром исходного и готового файла."
        )
        subtitle.setWordWrap(True)

        steps = QLabel(
            "1) Выберите файлы   2) Укажите направление конвертации   3) Нажмите «Конвертировать»"
        )
        steps.setObjectName("heroSteps")

        left_col.addWidget(title)
        left_col.addWidget(subtitle)
        left_col.addWidget(steps)
        left_col.addStretch()

        image = QLabel()
        image.setPixmap(self._build_brand_pixmap())
        image.setAlignment(Qt.AlignCenter)

        layout = QHBoxLayout()
        layout.setContentsMargins(16, 16, 16, 16)
        layout.setSpacing(12)
        layout.addLayout(left_col, 3)
        layout.addWidget(image, 2)
        card.setLayout(layout)
        return card

    def _build_brand_pixmap(self) -> QPixmap:
        logo = self._load_logo_pixmap()
        if logo is not None:
            return logo
        return self._build_hero_pixmap()

    def _load_logo_pixmap(self) -> QPixmap | None:
        image_exts = {".png", ".jpg", ".jpeg", ".bmp", ".ico", ".webp"}
        candidates = [
            self.repo_root / "Логотип_СарФТИ_НИЯУ_МИФИ.jpg",
            self.bundle_dir / "Логотип_СарФТИ_НИЯУ_МИФИ.jpg",
        ]

        for folder in (self.repo_root, self.bundle_dir):
            for path in folder.iterdir():
                if not path.is_file() or path.suffix.lower() not in image_exts:
                    continue
                lower_name = path.name.lower()
                if "logo" in lower_name or "логотип" in lower_name:
                    candidates.append(path)

        for path in candidates:
            if not path.exists():
                continue
            pixmap = QPixmap(str(path))
            if pixmap.isNull():
                continue
            return pixmap.scaled(260, 120, Qt.KeepAspectRatio, Qt.SmoothTransformation)

        return None

    def _build_hero_pixmap(self) -> QPixmap:
        pixmap = QPixmap(260, 120)
        pixmap.fill(Qt.transparent)

        painter = QPainter(pixmap)
        painter.setRenderHint(QPainter.Antialiasing, True)

        painter.setBrush(QColor("#eaf4ff"))
        painter.setPen(Qt.NoPen)
        painter.drawRoundedRect(0, 0, 260, 120, 14, 14)

        pen = QPen(QColor("#2f80ed"), 2)
        painter.setPen(pen)
        painter.drawLine(30, 30, 110, 30)
        painter.drawLine(30, 60, 110, 60)
        painter.drawLine(30, 90, 110, 90)

        painter.setBrush(QColor("#2f80ed"))
        painter.setPen(Qt.NoPen)
        painter.drawEllipse(118, 24, 12, 12)
        painter.drawEllipse(118, 54, 12, 12)
        painter.drawEllipse(118, 84, 12, 12)

        painter.setPen(QPen(QColor("#2f80ed"), 2))
        painter.drawLine(140, 30, 226, 30)
        painter.drawLine(140, 60, 226, 60)
        painter.drawLine(140, 90, 226, 90)

        painter.setPen(Qt.NoPen)
        painter.setBrush(QColor("#1f5fbf"))
        painter.drawRoundedRect(226, 24, 14, 14, 5, 5)
        painter.drawRoundedRect(226, 54, 14, 14, 5, 5)
        painter.drawRoundedRect(226, 84, 14, 14, 5, 5)

        painter.end()
        return pixmap

    def _apply_style(self) -> None:
        self.setStyleSheet(
            """
            QMainWindow {
                background: #f4f7fb;
            }
            QLabel {
                color: #1f2937;
            }
            QGroupBox {
                font-weight: 600;
                border: 1px solid #d4dde9;
                border-radius: 10px;
                margin-top: 14px;
                padding: 10px;
                background: #ffffff;
            }
            QGroupBox::title {
                subcontrol-origin: margin;
                left: 12px;
                padding: 0 6px;
                color: #334155;
            }
            QLineEdit, QComboBox, QTextEdit {
                border: 1px solid #c9d5e5;
                border-radius: 8px;
                padding: 6px 8px;
                background: #fbfdff;
            }
            QLineEdit:focus, QComboBox:focus, QTextEdit:focus {
                border: 1px solid #2f80ed;
                background: #ffffff;
            }
            QPushButton {
                border: none;
                border-radius: 8px;
                padding: 8px 12px;
                background: #dbe7f6;
                color: #1f2937;
                font-weight: 600;
            }
            QPushButton:hover {
                background: #cadcf3;
            }
            QPushButton:disabled {
                background: #e4eaf2;
                color: #9aa6b2;
            }
            #runButton {
                background: #2f80ed;
                color: #ffffff;
            }
            #runButton:hover {
                background: #2468c2;
            }
            #stopButton {
                background: #ef4444;
                color: #ffffff;
            }
            #stopButton:hover {
                background: #dc2626;
            }
            #heroCard {
                border: 1px solid #d4dde9;
                border-radius: 12px;
                background: qlineargradient(
                    x1:0, y1:0, x2:1, y2:1,
                    stop:0 #ffffff,
                    stop:1 #eef4fc
                );
            }
            #heroSteps {
                color: #2f80ed;
                font-weight: 600;
            }
            """
        )

    def _browse_input(self) -> None:
        path, _ = QFileDialog.getOpenFileName(self, "Выберите исходный конфиг")
        if path:
            self.input_edit.setText(path)
            self._load_preview(path, self.source_preview, "Не удалось прочитать исходный файл.")

    def _browse_output(self) -> None:
        path, _ = QFileDialog.getSaveFileName(self, "Выберите готовый файл")
        if path:
            self.output_edit.setText(path)
            if Path(path).exists():
                self._load_preview(path, self.output_preview, "Не удалось прочитать готовый файл.")
            else:
                self.output_preview.clear()
                self.output_preview.setPlaceholderText(
                    "Тут появится готовая конфигурация после конвертации"
                )

    def _add_mapping_row(self) -> None:
        row_widget = QWidget()
        row_layout = QHBoxLayout()
        row_layout.setContentsMargins(0, 0, 0, 0)

        from_iface = QComboBox()
        to_iface = QComboBox()
        remove_btn = QPushButton("Удалить")

        from_iface.addItem("-")
        from_iface.addItems(self.interface_types)
        to_iface.addItem("-")
        to_iface.addItems(self.interface_types)

        remove_btn.clicked.connect(lambda: self._remove_mapping_row(row_widget))

        row_layout.addWidget(QLabel("Из"))
        row_layout.addWidget(from_iface)
        row_layout.addWidget(QLabel("В"))
        row_layout.addWidget(to_iface)
        row_layout.addWidget(remove_btn)
        row_widget.setLayout(row_layout)

        self.mapping_rows.append((row_widget, from_iface, to_iface))
        self.mapping_layout.addWidget(row_widget)

    def _remove_mapping_row(self, row_widget: QWidget) -> None:
        if len(self.mapping_rows) == 1:
            QMessageBox.warning(self, "Ошибка", "Должно остаться хотя бы одно правило.")
            return

        for i, (widget, _, _) in enumerate(self.mapping_rows):
            if widget is row_widget:
                self.mapping_rows.pop(i)
                self.mapping_layout.removeWidget(widget)
                widget.deleteLater()
                break

    def _build_if_map_value(self) -> str:
        if not self.use_mapping_checkbox.isChecked():
            return ""
        pairs: list[str] = []
        for _, src_combo, dst_combo in self.mapping_rows:
            src = src_combo.currentText()
            dst = dst_combo.currentText()
            if src == "-" or dst == "-":
                continue
            if src == dst:
                continue
            pairs.append(f"{src}={dst}")
        return ",".join(pairs)

    def _run_conversion(self) -> None:
        if self.process.state() != QProcess.NotRunning:
            QMessageBox.warning(self, "Занято", "Конвертация уже выполняется.")
            return

        input_path = self.input_edit.text().strip()
        output_path = self.output_edit.text().strip()
        if not input_path or not output_path:
            QMessageBox.warning(
                self, "Ошибка", "Нужно указать исходный и готовый файлы."
            )
            return
        self._load_preview(input_path, self.source_preview, "Не удалось прочитать исходный файл.")

        try:
            program, arguments, cwd = self._build_command(
                input_path=input_path,
                output_path=output_path,
                source=self.from_combo.currentText(),
                target=self.to_combo.currentText(),
                if_map=self._build_if_map_value(),
                if_index=self._selected_index_style(),
                if_index_prefix=self.index_prefix_edit.text().strip() or "1",
            )
        except RuntimeError as exc:
            QMessageBox.critical(self, "Ошибка запуска", str(exc))
            self._set_status(str(exc), "error")
            return

        self.log.clear()
        self._append_log(f"$ {program} {' '.join(arguments)}")
        self._set_running_state(True)
        self._set_status("Идет конвертация...", "running")

        self.process.setWorkingDirectory(cwd)
        self.process.start(program, arguments)
        if not self.process.waitForStarted(3000):
            self._set_running_state(False)
            self._set_status("Не удалось запустить процесс", "error")
            QMessageBox.critical(self, "Ошибка запуска", "Не удалось запустить конвертер.")

    def _stop_conversion(self) -> None:
        if self.process.state() == QProcess.NotRunning:
            return
        self.process.kill()
        self._append_log("Процесс остановлен пользователем.")
        self._set_running_state(False)
        self._set_status("Остановлено", "error")

    def _build_command(
        self,
        input_path: str,
        output_path: str,
        source: str,
        target: str,
        if_map: str,
        if_index: str,
        if_index_prefix: str,
    ) -> tuple[str, list[str], str]:
        exe = self.bundle_dir / "converter.exe"
        if not exe.exists():
            exe = self.repo_root / "converter.exe"
        if exe.exists():
            program = str(exe)
            args = [
                "-in",
                input_path,
                "-out",
                output_path,
                "-from",
                source,
                "-to",
                target,
            ]
            if if_map:
                args.extend(["-if-map", if_map])
            if if_index != "keep":
                args.extend(["-if-index", if_index])
                if if_index == "3":
                    args.extend(["-if-index-prefix", if_index_prefix])
            return program, args, str(self.repo_root)

        if self.is_frozen:
            raise RuntimeError("В собранном приложении не найден converter.exe.")

        go_bin = shutil.which("go")
        if go_bin:
            args = [
                "run",
                "./cmd/converter",
                "-in",
                input_path,
                "-out",
                output_path,
                "-from",
                source,
                "-to",
                target,
            ]
            if if_map:
                args.extend(["-if-map", if_map])
            if if_index != "keep":
                args.extend(["-if-index", if_index])
                if if_index == "3":
                    args.extend(["-if-index-prefix", if_index_prefix])
            return go_bin, args, str(self.repo_root)

        raise RuntimeError("Не найден converter.exe и не установлен Go toolchain.")

    def _read_process_output(self) -> None:
        data = self.process.readAllStandardOutput().data().decode("utf-8", errors="replace")
        if data:
            self._append_log(data.rstrip("\n"))

    def _process_finished(self, exit_code: int, _exit_status: QProcess.ExitStatus) -> None:
        self._set_running_state(False)
        if exit_code == 0:
            self._append_log("Конвертация успешно завершена.")
            self._set_status("Готово", "ok")
            output_path = self.output_edit.text().strip()
            if output_path and Path(output_path).exists():
                self._load_preview(output_path, self.output_preview, "Не удалось прочитать готовый файл.")
        else:
            self._append_log(f"Конвертация завершилась с ошибкой, код: {exit_code}")
            self._set_status(f"Ошибка (код {exit_code})", "error")

    def _append_log(self, text: str) -> None:
        self.log.append(text)

    def _selected_index_style(self) -> str:
        text = self.index_style_combo.currentText()
        if text.startswith("2-частный"):
            return "2"
        if text.startswith("3-частный"):
            return "3"
        return "keep"

    def _set_running_state(self, running: bool) -> None:
        self.run_btn.setDisabled(running)
        self.stop_btn.setDisabled(not running)

    def _set_status(self, text: str, status: str) -> None:
        colors = {
            "ok": "#166534",
            "error": "#b91c1c",
            "running": "#1d4ed8",
            "idle": "#334155",
        }
        color = colors.get(status, colors["idle"])
        self.status_label.setText(text)
        self.status_label.setStyleSheet(f"font-weight: 700; color: {color};")

    def _toggle_mapping_controls(self, enabled: bool) -> None:
        self.add_mapping_btn.setEnabled(enabled)
        self.mapping_container.setEnabled(enabled)

    def _load_preview(self, file_path: str, editor: QTextEdit, error_text: str) -> None:
        path = Path(file_path)
        if not path.exists():
            editor.clear()
            return
        text = self._read_text_fallback(path)
        if text is None:
            editor.clear()
            editor.setPlaceholderText(error_text)
            return
        editor.setPlainText(text)

    def _read_text_fallback(self, path: Path) -> str | None:
        for encoding in ("utf-8", "cp1251", "latin-1"):
            try:
                return path.read_text(encoding=encoding)
            except UnicodeDecodeError:
                continue
            except OSError:
                return None
        return None


def main() -> int:
    app = QApplication(sys.argv)
    window = MainWindow()
    window.show()
    return app.exec()


if __name__ == "__main__":
    sys.exit(main())
