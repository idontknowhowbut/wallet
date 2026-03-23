# Wallet Service

Небольшой REST-сервис на Go для тестового задания.

Что умеет:
- `POST /api/v1/wallet` — пополнение или списание средств
- `GET /api/v1/wallets/{walletId}` — получить баланс кошелька
- запуск приложения и базы через `docker compose`

Задание просит Go, PostgreSQL, Docker, `docker-compose`, `config.env`, тесты и аккуратную работу в конкурентной среде. В этом проекте изменение баланса сделано одним SQL-запросом, чтобы не было lost update при одновременных запросах по одному кошельку. fileciteturn6file0

## Структура

```text
cmd/app/main.go             - точка входа
internal/config             - чтение env
internal/pg                 - базовый модуль подключения к Postgres
internal/wallet             - handler + repository
migrations/init.sql         - создание таблицы
Dockerfile                  - контейнер приложения
docker-compose.yml          - приложение + postgres
config.env                  - переменные окружения
```

## Запуск

```bash
docker compose up --build
```

После запуска:
- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`

Проверка:

```bash
curl http://localhost:8080/health
```

## Примеры запросов

### Пополнение

```bash
curl -X POST http://localhost:8080/api/v1/wallet \
  -H "Content-Type: application/json" \
  -d '{
    "walletId": "550e8400-e29b-41d4-a716-446655440000",
    "operationType": "DEPOSIT",
    "amount": 1000
  }'
```

### Списание

```bash
curl -X POST http://localhost:8080/api/v1/wallet \
  -H "Content-Type: application/json" \
  -d '{
    "walletId": "550e8400-e29b-41d4-a716-446655440000",
    "operationType": "WITHDRAW",
    "amount": 300
  }'
```

### Получить баланс

```bash
curl http://localhost:8080/api/v1/wallets/550e8400-e29b-41d4-a716-446655440000
```
