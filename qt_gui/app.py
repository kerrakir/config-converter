import shutil
import sys
from pathlib import Path

from PySide6.QtCore import QProcess
from PySide6.QtWidgets import (
    QApplication,
    QComboBox,
    QFileDialog,
    QFormLayout,
    QGridLayout,
    QGroupBox,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QTextEdit,
    QVBoxLayout,
    QWidget,
)


class MainWindow(QMainWindow):
    def __init__(self) -> None:
        super().__init__()
        self.setWindowTitle("Config Converter (Qt)")
        self.resize(900, 620)

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

        self.interface_types = [
            "FastEthernet",
            "GigabitEthernet",
            "TenGigabitEthernet",
            "10GE",
            "GE",
            "Ethernet",
        ]
        self.mapping_rows: list[tuple[QWidget, QComboBox, QComboBox]] = []

        self.from_combo.addItems(["cisco", "huawei", "json"])
        self.to_combo.addItems(["huawei", "cisco", "json"])
        self.index_style_combo.addItems(
            [
                "Keep original",
                "2-part (0/1)",
                "3-part (1/0/1)",
            ]
        )

        browse_input_btn = QPushButton("Browse...")
        browse_output_btn = QPushButton("Browse...")
        run_btn = QPushButton("Convert")
        stop_btn = QPushButton("Stop")
        add_mapping_btn = QPushButton("Add mapping")

        browse_input_btn.clicked.connect(self._browse_input)
        browse_output_btn.clicked.connect(self._browse_output)
        run_btn.clicked.connect(self._run_conversion)
        stop_btn.clicked.connect(self._stop_conversion)
        add_mapping_btn.clicked.connect(self._add_mapping_row)

        paths_box = QGroupBox("Files")
        paths_layout = QGridLayout()
        paths_layout.addWidget(QLabel("Input file"), 0, 0)
        paths_layout.addWidget(self.input_edit, 0, 1)
        paths_layout.addWidget(browse_input_btn, 0, 2)
        paths_layout.addWidget(QLabel("Output file"), 1, 0)
        paths_layout.addWidget(self.output_edit, 1, 1)
        paths_layout.addWidget(browse_output_btn, 1, 2)
        paths_box.setLayout(paths_layout)

        self.mapping_container = QWidget()
        self.mapping_layout = QVBoxLayout()
        self.mapping_layout.setContentsMargins(0, 0, 0, 0)
        self.mapping_layout.setSpacing(6)
        self.mapping_container.setLayout(self.mapping_layout)

        options_box = QGroupBox("Options")
        options_form = QFormLayout()
        options_form.addRow("From", self.from_combo)
        options_form.addRow("To", self.to_combo)
        options_form.addRow("Index format", self.index_style_combo)
        options_form.addRow("3-part prefix", self.index_prefix_edit)
        options_form.addRow("Interface map", add_mapping_btn)
        options_form.addRow("", QLabel("Example: FastEthernet -> GigabitEthernet"))
        options_form.addRow("", self.mapping_container)
        options_box.setLayout(options_form)
        self._add_mapping_row()

        controls_layout = QHBoxLayout()
        controls_layout.addWidget(run_btn)
        controls_layout.addWidget(stop_btn)
        controls_layout.addStretch()

        root = QWidget()
        root_layout = QVBoxLayout()
        root_layout.addWidget(paths_box)
        root_layout.addWidget(options_box)
        root_layout.addLayout(controls_layout)
        root_layout.addWidget(QLabel("Log"))
        root_layout.addWidget(self.log)
        root.setLayout(root_layout)
        self.setCentralWidget(root)

    def _browse_input(self) -> None:
        path, _ = QFileDialog.getOpenFileName(self, "Select input config")
        if path:
            self.input_edit.setText(path)

    def _browse_output(self) -> None:
        path, _ = QFileDialog.getSaveFileName(self, "Select output file")
        if path:
            self.output_edit.setText(path)

    def _add_mapping_row(self) -> None:
        row_widget = QWidget()
        row_layout = QHBoxLayout()
        row_layout.setContentsMargins(0, 0, 0, 0)

        from_iface = QComboBox()
        to_iface = QComboBox()
        remove_btn = QPushButton("Remove")

        from_iface.addItem("-")
        from_iface.addItems(self.interface_types)
        to_iface.addItem("-")
        to_iface.addItems(self.interface_types)

        remove_btn.clicked.connect(lambda: self._remove_mapping_row(row_widget))

        row_layout.addWidget(QLabel("From"))
        row_layout.addWidget(from_iface)
        row_layout.addWidget(QLabel("To"))
        row_layout.addWidget(to_iface)
        row_layout.addWidget(remove_btn)
        row_widget.setLayout(row_layout)

        self.mapping_rows.append((row_widget, from_iface, to_iface))
        self.mapping_layout.addWidget(row_widget)

    def _remove_mapping_row(self, row_widget: QWidget) -> None:
        if len(self.mapping_rows) == 1:
            QMessageBox.warning(self, "Validation error", "At least one mapping row must remain.")
            return

        for i, (widget, _, _) in enumerate(self.mapping_rows):
            if widget is row_widget:
                self.mapping_rows.pop(i)
                self.mapping_layout.removeWidget(widget)
                widget.deleteLater()
                break

    def _build_if_map_value(self) -> str:
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
            QMessageBox.warning(self, "Busy", "Conversion is already running.")
            return

        input_path = self.input_edit.text().strip()
        output_path = self.output_edit.text().strip()
        if not input_path or not output_path:
            QMessageBox.warning(
                self, "Validation error", "Input and output files are required."
            )
            return

        program, arguments, cwd = self._build_command(
            input_path=input_path,
            output_path=output_path,
            source=self.from_combo.currentText(),
            target=self.to_combo.currentText(),
            if_map=self._build_if_map_value(),
            if_index=self._selected_index_style(),
            if_index_prefix=self.index_prefix_edit.text().strip() or "1",
        )

        self.log.clear()
        self._append_log(f"$ {program} {' '.join(arguments)}")
        self.process.setWorkingDirectory(cwd)
        self.process.start(program, arguments)
        if not self.process.waitForStarted(3000):
            QMessageBox.critical(self, "Start error", "Failed to start converter process.")

    def _stop_conversion(self) -> None:
        if self.process.state() == QProcess.NotRunning:
            return
        self.process.kill()
        self._append_log("Process stopped by user.")

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
            raise RuntimeError("Bundled converter.exe not found in packaged application.")

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

        raise RuntimeError("converter.exe and Go toolchain are both unavailable.")

    def _read_process_output(self) -> None:
        data = self.process.readAllStandardOutput().data().decode("utf-8", errors="replace")
        if data:
            self._append_log(data.rstrip("\n"))

    def _process_finished(self, exit_code: int, _exit_status: QProcess.ExitStatus) -> None:
        if exit_code == 0:
            self._append_log("Conversion finished successfully.")
        else:
            self._append_log(f"Conversion failed with exit code: {exit_code}")

    def _append_log(self, text: str) -> None:
        self.log.append(text)

    def _selected_index_style(self) -> str:
        text = self.index_style_combo.currentText()
        if text.startswith("2-part"):
            return "2"
        if text.startswith("3-part"):
            return "3"
        return "keep"


def main() -> int:
    app = QApplication(sys.argv)
    window = MainWindow()
    window.show()
    return app.exec()


if __name__ == "__main__":
    sys.exit(main())
