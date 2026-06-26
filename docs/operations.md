# Эксплуатация и диагностика

## Проверка окружения

Перед запуском на новом сервере проверьте доступность внешних команд:

```bash
go version
vips --version
gs --version
mutool --version
pdfinfo -v
mc --version
soffice --version
```

Для сборки Go-кода:

```bash
go test ./...
go build ./...
```

В проекте нет явных unit-тестов, поэтому `go test ./...` в основном проверяет компиляцию пакетов.

## Debug-режим

Для разбора проблем используйте:

```bash
DEBUG=1 go run ./cmd "<url>" <assetId>
```

После падения или успешного завершения смотрите:

- `_tmp/<assetId>/channels` - цветные промежуточные каналы;
- `_tmp/<assetId>/channels_bw` - черно-белые каналы;
- `_tmp/<assetId>/dzi` - цветные DZI zip;
- `_tmp/<assetId>/dzi_bw` - черно-белые DZI zip;
- `_tmp/<assetId>/leads` и `_tmp/<assetId>/covers` - preview;
- `_tmp/<assetId>/manifest.json` - итоговый индекс.

## Частые проблемы

### `overprint not correct`

`DZI_OVERPRINT` должен быть одним из:

- `/enable`
- `/simulate`
- `/disable`

### Ошибка `unsupported color space`

Image-ветка поддерживает только RGB/RGB16/sRGB и CMYK. Файл с другим colorspace нужно предварительно нормализовать.

### Ошибки `mutool`, `gs`, `vips`, `mc`

Все эти инструменты вызываются через `exec.Command`. Если бинарь не найден или завершился с ненулевым кодом, ошибка вернет combined output команды. Проверьте `PATH` процесса и версии системных пакетов.

### Не загрузилось в S3

Production-выгрузка идет через `mc`:

```bash
mc alias set mediaquad<assetId> <host> <key> <secret>
mc cp -r <tmp>/ mediaquad<assetId>/<bucket>/<assetId> --quiet
mc alias rm mediaquad<assetId>
```

Проверьте endpoint, credentials, bucket и права на запись.

### Презентация не конвертируется

Проверьте `SOFFICE_PATH`. По умолчанию используется `soffice`; если LibreOffice установлен в нестандартное место, задайте полный путь.

## Производительность

- `MAX_CPU_COUNT` управляет параллелизмом worker pool, `vips.Startup` и `vips dzsave`.
- Большие PDF ограничиваются через `MAX_SIZE_PIXELS`, `DZI_MIN_RESOLUTION`, `DZI_MAX_RESOLUTION`.
- При `DZI_SPLIT_CHANNELS=true` объем работы и размер артефактов растут пропорционально числу каналов и spot-цветов.
- При `DZI_COPY_CHANNELS=false` промежуточные `channels` и `channels_bw` удаляются перед upload, что уменьшает итоговый объем.

## Замечания по коду

- Цветовой режим RGB в JSON сейчас пишется как `RBG`, потому что константа в коде называется `ColorModeRBG`.
- `DefaultFolderPerm` равен `0777`; это влияет на создаваемые директории.
- `colorize` использует worker pool с одним worker, несмотря на наличие `MaxCpuCount`.
- В PDF-ветке `renderPdf` временно меняет `c.Overprint` при дополнительном `tiff32nc` render; это общее состояние конфига.
