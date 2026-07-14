# URL Shortener

Сервис для сокращения ссылок, написанный на Go.

Сервис принимает обычный URL, сохраняет его и возвращает короткую ссылку с кодом длиной 10 символов.

Для одного и того же URL всегда возвращается один и тот же короткий код.

## Что реализовано

- создание короткой ссылки через POST-запрос;
- получение оригинального URL по короткому коду;
- перенаправление по короткой ссылке;
- хранение данных в памяти;
- хранение данных в PostgreSQL;
- выбор хранилища при запуске;
- Dockerfile и Docker Compose;
- запуск нескольких экземпляров приложения через Nginx;
- unit-тесты;
- health-check endpoints;
- корректное завершение работы сервиса.

## Требования

Для локального запуска нужны:

- Go 1.25 или новее;
- Docker и Docker Compose для запуска с PostgreSQL.

## Запуск с хранением в памяти

Сначала нужно скачать зависимости:

```bash
go mod download
```

Запуск:

```bash
go run ./cmd/shortener -storage=memory
```

После запуска сервис будет доступен по адресу:

```text
http://localhost:8080
```

В этом режиме данные хранятся только в памяти и пропадут после остановки программы.

## Запуск через Docker Compose

```bash
docker compose up --build
```

Будут запущены:

- PostgreSQL;
- три контейнера приложения;
- Nginx для распределения запросов.

Сервис будет доступен по адресу:

```text
http://localhost:8080
```

Остановка:

```bash
docker compose down
```

Если нужно удалить также данные PostgreSQL:

```bash
docker compose down -v
```

## Переменные окружения

Пример настроек находится в файле `.env.example`.

Основные переменные:

```text
STORAGE_TYPE
APP_ADDR
BASE_URL
DATABASE_URL
DB_MAX_CONNS
DB_MIN_CONNS
```

Пример:

```env
STORAGE_TYPE=memory
APP_ADDR=:8080
BASE_URL=http://localhost:8080
DATABASE_URL=postgres://shortener:shortener@postgres:5432/shortener?sslmode=disable
DB_MAX_CONNS=10
DB_MIN_CONNS=2
```

## Создание короткой ссылки

Запрос:

```bash
curl -i -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/some/long/path"}'
```

Пример ответа:

```json
{
  "code": "aB3_q9ZxK2",
  "short_url": "http://localhost:8080/aB3_q9ZxK2"
}
```

Если отправить тот же URL ещё раз, сервис вернёт тот же код.

Для новой ссылки возвращается статус:

```text
201 Created
```

Для уже существующей ссылки:

```text
200 OK
```

## Получение оригинального URL

```bash
curl http://localhost:8080/api/v1/links/aB3_q9ZxK2
```

Пример ответа:

```json
{
  "url": "https://example.com/some/long/path"
}
```

Если код не найден, сервис вернёт:

```text
404 Not Found
```

## Перенаправление

Можно открыть короткую ссылку в браузере:

```text
http://localhost:8080/aB3_q9ZxK2
```

Сервис перенаправит на оригинальный URL с помощью статуса:

```text
302 Found
```

## Проверка состояния

Проверка, что процесс работает:

```bash
curl http://localhost:8080/health/live
```

Проверка, что сервис готов принимать запросы:

```bash
curl http://localhost:8080/health/ready
```

## Тесты

Запуск всех тестов:

```bash
go test ./...
```

Проверка гонок данных:

```bash
go test -race ./...
```

Дополнительная проверка:

```bash
go vet ./...
```

Проверка сборки:

```bash
go build ./cmd/shortener
```

## Хранилища

Поддерживаются два режима:

```text
memory
postgres
```

### Memory

Данные хранятся в оперативной памяти одного процесса.

После остановки приложения данные удаляются.

Этот режим подходит для локального запуска и тестов.

### PostgreSQL

Данные хранятся в PostgreSQL и сохраняются после перезапуска приложения.

Этот режим используется в Docker Compose и подходит для запуска нескольких экземпляров приложения, так как все контейнеры работают с одной базой.

## Короткие коды

Код состоит ровно из 10 символов.

Используются:

```text
a-z
A-Z
0-9
_
```

Коды генерируются через `crypto/rand`.

Уникальность дополнительно проверяется хранилищем. Если код уже занят, сервис генерирует новый.

## Валидация URL

Сервис принимает только полные URL, начинающиеся с:

```text
http://
https://
```

Например, этот URL будет принят:

```text
https://example.com/page
```

А этот будет отклонён:

```text
example.com/page
```

Перед сохранением пробелы в начале и конце строки удаляются.

Если у URL нет пути, сервис добавляет `/`, поэтому эти два адреса считаются одним URL:

```text
https://example.com
https://example.com/
```

При этом адреса:

```text
https://example.com/page
https://example.com/page/
```

считаются разными, так как завершающий `/` в пути может менять смысл адреса.

## Структура проекта

```text
cmd/shortener              точка запуска приложения
internal/config            чтение настроек
internal/generator         генерация коротких кодов
internal/service           основная логика
internal/storage/memory    хранение в памяти
internal/storage/postgres  хранение в PostgreSQL
internal/transport/httpapi HTTP-обработчики
migrations                 SQL-миграция
deploy/nginx               конфигурация Nginx
```

## Масштабирование

В режиме PostgreSQL можно запускать несколько экземпляров приложения.

В Docker Compose запускаются три контейнера сервиса:

```text
app1
app2
app3
```

Запросы между ними распределяет Nginx.

Все экземпляры используют одну PostgreSQL, поэтому любой контейнер может обработать любой запрос.

Режим `memory` для такого запуска не подходит, так как у каждого контейнера будет собственная память.

## Миграция PostgreSQL

SQL-миграция находится в папке:

```text
migrations
```

При первом запуске Docker Compose она создаёт таблицу для хранения ссылок.

Если миграция была изменена и нужно пересоздать локальную базу:

```bash
docker compose down -v
docker compose up --build
```

Команда `down -v` удаляет данные PostgreSQL.