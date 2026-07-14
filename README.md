# URL Shortener

Небольшой сервис для сокращения ссылок, написанный на Go.

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
- unit-тесты.

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

Данные в этом режиме хранятся только в памяти и пропадут после остановки программы.

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

## Перенаправление

Можно открыть короткую ссылку в браузере:

```text
http://localhost:8080/aB3_q9ZxK2
```

Сервис перенаправит на оригинальный URL.

## Проверка состояния

```bash
curl http://localhost:8080/health/live
```

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

## Хранилища

Поддерживаются два режима:

```text
memory
postgres
```

`memory` подходит для локального запуска и тестов.

`postgres` используется в Docker Compose и подходит для запуска нескольких экземпляров приложения, так как все контейнеры работают с одной базой.

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

## Примечания

Сервис принимает только URL со схемой `http` или `https`.

URL сравниваются как строки после удаления пробелов по краям. Поэтому эти адреса считаются разными:

```text
https://example.com
https://example.com/
```

SQL-миграция из папки `migrations` автоматически применяется при первом создании PostgreSQL-контейнера.