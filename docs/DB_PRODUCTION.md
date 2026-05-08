# Работа с БД и миграции в продакшене (Go)

---

## Часть 1 — Миграции

### Что такое миграция и зачем она нужна

Миграция — это SQL-файл с изменением схемы БД, у которого есть номер версии.
Инструмент миграций запускает их по порядку и запоминает, какие уже применены.

Без инструмента миграций происходит следующее:

- разработчик руками пишет ALTER TABLE в проде
- никто не знает в каком состоянии схема на каждом сервере
- откат невозможен
- при деплое новой версии приложения БД может быть несовместима

### Как выглядит миграция

Каждая миграция — это два SQL-файла:

```
migrations/
  001_init.up.sql       ← применить изменение
  001_init.down.sql     ← откатить изменение
  002_add_index.up.sql
  002_add_index.down.sql
```

Инструмент хранит в таблице `schema_migrations` какая версия применена:

```sql
SELECT * FROM schema_migrations;
-- version | dirty
-- 1       | false
-- 2       | false
```

`dirty = true` означает что миграция упала на полпути — требует ручного вмешательства.

---

### Инструменты миграций в Go

#### golang-migrate — стандарт де-факто

```
github.com/golang-migrate/migrate/v4
```

Используется в большинстве Go-проектов. Простой, стабильный, без магии.

```go
m, err := migrate.New(
    "file://migrations",
    "postgres://user:pass@localhost/db",
)
m.Up() // применить все новые
m.Steps(1) // применить одну
m.Steps(-1) // откатить одну
```

Почему его выбирают:

- поддерживает 20+ источников (файлы, S3, embed, GitHub)
- поддерживает 20+ баз данных
- запускается и как библиотека в коде, и как CLI
- активно поддерживается

Минус: нет поддержки Go-функций в миграциях (только SQL).

---

#### goose — альтернатива с Go-миграциями

```
github.com/pressly/goose/v3
```

Отличается от golang-migrate двумя вещами:

1. Миграции можно писать на Go:

```go
func init() {
    goose.AddMigration(upAddUsers, downAddUsers)
}

func upAddUsers(tx *sql.Tx) error {
    _, err := tx.Exec(`ALTER TABLE items ADD COLUMN description TEXT`)
    return err
}
```

1. Нумерация через timestamp вместо последовательных номеров:

```
20240822120000_add_index.sql  ← меньше конфликтов при параллельной разработке
```

Используется в Cloudflare, Buffer и других компаниях.

---

#### Atlas — современный подход (schema-first)

```
ariga.io/atlas
```

Принципиально другая концепция: ты описываешь **желаемое состояние схемы**, Atlas сам генерирует миграции.

```hcl
table "items" {
  schema = schema.public
  column "market_hash_name" {
    type = text
  }
  primary_key {
    columns = [column.market_hash_name]
  }
}
```

```bash
atlas schema apply --url "postgres://..." --to "file://schema.hcl"
# Atlas сам напишет ALTER TABLE
```

Используется в больших командах где schema drift (расхождение схемы на разных серверах) — реальная проблема. Набирает популярность, но сложнее в освоении.

---

### Как запускать миграции в продакшене

**Плохо:** запускать миграции внутри `main()` при старте приложения.

Причина: если задеплоили 3 инстанса одновременно, все три попытаются мигрировать одновременно. Даже с блокировками это приводит к грязным миграциям и непредсказуемому поведению.

**Хорошо:** отдельная джоба в CI/CD до деплоя приложения.

```yaml
# в GitHub Actions / Kubernetes
steps:
  - name: Run migrations
    run: migrate -path ./migrations -database $DATABASE_URL up

  - name: Deploy app
    run: kubectl apply -f deployment.yaml
```

Логика: сначала схема, потом код. Приложение стартует уже с готовой схемой.

**В Kubernetes** это обычно `initContainer` или отдельный `Job`.

---

### Правила безопасных миграций в проде

**1. Никогда не делай breaking change в одном деплое**

Нельзя удалить колонку и задеплоить новый код одновременно — старый код ещё читает эту колонку на соседних подах.

Правильный порядок через несколько деплоев:

```
Деплой 1: ADD COLUMN new_col (nullable)
Деплой 2: код начинает писать в new_col
Деплой 3: бэкфилл данных (UPDATE ... SET new_col = ...)
Деплой 4: ADD NOT NULL CONSTRAINT
Деплой 5: DROP COLUMN old_col
```

**2. Избегай долгих блокировок**

`ALTER TABLE ADD COLUMN NOT NULL` на большой таблице блокирует её на минуты.
В проде используют `ADD COLUMN` (nullable) + отдельный `UPDATE` батчами.

**3. Всегда пиши down-миграцию**

Даже если никогда не откатываешь — это документация того, что делает up.

---

## Часть 2 — Работа с БД в Go-коде

### Уровни абстракции

