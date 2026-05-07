# Database Schema

## Ключевой принцип матчинга

`market_hash_name` — единственный общий ключ между MarketCSGO и Steam.
Все таблицы связаны через него.

```
marketcsgo_listings ──┐
                       ├── items (market_hash_name) ── steam_items
steam_price_history ──┘                              └── steam_price_history
```

---

## Таблицы

### `items` — справочник предметов

Одна строка на уникальный скин. Заполняется из Steam Market Search.
Служит якорем для FK всех остальных таблиц.

```sql
CREATE TABLE items (
    market_hash_name TEXT        PRIMARY KEY,
    classid          TEXT        NOT NULL,
    type             TEXT,                        -- "Rifle", "Pistol", "Knife", ...
    commodity        BOOLEAN     NOT NULL DEFAULT false, -- true = кейсы/патчи (все одинаковые)
    tradable         BOOLEAN     NOT NULL DEFAULT true,
    icon_url         TEXT,
    name_color       TEXT,                        -- hex, определяет редкость
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### `marketcsgo_listings` — активные лоты MarketCSGO

Один лот = одна строка. Полная замена при каждом синке full-export.
Содержит индивидуальные характеристики каждого предмета (float, стикеры).

```sql
CREATE TABLE marketcsgo_listings (
    id                 BIGINT      PRIMARY KEY,   -- ID лота на market.csgo.com [поле 1]
    market_hash_name   TEXT        NOT NULL REFERENCES items(market_hash_name),
    price_cents        INT         NOT NULL,      -- текущая цена в центах [поле 0]
    old_price_cents    INT,                       -- предыдущая цена [поле 7]
    asset_id           BIGINT      NOT NULL,      -- Steam assetid [поле 6]
    classid            BIGINT      NOT NULL,      -- [поле 3]
    instanceid         BIGINT      NOT NULL,      -- [поле 4]
    base_id            BIGINT,                    -- группировка одинаковых скинов [поле 9]
    float_value        NUMERIC(18,15),            -- износ [поле 10]
    phase              TEXT,                      -- для Doppler: "Ruby", "Sapphire", ... [поле 11]
    paintseed          INT,                       -- [поле 12]
    paintindex         INT,                       -- [поле 13]
    stickers           TEXT,                      -- "id|id|id" или NULL [поле 14]
    chance_to_transfer INT,                       -- вероятность переноса 0-100 [поле 16]
    source             INT,                       -- платформа-источник [поле 17]
    listed_at          TIMESTAMPTZ,               -- время создания лота [поле 8]
    synced_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- для быстрого поиска минимальной цены по предмету
CREATE INDEX ON marketcsgo_listings (market_hash_name, price_cents);
```

---

### `steam_items` — снапшот Steam Market Search

Одна строка на предмет. Обновляется раз в 15 минут.
Даёт минимальную цену продажи и количество лотов.

```sql
CREATE TABLE steam_items (
    market_hash_name TEXT        PRIMARY KEY REFERENCES items(market_hash_name),
    sell_listings    INT         NOT NULL DEFAULT 0,  -- кол-во активных лотов
    sell_price_cents INT         NOT NULL DEFAULT 0,  -- минимальная цена продажи в центах
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### `steam_price_history` — история цен Steam

Time-series. Данные из `/market/pricehistory/`.
Хранит и дневные агрегаты, и почасовые записи — различаются полем `granularity`.

```sql
CREATE TABLE steam_price_history (
    market_hash_name TEXT        NOT NULL REFERENCES items(market_hash_name),
    ts               TIMESTAMPTZ NOT NULL,           -- UTC
    granularity      TEXT        NOT NULL CHECK (granularity IN ('daily', 'hourly')),
    avg_price        NUMERIC(10,4) NOT NULL,          -- средняя цена в долларах
    volume           INT         NOT NULL,            -- кол-во продаж за период
    PRIMARY KEY (market_hash_name, ts)
);

CREATE INDEX ON steam_price_history (market_hash_name, ts DESC);
```

---

### `steam_session` — сессия авторизации Steam

Всегда одна строка (id = 1).
Хранит refresh token для восстановления сессии без повторного ввода credentials.

```sql
CREATE TABLE steam_session (
    id                  INT  PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    login               TEXT NOT NULL,
    steam_refresh_token TEXT NOT NULL,   -- значение steamRefresh_steam cookie (~200 дней)
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## VIEW для арбитража

Сводит данные двух платформ. Арбитражный движок читает отсюда.

```sql
CREATE VIEW arbitrage_view AS
SELECT
    i.market_hash_name,
    i.type,
    i.commodity,
    MIN(ml.price_cents)                                             AS buy_price_cents,   -- мин. цена покупки на MarketCSGO
    si.sell_price_cents                                             AS steam_sell_cents,  -- мин. цена продажи в Steam
    si.sell_listings,
    ROUND(si.sell_price_cents * 0.85)                              AS net_cents,         -- получим после комиссии Steam 15%
    ROUND(
        (si.sell_price_cents * 0.85 / NULLIF(MIN(ml.price_cents), 0) - 1) * 100, 2
    )                                                              AS profit_pct
FROM items i
JOIN marketcsgo_listings ml ON ml.market_hash_name = i.market_hash_name
JOIN steam_items si          ON si.market_hash_name = i.market_hash_name
GROUP BY
    i.market_hash_name,
    i.type,
    i.commodity,
    si.sell_price_cents,
    si.sell_listings;
```

---

## Вопросы для обсуждения

1. **`marketcsgo_listings`** — хранить все лоты или только минимальную цену на предмет?
   Все лоты нужны если хотим выбирать по float/стикерам. Если только арбитраж — достаточно `marketcsgo_prices (market_hash_name, min_price_cents)`.

2. **`steam_price_history`** — нужна ли вся история с Aug 2025 или только последние N дней?
   Полная история (~240 строк/предмет дневных + ~720 почасовых) при 5000 отслеживаемых предметов = ~5M строк. Нормально для PostgreSQL, но стоит обсудить retention policy.

3. **`items`** — заполнять сразу из MarketCSGO или ждать Steam Market Search?
   MarketCSGO даёт `market_hash_name` и `type`, но не даёт `classid`, `icon_url`, `commodity`. Эти поля приходят только из Steam.
