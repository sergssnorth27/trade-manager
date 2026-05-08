# Миграции базы данных (goose + pgx)

## Структура файлов

```
trade-manager-1/
├── migrations/
│   ├── 20260508000001_init_items.sql
│   ├── 20260508000002_marketcsgo_prices.sql
│   ├── 20260508000003_steam_items.sql
│   ├── 20260508000004_steam_price_history.sql
│   ├── 20260508000005_steam_session.sql
│   └── 20260508000006_arbitrage_view.sql
├── cmd/
│   └── migrate/
│       └── main.go        ← точка запуска миграций
```

Миграции — отдельные файлы, не один большой `init.sql`. Каждая таблица — своя миграция.
Если нужно откатить только одну — можно.

---

## Формат файла миграции

Goose использует один файл с аннотациями вместо пары `up/down`:

```sql
-- migrations/20260508000001_init_items.sql

-- +goose Up
CREATE TABLE items (
    market_hash_name TEXT        PRIMARY KEY,
    classid          TEXT        NOT NULL,
    type             TEXT,
    commodity        BOOLEAN     NOT NULL DEFAULT false,
    tradable         BOOLEAN     NOT NULL DEFAULT true,
    icon_url         TEXT,
    name_color       TEXT,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE items;
```

`-- +goose Up` и `-- +goose Down` — обязательные маркеры, goose их ищет.

---

## Порядок миграций

Таблицы создаются в правильном порядке из-за FK (`REFERENCES`):

```
1. items                ← якорь, на неё ссылаются все остальные
2. marketcsgo_prices    ← REFERENCES items
3. steam_items          ← REFERENCES items
4. steam_price_history  ← REFERENCES items
5. steam_session        ← standalone, ни на что не ссылается
6. arbitrage_view       ← VIEW поверх items + marketcsgo_prices + steam_items
```

---

## cmd/migrate/main.go

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/jackc/pgx/v5/stdlib"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/joho/godotenv"
    "github.com/pressly/goose/v3"
)

func main() {
    godotenv.Load()

    dsn := os.Getenv("PG_DSN")
    if dsn == "" {
        log.Fatal("PG_DSN not set")
    }

    // goose работает через database/sql — используем stdlib-обёртку pgx
    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        log.Fatal("parse dsn: ", err)
    }

    db := stdlib.OpenDB(*cfg.ConnConfig)
    defer db.Close()

    if err := goose.SetDialect("postgres"); err != nil {
        log.Fatal(err)
    }

    // аргумент командной строки: up / down / status / reset
    command := "up"
    if len(os.Args) > 1 {
        command = os.Args[1]
    }

    if err := goose.RunContext(context.Background(), command, db, "migrations"); err != nil {
        log.Fatalf("goose %s: %v", command, err)
    }
}
```

---

## Команды

```powershell
# установить goose CLI (один раз)
go install github.com/pressly/goose/v3/cmd/goose@latest

# применить все новые миграции
go run ./cmd/migrate/ up

# откатить последнюю
go run ./cmd/migrate/ down

# посмотреть статус
go run ./cmd/migrate/ status

# применить только одну следующую
go run ./cmd/migrate/ up-by-one
```

Вывод `status`:

```
Applied At                  Migration
================================
2026-05-08 10:00:00 UTC  -- 20260508000001_init_items.sql
2026-05-08 10:00:00 UTC  -- 20260508000002_marketcsgo_prices.sql
Pending                  -- 20260508000003_steam_items.sql
```

---

## Проверка после применения

```sql
-- список таблиц
\dt

-- версии миграций (goose хранит здесь)
SELECT * FROM goose_db_version;

-- проверить схему конкретной таблицы
\d items
\d marketcsgo_prices
\d steam_price_history
```