```
database/sql    — стандартная библиотека, низкий уровень
pgx             — драйвер PostgreSQL, заменяет database/sql
sqlx            — тонкая обёртка над database/sql
sqlc            — генерация Go-кода из SQL-запросов
squirrel        — билдер SQL-запросов
GORM / ent      — полноценные ORM
```

---

### pgx — основной драйвер для PostgreSQL

```
github.com/jackc/pgx/v5
```

Используется в подавляющем большинстве Go+Postgres проектов.

Два режима:

```go
// Режим 1: через стандартный database/sql интерфейс
import "github.com/jackc/pgx/v5/stdlib"
db, _ := sql.Open("pgx", connStr)

// Режим 2: нативный pgx (быстрее, больше возможностей)
conn, _ := pgx.Connect(ctx, connStr)
rows, _ := conn.Query(ctx, "SELECT ...", args...)
```

Нативный pgx лучше: поддерживает pgx-типы (pgtype.Text, pgtype.Int4), prepared statements, COPY protocol для bulk insert, лучшую работу с NULL.

**pgxpool** — пул соединений:

```go
pool, _ := pgxpool.New(ctx, connStr)
// pool.Acquire(), pool.QueryRow(), pool.Exec() — thread-safe
```

В продакшене всегда используется пул, не одиночное соединение.

---

### sqlc — генерация кода из SQL (популярный подход)

```
github.com/sqlc-dev/sqlc
```

Концепция: ты пишешь SQL-запросы, sqlc генерирует типизированный Go-код.

```sql
-- query.sql
-- name: GetItem :one
SELECT market_hash_name, type, commodity
FROM items
WHERE market_hash_name = $1;
```

```bash
sqlc generate
```

Получаешь:

```go
// автосгенерированный код — не редактировать
func (q *Queries) GetItem(ctx context.Context, marketHashName string) (Item, error) {
    row := q.db.QueryRow(ctx, getItem, marketHashName)
    var i Item
    err := row.Scan(&i.MarketHashName, &i.Type, &i.Commodity)
    return i, err
}
```

Почему это хорошо:

- SQL пишешь сам — полный контроль над запросами
- Go-код типизирован и безопасен на этапе компиляции
- нет runtime-ошибок типа "колонка не та"
- нет магии ORM

Используется в Stripe, PlanetScale и многих других.

---

### GORM — ORM (популярен, но спорен)

```
gorm.io/gorm
```

```go
db.Where("price_cents < ?", 1000).Find(&items)
db.Create(&item)
db.Save(&item)
```

Удобен для быстрого старта и CRUD-операций. Генерирует SQL автоматически.

Почему в проде часто от него уходят:

- сложные запросы превращаются в нечитаемые цепочки
- N+1 проблема (делает много маленьких запросов вместо одного JOIN)
- скрытые запросы при обращении к полям (preload)
- сложно отлаживать что именно ушло в БД
- медленнее sqlc/pgx на высоких нагрузках

Вывод: GORM ок для небольших сервисов и прототипов. В высоконагруженных системах предпочитают sqlc или raw SQL через pgx.

---

### Как выглядит типичный стек в Go-продакшене

**Маленькая/средняя команда:**

```
pgx/pgxpool  +  sqlc  +  golang-migrate
```

Максимальный контроль, типобезопасность, простота.

**Команда хочет скорость разработки:**

```
pgx/pgxpool  +  GORM  +  goose
```

Меньше кода, но меньше контроля.

**Большая команда, сложная схема:**

```
pgx/pgxpool  +  sqlc  +  Atlas
```

Atlas следит за schema drift между окружениями.

---

### Пример организации кода в проекте

```
internal/
└── store/
    ├── postgres.go      ← подключение, pgxpool
    ├── queries.sql      ← SQL-запросы (для sqlc) или raw
    ├── models.go        ← типы (генерирует sqlc или пишешь сам)
    └── store.go         ← интерфейс Store + реализация

migrations/
    001_init.up.sql
    001_init.down.sql
    002_steam_session.up.sql
    002_steam_session.down.sql
```

`Store` оборачивается в интерфейс чтобы в тестах можно было подменить на моки:

```go
type Store interface {
    GetItem(ctx context.Context, hashName string) (Item, error)
    UpsertMarketPrice(ctx context.Context, arg UpsertMarketPriceParams) error
    // ...
}
```

---

### Транзакции

В продакшене операции которые затрагивают несколько таблиц всегда оборачиваются в транзакцию:

```go
tx, err := pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx) // откат если что-то пошло не так

// несколько операций
_, err = tx.Exec(ctx, "INSERT INTO items ...", ...)
_, err = tx.Exec(ctx, "INSERT INTO steam_items ...", ...)

return tx.Commit(ctx) // фиксируем только если всё прошло
```

`defer tx.Rollback()` — идиома Go: если `Commit()` был вызван успешно, `Rollback()` на уже закоммиченной транзакции — no-op. Если что-то упало — откат гарантирован.
