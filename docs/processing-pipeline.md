# Пайплайн обработки

Основной процесс находится в `Processing(url, assetId, c)`.

## 1. Подготовка временных директорий

В debug-режиме используется `_tmp/<assetId>`, в обычном режиме - `os.TempDir()/assetId`.

Создаются подпапки:

```text
leads/
dzi/
dzi_bw/
channels/
channels_bw/
covers/
ranges/
```

## 2. Скачивание исходника

`downloadFileTemporary` скачивает файл по URL через `http.Get` во временный файл. Расширение берется из URL.

## 3. Конвертация презентаций

Если расширение входит в список `pptx`, `ppt`, `pptm`, `pps`, `pot`, файл конвертируется в PDF:

```bash
soffice --headless --convert-to pdf <file>
```

После этого обработка идет по PDF-ветке.

## 4. Определение типа файла

`vips.LoadImageFromFile` открывает файл и `OriginalFormat()` определяет загрузчик:

- `PDF` - обработка через `extractPDF`;
- остальные поддерживаемые изображения - через `extractImage`.

## 5. PDF-ветка

`renderPdf`:

1. Получает размеры страниц через `mutool pages`.
2. Уточняет реальные размеры через `mutool info -M`, при необходимости использует `pdfinfo`.
3. Пересчитывает DPI по `MaxSizePixels`, `MinResolution`, `MaxResolution`.
4. Получает spot-цвета через Ghostscript и `info.ps`.
5. Рендерит страницы:
   - `tiffsep`, если `SplitChannels=true`;
   - дополнительный `tiff32nc` для итогового color-render;
   - `png16m`, если `SplitChannels=false`.

`extractPDF`:

1. Открывает PDF через `go-poppler`.
2. Читает XMP metadata для размеров, plate names и swatches.
3. При `ExtractText=true` извлекает текст через `mutool draw -F stext.json`.
4. Собирает `pageInfo` и список `Swatch` для каждой страницы.

## 6. Image-ветка

`extractImage`:

1. Открывает файл через libvips.
2. Проверяет colorspace: RGB/RGB16/sRGB или CMYK.
3. Создает итоговый `Color` TIFF.
4. Если `SplitChannels=true`, делает `BandSplit`, инвертирует каналы и сохраняет отдельные TIFF:
   - CMYK: `Cyan`, `Magenta`, `Yellow`, `Black`, `Alpha`;
   - RGB: `Red`, `Green`, `Blue`, `Alpha`.

## 7. Colorize

`colorize` обрабатывает каждый swatch:

- копирует исходный канал в `channels_bw`;
- если `NeedMate=true`, создает цветную плашку из RGB swatch-цвета и композитит канал через `BlendModeScreen`;
- сохраняет цветной результат в `channels`;
- для итогового `Color`-канала mate не создается.

## 8. Генерация DZI

`makeDZI` вызывается дважды:

- для цветных каналов из `channels` в `dzi`;
- для черно-белых каналов из `channels_bw` в `dzi_bw`.

Для TIFF в цветной ветке перед DZI выполняется ICC-конвертация:

```bash
vips icc_transform <input.tiff> <output.jpeg>[Q=95] <ICC_PROFILE_PATH>
```

Если ICC-transform не сработал, используется fallback:

```bash
vips jpegsave <input.tiff> <output.jpeg>
```

DZI создается командой:

```bash
vips dzsave <file> <dziPath.zip> \
  --strip \
  --container=zip \
  --suffix ".<DZI_TILE_FORMAT><DZI_TILE_SETTING>" \
  --vips-concurrency=<MAX_CPU_COUNT> \
  --tile-size=<DZI_TILE_SIZE> \
  --overlap=<DZI_OVERLAP>
```

После генерации zip сканируется, и для каждого тайла сохраняются byte ranges: `offset` и `length`.

## 9. Covers и leads

`makeCovers` открывает цветный DZI zip, выбирает уровень тайлов с шириной не меньше 2000 пикселей или максимальный доступный уровень, склеивает PNG:

- `leads/<page>/<channel>.png` - крупный preview;
- `covers/<page>/<channel>.png` - thumbnail по `DZI_COVER_H`.

## 10. Manifest и выгрузка

`makeManifest` собирает `manifest.json`, пишет его во временную директорию, а обычный режим затем вызывает:

```bash
mc alias set <alias> <DZI_S3_HOST> <DZI_S3_KEY> <DZI_S3_SECRET>
mc cp -r <tmp>/ <alias>/<DZI_BUCKET>/<assetId> --quiet
mc alias rm <alias>
```

После успешного завершения временные файлы удаляются, если debug-режим выключен.
