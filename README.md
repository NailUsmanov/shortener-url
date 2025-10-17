# URL Shortener (Go)

Сервис сокращения ссылок на Go с поддержкой HTTP и gRPC, авторизацией по cookie, gzip-сжатием, переключаемыми типами хранилищ (in‑memory, файл, PostgreSQL) и полным набором автотестов. Проект включает CLI‑клиент, HTTP/gRPC‑сервер, миграции БД и профилирование через pprof.

## Цель проекта

Реализовать надежный и расширяемый сервис сокращения URL, который:
- принимает длинные URL и выдает короткие идентификаторы;
- выполняет редирект по коротким ссылкам;
- поддерживает пакетные операции и учет ссылок по пользователю;
- работает с различными типами хранилищ (память, файл, PostgreSQL);
- предоставляет HTTP и gRPC интерфейсы, покрыт тестами и статическим анализом.

## Выполненные задачи

- HTTP API на базе chi/v5: создание коротких ссылок, редирект, пакетные операции, список и удаление ссылок пользователя, /ping.  
- gRPC API (proto в `api/shortener/v1/shortener.proto`, сгенерированный код в `internal/genproto/shortener/v1`), отдельный gRPC‑сервер на :3200.  
- Три варианта хранилища: in‑memory (`internal/storage/memory.go`), файловое (`filestorage.go`), PostgreSQL (`postgres.go`) с миграциями (`migrations`).  
- Авторизация через cookie и привязка ссылок к пользователю (middleware `auth`).  
- gzip‑сжатие ответов и логирование запросов (middleware `compress`, `logger`).  
- Конфигурация через переменные окружения/файл (`pkg/config`).  
- Профилирование через pprof на :6060.  
- Юнит‑тесты для хендлеров, сервисов и хранилищ; бенчмарки для слоя storage.  
- Настроен CI: запуск тестов и статического анализа (workflows в `.github/workflows`).  
- Статический анализ: errcheck, ineffassign, bodyclose, gosec, staticcheck, honnef/tools и др.

## Технологии

- Go 1.24+  
- Chi (маршрутизация HTTP)  
- PostgreSQL, pgx/v5 (драйвер) и golang‑migrate (миграции)  
- gRPC, Protobuf (контракты и сервер)  
- Zap (структурированное логирование)  
- Testify (тесты), bench‑тесты для storage  
- Статический анализ: errcheck, ineffassign, bodyclose, gosec, staticcheck  
- pprof (профилирование)

## Результаты

- Реализован производительный сервис сокращения ссылок с HTTP и gRPC интерфейсами.  
- Поддержаны три типа хранилища с миграциями для PostgreSQL.  
- Добавлены авторизация по cookie, gzip‑сжатие и централизованное логирование.  
- Покрыт тестами ключевой функционал; включены бенчмарки слоя хранения.  
- Настроен CI и статический анализ.  
- Проект структурирован по слоям и готов к дальнейшему расширению.

## Архитектура

```
.
├── api/shortener/v1/shortener.proto      # gRPC контракт
├── cmd/
│   ├── client/                           # CLI‑клиент
│   └── shortener/                        # HTTP/gRPC сервер (main.go)
├── internal/
│   ├── app/                              # Инициализация HTTP‑приложения
│   ├── genproto/shortener/v1/            # gRPC сгенерированные типы
│   ├── grpcserver/                       # gRPC‑сервер, перехватчики
│   ├── handlers/                         # HTTP‑хендлеры (create, redirect, delete, batch, list)
│   ├── middleware/                       # auth, compress, logger
│   ├── models/                           # доменные структуры
│   ├── storage/                          # memory, file, postgres (интерфейс + реализации)
│   └── tasks/                            # фоновые задачи (при необходимости)
├── migrations/                           # SQL‑миграции для PostgreSQL
├── pkg/config/                           # конфиг и парсинг env
└── profiles/                             # pprof профили
```


## Конфигурация

Основные переменные окружения (поддерживаются через `pkg/config`):

- `RUN_ADDRESS` — адрес HTTP‑сервера, например `:8080`  
- `BASE_URL` — базовый URL коротких ссылок, например `http://localhost:8080`  
- `DATABASE_DSN` — строка подключения к PostgreSQL (если не пустая — включается режим БД)  
- `SAVE_IN_FILE` — путь к файлу хранения (если не пустой — используется файловое хранилище)  
- `ENABLE_HTTPS` — включить HTTPS для HTTP‑сервера (`true/false`)  
- `CERT_FILE`, `KEY_FILE` — пути к TLS‑сертификату и ключу (если `ENABLE_HTTPS=true`)  
- `TRUSTED_SUBNET` — CIDR доверенной подсети (для внутренних эндпоинтов, если используются)

Приоритет выбора хранилища (см. `cmd/shortener/main.go`):  
1) если задан `SAVE_IN_FILE` — файловое хранилище;  
2) иначе если задан `DATABASE_DSN` — PostgreSQL;  
3) иначе — in‑memory.

## Запуск

### 1) С PostgreSQL

1. Запустить БД и применить миграции (файлы в `migrations`).  
2. Установить переменные окружения, например:
```env
RUN_ADDRESS=:8080
BASE_URL=http://localhost:8080
DATABASE_DSN=postgres://user:password@localhost:5432/shortener?sslmode=disable
```
3. Запустить сервер:
```bash
go run ./cmd/shortener
```
gRPC‑сервер стартует на `:3200`, pprof — на `:6060`.

### 2) С файловым хранилищем
```env
RUN_ADDRESS=:8080
BASE_URL=http://localhost:8080
SAVE_IN_FILE=/path/to/shortener.db.json
```
```bash
go run ./cmd/shortener
```

### 3) In‑memory (по умолчанию)
```env
RUN_ADDRESS=:8080
BASE_URL=http://localhost:8080
```
```bash
go run ./cmd/shortener
```

## HTTP API (основные эндпоинты)

| Метод | Путь | Назначение |
|------|------|------------|
| POST | `/` | Создать короткую ссылку (тело: text/plain с длинным URL) |
| POST | `/api/shorten` | Создать короткую ссылку (тело: JSON `{"url": "..."}`) |
| POST | `/api/shorten/batch` | Пакетное создание ссылок |
| GET  | `/{id}` | Редирект по короткому идентификатору |
| GET  | `/api/user/urls` | Список ссылок текущего пользователя |
| DELETE | `/api/user/urls` | Пакетное удаление ссылок пользователя |
| GET  | `/ping` | Проверка доступности БД |
| GET  | `/api/internal/stats` | Внутренняя статистика (доступ из `TRUSTED_SUBNET`, если реализовано) |

Авторизация пользователя выполняется через cookie (middleware `auth`). Ответы автоматически сжимаются, если клиент поддерживает gzip.

## gRPC API

Контракт расположен в `api/shortener/v1/shortener.proto`, сгенерированный код — в `internal/genproto/shortener/v1`.  
Сервер поднимается на `:3200` (`internal/grpcserver`). Набор RPC соответствует функциям HTTP‑API: укорочение, пакетное укорочение, получение/перенаправление, список/удаление ссылок пользователя, ping.

## Тестирование и качество

- Юнит‑тесты для хендлеров, приложения и хранилищ (`*_test.go`).  
- Бенчмарки для слоя хранения (`internal/storage/storage_bench_test.go`).  
- CI‑пайплайны для тестов и статики в `.github/workflows`.  
- Статический анализ: errcheck, ineffassign, bodyclose, gosec, staticcheck, honnef/tools.  

Запуск тестов:
```bash
go test ./... -v
```


