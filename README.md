# ImageProcessor

Сервис фоновой обработки изображений на Go 1.25 с использованием фреймворка [wbf](https://github.com/wb-go/wbf).

## Описание

Сервис принимает изображения от пользователей, кладёт задачи на обработку в очередь (Apache Kafka), и в фоне обрабатывает файлы:

- Создание миниатюр (thumbnail)
- Изменение размера (resize)
- Добавление водяных знаков (watermark)

## Архитектура

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   HTTP API  │────▶│   Kafka     │────▶│   Worker    │
│  (ginext)   │     │  (queue)    │     │ (processor) │
└─────────────┘     └─────────────┘     └─────────────┘
       │                                       │
       ▼                                       ▼
┌─────────────┐                         ┌─────────────┐
│   Redis/    │                         │   File      │
│  Memory     │                         │  Storage    │
└─────────────┘                         └─────────────┘
```

## API

### POST /api/upload

Загрузка изображения на обработку.

**Request:**

- `Content-Type: multipart/form-data`
- Body: `file` — изображение (jpeg, png, gif)

**Response:**

```json
{
  "id": "uuid",
  "filename": "image.jpg",
  "status": "pending",
  "message": "Image uploaded successfully. Processing started."
}
```

### GET /api/image/:id

Получение обработанного изображения.

**Query параметры:**

- `type` — тип обработки: `thumbnail`, `resize`, `watermark`, `original`

**Response:** Файл изображения

### DELETE /api/image/:id

Удаление изображения.

**Response:**

```json
{
  "message": "Image deleted successfully"
}
```

### GET /api/status/:id

Получение статуса обработки.

**Response:**

```json
{
  "id": "uuid",
  "filename": "image.jpg",
  "status": "completed",
  "original_url": "http://localhost:8080/image/uuid?type=original",
  "processed_urls": {
    "thumbnail": "http://localhost:8080/image/uuid?type=thumbnail",
    "resize": "http://localhost:8080/image/uuid?type=resize",
    "watermark": "http://localhost:8080/image/uuid?type=watermark"
  },
  "created_at": "2026-03-12T10:00:00Z"
}
```

## Веб-интерфейс

Доступен по адресу `http://localhost:8080/`

Функции:

- Загрузка изображений через форму
- Отображение статуса обработки
- Просмотр и скачивание результатов
- Удаление изображений

## Запуск

### Требования

- Go 1.25+
- Docker и Docker Compose (для Kafka и Redis)

### 1. Запуск зависимостей (Kafka, Redis)

```bash
docker-compose up -d
```

### 2. Установка зависимостей Go

```bash
go mod tidy
```

### 3. Запуск сервиса

```bash
go run cmd/main.go
```

### Сборка бинарного файла

```bash
go build -o image-processor cmd/main.go
```

## Структура проекта

```
image-processor/
├── cmd/
│   └── main.go              # Точка входа
├── internal/
│   ├── config/
│   │   └── config.go        # Конфигурация
│   ├── handler/
│   │   ├── handler.go       # HTTP обработчики
│   │   └── kafka.go         # Kafka producer/consumer
│   ├── model/
│   │   └── model.go         # Модели данных
│   ├── router/
│   │   └── router.go        # HTTP роутер (ginext)
│   ├── service/
│   │   └── image.go         # Обработка изображений
│   ├── storage/
│   │   ├── storage.go       # Файловое хранилище
│   │   ├── status.go        # Redis хранилище статусов
│   │   └── memory.go        # In-memory хранилище
│   └── worker/
│       └── worker.go        # Фоновый воркер
├── templates/
│   └── index.html           # Веб-интерфейс
├── storage/
│   ├── originals/           # Оригинальные изображения
│   └── processed/           # Обработанные изображения
├── .env                     # Переменные окружения
├── docker-compose.yml       # Docker для Kafka и Redis
├── go.mod
└── README.md
```

## Поддерживаемые форматы

- JPEG
- PNG
- GIF

## Используемые пакеты wbf

- `github.com/wb-go/wbf/ginext` — HTTP сервер
- `github.com/wb-go/wbf/kafka` — Kafka producer/consumer
- `github.com/wb-go/wbf/redis` — Redis клиент
- `github.com/wb-go/wbf/config` — Конфигурация
- `github.com/wb-go/wbf/zlog` — Логирование
