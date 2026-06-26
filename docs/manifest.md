# Формат manifest.json

`manifest.json` описывает результат обработки и используется как индекс для страниц, каналов, preview и DZI-тайлов.

## Верхний уровень

| Поле | Тип | Описание |
| --- | --- | --- |
| `version` | string | Текущая версия manifest, в коде фиксирована как `"5"`. |
| `id` | string | `assetId`, переданный в CLI/`Processing`. |
| `timestamp_start` | string | Время начала обработки в формате `YYYY-MM-DD HH:mm:ss`. |
| `timestamp_end` | string | Время завершения сборки manifest. |
| `source` | string | URL исходного файла. |
| `filename` | string | Имя файла из URL. |
| `basename` | string | UUID, используемый как базовое имя промежуточных файлов. |
| `tile_size` | string | Размер тайла. |
| `tile_format` | string | Формат тайлов. |
| `cover_height` | string | Размер cover preview. |
| `overlap` | string | DZI overlap. |
| `mode` | string | Сейчас фиксирован как `"Perpage"`. |
| `pages` | array | Список страниц. |
| `swatches` | array | Уникальный список swatches по всему документу. |
| `split_channels` | bool | Было ли включено разделение каналов. |
| `overprint` | string | Использованный режим overprint. |

## Page

| Поле | Тип | Описание |
| --- | --- | --- |
| `page_num` | int | Номер страницы, начиная с 1. |
| `mode` | string | Цветовой режим страницы: `CMYK` или `RBG` в текущем коде. |
| `size` | object | Размер и DPI страницы. |
| `text_content` | string | JSON-строка из `mutool` для PDF, если `DZI_EXTRACT_TEXT=true`. |
| `channels_v4` | array | Подробное описание каналов. |
| `channels` | array | Список имен каналов. |

## Size

| Поле | Тип | Описание |
| --- | --- | --- |
| `width` | string | Ширина страницы. Для изображений - пиксели, для PDF - миллиметры. |
| `height` | string | Высота страницы. |
| `units` | string | `px`, `mm` или единица из PDF metadata. |
| `dpi` | string | Итоговый DPI страницы. |

## ChannelV4

| Поле | Тип | Описание |
| --- | --- | --- |
| `name` | string | Имя канала: `Color`, `Cyan`, spot-name и т.д. |
| `dzi_color_path` | string | Относительный путь к цветному DZI zip. |
| `dzi_bw_path` | string | Относительный путь к черно-белому DZI zip. |
| `lead_path` | string | Относительный путь к lead PNG. |
| `cover_path` | string | Относительный путь к cover PNG. |
| `color_ranges` | object | Byte ranges тайлов внутри цветного zip. |
| `bw_ranges_path` | string | Относительный путь к JSON с byte ranges для черно-белого zip. |

## ZipRange

`color_ranges` и bw ranges содержат map:

```json
{
  "12/0_0.png": {
    "offset": 12345,
    "length": 6789
  }
}
```

- `offset` - позиция сжатых данных файла внутри zip.
- `length` - `CompressedSize64` файла внутри zip.

`.dzi`/`.xml` записи из zip не попадают в ranges.

## Swatch

| Поле | Тип | Описание |
| --- | --- | --- |
| `name` | string | Имя swatch/канала. |
| `rgb` | string | Цвет в hex-формате. В структуре поле называется `RBG`, но JSON-ключ - `rgb`. |
| `type` | string | `CmykComponent`, `SpotComponent` или `Final`. |
| `need_mate` | bool | Нужно ли создавать цветную mate-версию канала. |

## Пример структуры

```json
{
  "version": "5",
  "id": "100500",
  "source": "https://example.com/source.pdf",
  "filename": "source.pdf",
  "tile_size": "1024",
  "tile_format": "png",
  "pages": [
    {
      "page_num": 1,
      "mode": "CMYK",
      "size": {
        "width": "210.000000",
        "height": "297.000000",
        "units": "mm",
        "dpi": "600"
      },
      "channels": ["Color", "Cyan", "Magenta", "Yellow", "Black"],
      "channels_v4": []
    }
  ],
  "split_channels": true,
  "overprint": "/enable"
}
```
