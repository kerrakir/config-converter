$ErrorActionPreference = "Stop"

Write-Host "Building converter backend..."
go build -o converter.exe ./cmd/converter

Write-Host "Building one-file Qt GUI..."
pyinstaller --noconfirm --onefile --windowed --name converter-gui --add-binary "converter.exe;." qt_gui/app.py

Write-Host "Done. Output: dist\\converter-gui.exe"
