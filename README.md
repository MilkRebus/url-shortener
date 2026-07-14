# URL Shortener

Сервис сокращения ссылок на Go. Для одного оригинального URL всегда возвращается одна сокращённая ссылка. Короткий код состоит ровно из 10 символов: `a-z`, `A-Z`, `0-9` и `_`.

## Возможности

- `POST /api/v1/links` — создать или получить короткую ссылку.
- `GET /api/v1/links/{code}` — получить оригинальный URL в JSON.
- `GET /{code}` — перенаправиться на оригинальный URL через `302 Found`.
- Два хранилища: потокобезопасное in-memory и PostgreSQL.
- Выбор хранилища параметром запуска или переменной окружения.
- Генерация кодов через `crypto/rand` с равномерным выбором символов.
- Атомарная обработка одинаковых URL и коллизий кодов.
- Health checks, таймауты HTTP-сервера и graceful shutdown.
- Docker-образ и масштабируемый Docker Compose с тремя репликами приложения и Nginx.
- Unit-тесты и конкурентные тесты.

## Архитектура

```text
Client
  |
Nginx
  |---- app1 ----|
  |---- app2 ----|---- PostgreSQL
  |---- app3 ----|
```

Приложение stateless в PostgreSQL-режиме: любой контейнер может обработать любой запрос. Уникальность кода и оригинального URL гарантируется ограничениями PostgreSQL. In-memory режим предназначен для локального запуска, тестирования и одной реплики.

## Структура проекта

```text
.
├── cmd/shortener/main.go
├── internal/
│   ├── config/
│   ├── generator/
│   ├── service/
│   ├── storage/
│   │   ├── memory/
│   │   └── postgres/
│   └── transport/httpapi/
├── migrations/001_create_links.sql
├── deploy/nginx/nginx.conf
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## Требования

- Go 1.25 или новее.
- Docker и Docker Compose для контейнерного запуска.
- PostgreSQL для режима `postgres`.

После клонирования загрузите зависимости:

```bash
go mod tidy
```

## Локальный запуск с in-memory хранилищем

```bash
go run ./cmd/shortener -storage=memory -addr=:8080 -base-url=http://localhost:8080
```

Аналогично через переменные окружения:

```bash
STORAGE_TYPE=memory APP_ADDR=:8080 BASE_URL=http://localhost:8080 go run ./cmd/shortener
```

Данные in-memory хранилища теряются после завершения процесса.

## Запуск с PostgreSQL

Сначала примените миграцию `migrations/001_create_links.sql`, затем запустите сервис:

```bash
go run ./cmd/shortener \
  -storage=postgres \
  -database-url='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable' \
  -addr=:8080 \
  -base-url=http://localhost:8080
```

## Масштабируемый запуск через Docker Compose

```bash
docker compose up --build
```

Будут запущены:

- PostgreSQL;
- три одинаковых контейнера приложения;
- Nginx, распределяющий запросы между контейнерами.

Сервис будет доступен на `http://localhost:8080`.

Остановка:

```bash
docker compose down
```

Удаление базы и повторное применение init-миграции:

```bash
docker compose down -v
```

## API

### Создание короткой ссылки

```http
POST /api/v1/links
Content-Type: application/json
```

```json
{
  "url": "https://example.com/very/long/path"
}
```

Новая ссылка возвращается со статусом `201 Created`:

```json
{
  "code": "aB3_q9ZxK2",
  "short_url": "http://localhost:8080/aB3_q9ZxK2"
}
```

Если URL уже существует, возвращается тот же код и статус `200 OK`.

Пример с `curl`:

```bash
curl -i -X POST http://localhost:8080/api/v1/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/very/long/path"}'
```

### Получение оригинального URL

```bash
curl http://localhost:8080/api/v1/links/aB3_q9ZxK2
```

```json
{
  "url": "https://example.com/very/long/path"
}
```

### Перенаправление

```bash
curl -i http://localhost:8080/aB3_q9ZxK2
```

Ответ содержит статус `302 Found` и заголовок `Location`.

### Health checks

```text
GET /health/live
GET /health/ready
```

`live` проверяет, что процесс работает. `ready` в PostgreSQL-режиме дополнительно проверяет доступность базы.

## Генерация короткого кода

Алфавит содержит 63 символа:

```text
abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_
```

Код создаётся из 10 криптографически случайных символов. Для исключения смещения распределения байты `252-255` отбрасываются, потому что `252` делится на `63` без остатка.

Уникальность не полагается только на вероятность:

1. сервис генерирует код;
2. хранилище атомарно пытается сохранить его;
3. при конфликте кода сервис генерирует новый;
4. число попыток ограничено параметром `GENERATION_ATTEMPTS`.

## Конкурентность

### In-memory

Хранилище использует две map и `sync.RWMutex`:

```text
code -> original URL
original URL -> code
```

Проверка существующего URL, проверка коллизии и запись выполняются под одной write-блокировкой.

### PostgreSQL

Таблица содержит:

```sql
PRIMARY KEY (code)
UNIQUE (original_url)
```

Сначала выполняется `INSERT ... ON CONFLICT DO NOTHING RETURNING ...`. Если запись не вставлена, сервис проверяет существующий URL. Если URL отсутствует, конфликт произошёл по короткому коду и генерация повторяется.

Такая схема корректна при одновременной работе нескольких контейнеров и не требует распределённого mutex.

## Конфигурация

| Переменная | По умолчанию | Описание |
|---|---:|---|
| `APP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `BASE_URL` | `http://localhost:8080` | Публичный адрес коротких ссылок |
| `STORAGE_TYPE` | `memory` | `memory` или `postgres` |
| `DATABASE_URL` | пусто | Строка подключения PostgreSQL |
| `DB_MAX_CONNS` | `10` | Максимум соединений на контейнер |
| `DB_MIN_CONNS` | `1` | Минимум соединений на контейнер |
| `GENERATION_ATTEMPTS` | `10` | Максимум попыток при коллизиях |
| `READ_HEADER_TIMEOUT` | `5s` | Таймаут заголовков |
| `READ_TIMEOUT` | `10s` | Таймаут чтения запроса |
| `WRITE_TIMEOUT` | `10s` | Таймаут ответа |
| `IDLE_TIMEOUT` | `60s` | Таймаут keep-alive соединения |
| `SHUTDOWN_TIMEOUT` | `10s` | Время graceful shutdown |

Все основные параметры также доступны как CLI-флаги:

```bash
go run ./cmd/shortener -h
```

## Тесты

```bash
go test ./...
```

Проверка гонок:

```bash
go test -race ./...
```

Дополнительно:

```bash
go vet ./...
```

Тестами покрыты:

- формат и ошибки генератора;
- in-memory хранилище и параллельное создание одного URL;
- повторная генерация после коллизии;
- PostgreSQL-логика через подменяемый интерфейс базы;
- HTTP-обработчики, статусы, JSON, redirect и readiness;
- конфигурация.

## Основные решения и ограничения

- URL сравниваются как строки после удаления пробелов по краям. `https://example.com` и `https://example.com/` считаются разными URL.
- Поддерживаются только абсолютные URL со схемой `http` или `https`.
- In-memory хранилище не подходит для нескольких реплик и не сохраняет данные после рестарта.
- Docker Compose использует PostgreSQL init-скрипт для демонстрации. В production миграции следует запускать отдельным migration job.
- Для дальнейшего роста можно добавить Redis-кэш, rate limiting, срок жизни ссылок, аналитику через очередь и реплики PostgreSQL.
