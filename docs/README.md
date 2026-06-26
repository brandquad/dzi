# DZI Processor Documentation

Документация описывает проект `github.com/brandquad/dzi`: Go-библиотеку и CLI для подготовки Deep Zoom Image (DZI) артефактов из PDF, изображений и презентаций.

## Содержание

- [Обзор проекта](./overview.md)
- [Установка и запуск](./setup-and-run.md)
- [Конфигурация](./configuration.md)
- [Пайплайн обработки](./processing-pipeline.md)
- [Формат manifest.json](./manifest.md)
- [Эксплуатация и диагностика](./operations.md)

## Быстрый старт

Минимальный запуск CLI требует URL исходного файла, `assetId` и S3-настроек через переменные окружения:

```bash
export DZI_S3_HOST="https://s3.example.com"
export DZI_S3_KEY="access-key"
export DZI_S3_SECRET="secret-key"
export DZI_BUCKET="dzi"

go run ./cmd "https://example.com/file.pdf" 12345
```

В обычном режиме результат складывается во временную директорию, синхронизируется в S3 через `mc`, после чего временные файлы удаляются. Для локальной отладки включите `DEBUG=1` или `DZI_DEBUG=true`: артефакты останутся в `_tmp/<assetId>`.
