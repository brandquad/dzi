# Установка и запуск

## Требования

- Go `1.23.2` или совместимая версия.
- `libvips` и CLI `vips`.
- Ghostscript (`gs`).
- MuPDF tools (`mutool`).
- Poppler tools (`pdfinfo`).
- MinIO Client (`mc`) для production-загрузки в S3.
- LibreOffice (`soffice`) для обработки презентаций.

На macOS зависимости обычно ставятся через Homebrew:

```bash
brew install go vips ghostscript mupdf poppler minio/stable/mc libreoffice
```

## Сборка

```bash
go mod download
go build ./...
go build -o dzi ./cmd
```

## Запуск CLI

CLI ожидает два позиционных аргумента:

1. URL исходного файла.
2. Числовой `assetId`.

```bash
./dzi "https://example.com/source.pdf" 100500
```

Перед запуском должны быть заданы обязательные S3-переменные:

```bash
export DZI_S3_HOST="https://s3.example.com"
export DZI_S3_KEY="access-key"
export DZI_S3_SECRET="secret-key"
export DZI_BUCKET="dzi"
```

## Локальная отладка

В debug-режиме результат не отправляется в S3 и временная директория не удаляется:

```bash
DEBUG=1 go run ./cmd "https://example.com/source.pdf" 100500
```

Артефакты будут лежать в:

```text
_tmp/<assetId>/
  channels/
  channels_bw/
  covers/
  dzi/
  dzi_bw/
  leads/
  ranges/
  manifest.json
```

`DZI_DEBUG=true` также включает debug-настройку в конфиге, но код дополнительно проверяет именно наличие переменной `DEBUG` и перезаписывает `config.DebugMode`.

## Использование как библиотеки

```go
package main

import "github.com/brandquad/dzi"

func main() {
    cfg := &dzi.Config{
        S3Host: "https://s3.example.com",
        S3Key: "access-key",
        S3Secret: "secret-key",
        S3Bucket: "dzi",
        TileSize: "1024",
        Overlap: "1",
        Resolution: 600,
        MinResolution: 200,
        MaxResolution: 1600,
        CoverHeight: "300",
        SplitChannels: true,
        Overprint: dzi.OverprintEnabled,
        DefaultDPI: 600,
        MaxSizePixels: 15000,
        MaxCpuCount: 4,
        ExtractText: true,
        TileFormat: "png",
        ICCProfileFilepath: "./icc/sRGB_Profile.icc",
        GraphicsAlphaBits: 4,
    }

    _, _ = dzi.Processing("https://example.com/source.pdf", 100500, cfg)
}
```
