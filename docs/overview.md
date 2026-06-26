# Обзор проекта

`dzi` принимает исходный файл по URL и строит набор артефактов для просмотра больших макетов через Deep Zoom:

- zip-архивы DZI по страницам и каналам;
- цветные и черно-белые версии каналов;
- lead/cover preview PNG;
- `manifest.json` с метаданными страниц, каналов, размеров, тайлов и byte ranges.

Основная публичная точка входа библиотеки:

```go
manifest, err := dzi.Processing(url, assetId, config)
```

CLI находится в `cmd/main.go` и преобразует переменные окружения в `dzi.Config`, затем вызывает `dzi.Processing`.

## Поддерживаемые входные файлы

- PDF: основной сценарий, включая многостраничные документы и spot-каналы.
- Растровые изображения: RGB/RGB16/sRGB и CMYK.
- Презентации: `pptx`, `ppt`, `pptm`, `pps`, `pot`; перед обработкой конвертируются в PDF через LibreOffice.

## Основные модули

- `processing.go` - оркестрация всего процесса.
- `extract_pdf.go`, `render_pdf.go` - анализ PDF, расчет DPI, рендер страниц и каналов через MuPDF/Ghostscript.
- `extract_image.go` - обработка одиночных изображений.
- `colorize.go` - создание цветных и черно-белых каналов.
- `make_dzi.go` - генерация DZI zip-архивов через `vips dzsave`.
- `make_covers.go` - сборка lead/cover preview из DZI-тайлов.
- `make_manifest.go`, `manifest.go` - структура и сериализация `manifest.json`.
- `utils.go` - скачивание файла, S3-синхронизация, вызов внешних команд, цветовые утилиты.
- `text_processor.go` - отдельный extractor текстовых блоков через `mutool`.

## Внешние инструменты

Проект зависит не только от Go-модулей, но и от системных бинарей:

- `vips` / `libvips` - чтение изображений, ICC transform, DZI, PNG/TIFF/JPEG export.
- `gs` / Ghostscript - рендер PDF и извлечение separations.
- `mutool` / MuPDF - размеры страниц и текст.
- `pdfinfo` - fallback для размеров PDF.
- `mc` / MinIO Client - загрузка результатов в S3-совместимое хранилище.
- `soffice` / LibreOffice - конвертация презентаций в PDF.
