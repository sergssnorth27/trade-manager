# Database Schema

## Ключевой принцип матчинга

`market_hash_name` — единственный общий ключ между MarketCSGO и Steam.
Все таблицы связаны через него.

```
marketcsgo_prices ──┐
                     ├── items (market_hash_name) ── steam_items
steam_price_history─┘                             └── steam_price_history
```

---

## Таблицы

### `items` — справочник предметов

Одна строка на уникальный скин. **Заполняется из Steam Market Search** — там есть все
предметы CS2, тогда как на MarketCSGO часть предметов может отсутствовать.

```sql
CREATE TABLE items (
    market_hash_name TEXT        PRIMARY KEY,
    classid          TEXT        NOT NULL,
    type             TEXT,                               -- "Rifle", "Pistol", "Knife", ...
    commodity        BOOLEAN     NOT NULL DEFAULT false, -- true = кейсы/патчи (все одинаковые)
    tradable         BOOLEAN     NOT NULL DEFAULT true,
    icon_url         TEXT,
    name_color       TEXT,                               -- hex, определяет редкость
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### `marketcsgo_prices` — минимальные цены MarketCSGO

Одна строка на предмет, только минимальная цена. Обновляется при каждом синке full-export.
Индивидуальные лоты (float, стикеры) не хранятся — нас интересует только арбитраж.

```sql
CREATE TABLE marketcsgo_prices (
    market_hash_name TEXT        PRIMARY KEY REFERENCES items(market_hash_name),
    min_price_cents  INT         NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### `steam_items` — снапшот Steam Market Search

Одна строка на предмет. Обновляется раз в 15 минут.
Даёт минимальную цену продажи и количество активных лотов.

```sql
CREATE TABLE steam_items (
    market_hash_name TEXT        PRIMARY KEY REFERENCES items(market_hash_name),
    sell_listings    INT         NOT NULL DEFAULT 0, -- кол-во активных лотов на продажу
    sell_price_cents INT         NOT NULL DEFAULT 0, -- минимальная цена продажи в центах
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### `steam_price_history` — история цен Steam (дневная)

Одна строка на предмет в день. Обновляется раз в час.

Почасовые данные (последние ~30 дней) агрегируются в дневные перед записью:
все часы одного дня → среднее `avg_price`, сумма `volume`.

```sql
CREATE TABLE steam_price_history (
    market_hash_name TEXT          NOT NULL REFERENCES items(market_hash_name),
    date             DATE          NOT NULL, -- UTC
    avg_price        NUMERIC(10,4) NOT NULL, -- средняя цена за день в долларах
    volume           INT           NOT NULL, -- кол-во продаж за день
    PRIMARY KEY (market_hash_name, date)
);

CREATE INDEX ON steam_price_history (market_hash_name, date DESC);
```

---

### `steam_session` — сессия авторизации Steam

Всегда одна строка (id = 1).
Хранит refresh token — позволяет восстановить сессию без credentials и Steam Guard (~200 дней).

```sql
CREATE TABLE steam_session (
    id                  INT  PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    login               TEXT NOT NULL,
    steam_refresh_token TEXT NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## VIEW для арбитража

Сводит минимальную цену покупки (MarketCSGO) и минимальную цену продажи (Steam).
Арбитражный движок читает отсюда.

```sql
CREATE VIEW arbitrage_view AS
SELECT
    i.market_hash_name,
    i.type,
    i.commodity,
    mp.min_price_cents                                          AS buy_price_cents,  -- мин. цена на MarketCSGO
    si.sell_price_cents                                         AS steam_sell_cents, -- мин. цена в Steam
    si.sell_listings,
    ROUND(si.sell_price_cents * 0.85)                          AS net_cents,        -- после комиссии Steam 15%
    ROUND(
        (si.sell_price_cents * 0.85 / NULLIF(mp.min_price_cents, 0) - 1) * 100, 2
    )                                                          AS profit_pct
FROM items i
JOIN marketcsgo_prices mp ON mp.market_hash_name = i.market_hash_name
JOIN steam_items si        ON si.market_hash_name = i.market_hash_name;
```
