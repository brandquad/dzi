# Конфигурация

CLI читает настройки через `github.com/kelseyhightower/envconfig`. Внутренняя структура `cmd.Config` преобразуется в `dzi.Config`.

## Переменные окружения

| Переменная | Обязательная | Значение по умолчанию | Описание |
| --- | --- | --- | --- |
| `DZI_S3_HOST` | да | - | S3-compatible endpoint для `mc alias set`. |
| `DZI_S3_KEY` | да | - | Access key. |
| `DZI_S3_SECRET` | да | - | Secret key. |
| `DZI_BUCKET` | да | `dzi` | Bucket для загрузки артефактов. |
| `DZI_TILE_SIZE` | нет | `1024` | Размер тайла для `vips dzsave`. |
| `DZI_OVERLAP` | нет | `1` | Overlap тайлов DZI. |
| `DZI_RESOLUTION` | нет | `600` | Базовый DPI для PDF. |
| `DZI_MIN_RESOLUTION` | нет | `200` | Нижняя граница DPI после перерасчета. |
| `DZI_MAX_RESOLUTION` | нет | `1600` | Верхняя граница DPI после перерасчета. |
| `DZI_COVER_H` | нет | `300` | Размер cover preview. |
| `DZI_DEBUG` | нет | `false` | Debug-режим в envconfig. |
| `DEBUG` | нет | - | Если переменная существует, `config.DebugMode` становится `true`. |
| `DZI_SPLIT_CHANNELS` | нет | `true` | Разделять файл на цветовые/spot-каналы. |
| `DZI_OVERPRINT` | нет | `/enable` | Режим overprint для Ghostscript. |
| `HOOK_URL` | нет | - | Присутствует в CLI-конфиге, в текущем коде не используется. |
| `DZI_COPY_CHANNELS` | нет | `false` | Оставлять `channels` и `channels_bw` в итоговой выгрузке. |
| `MAX_CPU_COUNT` | нет | `4` | Параллелизм worker pool и `vips` concurrency. |
| `MAX_SIZE_PIXELS` | нет | `15000` | Целевая/предельная сторона страницы в пикселях при расчете DPI. |
| `DZI_EXTRACT_TEXT` | нет | `true` | Извлекать текст PDF через `mutool draw -F stext.json`. |
| `DZI_TILE_FORMAT` | нет | `png` | Формат тайлов: используется в suffix `vips dzsave`. |
| `DZI_TILE_SETTING` | нет | пусто | Дополнительная часть suffix, например параметры качества. |
| `ICC_PROFILE_PATH` | нет | `./icc/sRGB_Profile.icc` | ICC-профиль для `vips icc_transform`. |
| `GRAPHICS_ALPHA_BITS` | нет | `4` | Значение `-dGraphicsAlphaBits` для Ghostscript. |
| `DZI_USE_PDFX3` | нет | `false` | Управляет `-dUsePDFX3Profile`. |
| `SOFFICE_PATH` | нет | `soffice` | Путь к LibreOffice CLI. |

## Допустимые overprint-режимы

`DZI_OVERPRINT` валидируется при старте. Допустимые значения:

- `/enable`
- `/simulate`
- `/disable`

При другом значении CLI завершится с ошибкой `overprint not correct`.

## Особенности настроек

- Для презентаций после конвертации в PDF код принудительно выставляет `MaxSizePixels = 5000`, `MaxResolution = 600`, `SplitChannels = false`.
- `MaxCpuCount` используется одновременно для worker pool и `vips.Startup`.
- `TileSize`, `Overlap`, `CoverHeight` хранятся строками, потому что напрямую передаются в CLI-команды и manifest.
- Если `CopyChannelsToS3=false`, папки `channels` и `channels_bw` удаляются перед формированием итоговой S3-выгрузки.
