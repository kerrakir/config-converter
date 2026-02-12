# Converter Project

Конвертер сетевых конфигураций между Cisco, Huawei и JSON.

## Структура проекта

- `cmd/converter` — CLI точка входа (`main.go`) и CLI-утилиты.
- `parser` — парсеры исходных конфигов.
- `generator` — генераторы целевых конфигов.
- `model` — общая модель конфигурации.
- `qt_gui` — GUI на Qt (PySide6).
- `examples` — примеры входных/выходных конфигов.
- `examples/outputs` — временные результаты конвертаций.

## CLI запуск

```bash
go run ./cmd/converter -in examples/cisco_sample.txt -out examples/huw.txt -from cisco -to huawei
```

Сборка бинарника:

```bash
go build -o converter.exe ./cmd/converter
```

## GUI запуск

```bash
python qt_gui/app.py
```
