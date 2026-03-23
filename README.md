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

## Как работает конкурентная безопасность

Опасный вариант — сначала читать баланс, потом в Go его менять, потом делать `UPDATE`. Тогда при нескольких одновременных запросах можно потерять часть обновлений.

Здесь сделано проще и надежнее:
- `DEPOSIT` — через `INSERT ... ON CONFLICT DO UPDATE`
- `WITHDRAW` — через один `UPDATE ... WHERE balance >= $1`

Примеры SQL:

```sql
-- deposit
INSERT INTO wallets (id, balance, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (id) DO UPDATE
SET balance = wallets.balance + EXCLUDED.balance,
    updated_at = NOW()
RETURNING balance;

-- withdraw
UPDATE wallets
SET balance = balance - $1,
    updated_at = NOW()
WHERE id = $2
  AND balance >= $1
RETURNING balance;
```

Это работает надежнее в конкурентной среде, потому что изменение делается одним SQL-стейтментом, а синхронизацию изменений строки берет на себя Postgres.

## Как инициализируется база

Файл `migrations/init.sql` автоматически применяется контейнером Postgres при первом запуске, если volume базы пустой.

## Как база сохраняется между перезапусками

В `docker-compose.yml` подключен named volume:

```yaml
volumes:
  - postgres_data:/var/lib/postgresql/data
```

Поэтому данные сохраняются между перезапусками контейнеров.

Если выполнить:

```bash
docker compose down
```

данные останутся.

Если выполнить:

```bash
docker compose down -v
```

volume удалится, и база создастся заново.

## Тесты

Есть базовые unit-тесты для HTTP-слоя:

```bash
go test ./...
```

## Замечания

- Кошелек автоматически создается при первом `DEPOSIT`.
- Для `WITHDRAW` несуществующий кошелек возвращает `404`.
- Если денег не хватает, сервис возвращает `409`.
- Приложение специально сделано без лишней архитектурной сложности, чтобы выглядело как аккуратное джунское решение.
