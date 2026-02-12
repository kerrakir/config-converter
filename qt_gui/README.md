# Qt GUI

This folder contains a Qt desktop interface for the converter.

## Run

1. Install dependencies:

```bash
pip install -r qt_gui/requirements.txt
```

2. Start the GUI from repository root:

```bash
python qt_gui/app.py
```

## Notes

- The GUI calls `converter.exe` if it exists in the repo root.
- If `converter.exe` is missing, the GUI falls back to `go run ./cmd/converter`.
- Interface type mapping is supported through the same `-if-map` syntax used by CLI.
- Interface index format is supported: keep original, 2-part (`0/1`), or 3-part (`1/0/1`).

## Build Single EXE (Windows)

Build backend first:

```bash
go build -o converter.exe ./cmd/converter
```

Install PyInstaller:

```bash
pip install pyinstaller
```

Build one-file GUI EXE that includes `converter.exe`:

```bash
pyinstaller --noconfirm --onefile --windowed --name converter-gui --add-binary "converter.exe;." qt_gui/app.py
```

Output EXE:

- `dist/converter-gui.exe`
