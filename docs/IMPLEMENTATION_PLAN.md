# Trade Manager v1 — План реализации

## Принцип работы

Разрабатываем итеративно, модуль за модулем.
Каждый этап даёт рабочий результат, который можно запустить и проверить.

---

## Этап 1 — Foundation (фундамент)

**Цель:** Скелет проекта, который компилируется и подключается к БД.

- [ ] Инициализация Go-модуля (`go mod init`)
- [ ] Структура папок (см. ARCHITECTURE.md)
- [ ] `internal/config/config.go` — чтение env-переменных
- [ ] `internal/store/postgres.go` — подключение к PostgreSQL (pgx/v5)
- [ ] `migrations/001_init.sql` — начальная схема БД
- [ ] `cmd/trade-manager/main.go` — точка входа, инициализация
- [ ] `.env.example` — шаблон конфигурации

**Зависимости:**

```text
github.com/jackc/pgx/v5
github.com/joho/godotenv
```

**Результат:** `go run ./cmd/trade-manager` стартует, логирует через slog, подключается к PostgreSQL.

---

## Этап 2 — Steam Scraper ⬅️ НАЧИНАЕМ ЗДЕСЬ

**Цель:** Собирать данные о предметах CS2 со Steam Market.

### 2.1 Прокси-пул

**Файл:** `internal/steam/proxy.go`

- Структура `ProxyPool` со списком прокси
- Per-proxy `rate.Limiter` (golang.org/x/time/rate)
- Метод `Acquire() *Proxy` — получить свободный прокси
- Метод `MarkDead(proxy)` — убрать нерабочий прокси
- Retry логика при 429

**Зависимости:**

```text
golang.org/x/time
```

### 2.2 HTTP клиент

**Файл:** `internal/steam/client.go`

- `Client` с `http.Client`, настроенным на прокси
- Метод `Do(req)` с ротацией прокси и retry при 429
- Заголовки как у браузера (User-Agent, Accept-Language и т.д.)
- Опциональный Cookie jar для авторизованных запросов

### 2.3 Steam API — список предметов

**Файл:** `internal/steam/scraper.go`

Endpoint:

```text
GET /market/search/render/?appid=730&norender=1&start={start}&count=100
```

Что парсим:

```json
{
  "total_count": 15000,
  "results": [
    {
      "name": "AK-47 | Redline (Field-Tested)",
      "sell_listings": 1234,
      "sell_price": 500,
      "sell_price_text": "$5.00"
    }
  ]
}
```

Стратегия:

- Пагинация: start=0, 100, 200, ...
- Ограничить скорость прокси-пулом
- Сохранять `hash_name` и `sell_price` в store

### 2.4 Steam API — история цен

**Файл:** `internal/steam/scraper.go`

Endpoint:

```text
GET /market/pricehistory/?appid=730&market_hash_name={name}
```

⚠️ Требует авторизации (Steam cookie: `steamLoginSecure`)

Формат ответа:

```json
{
  "prices": [
    ["Jun 01 2025 01: +0", "1.50", "100"]
  ]
}
```

Каждый элемент: `[дата, цена, объём]`

Что считаем из истории:

- `avg_24h`, `avg_7d`, `avg_30d` — средние цены
- `volume_24h`, `volume_7d`, `volume_30d` — объёмы

### 2.5 Steam API — стакан ордеров

**Файл:** `internal/steam/scraper.go`

Требует `item_nameid` (число из HTML страницы предмета).

Шаг 1 — получить nameid из HTML:

```text
GET /market/listings/730/{hash_name}
```

Ищем в HTML: `Market_LoadOrderSpread( 12345678 )`

Шаг 2 — получить стакан:

```text
GET /market/itemordershistogram?country=US&language=english&currency=1&item_nameid={id}
```

Ответ содержит `buy_order_graph` и `sell_order_graph` — массивы `[цена, количество, описание]`.

### 2.6 Хранение в store

**Файл:** `internal/store/queries.go`

Таблицы (добавить в миграции):

```sql
CREATE TABLE steam_items (
    hash_name      TEXT PRIMARY KEY,
    min_sell_milli BIGINT,
    avg_24h_milli  BIGINT,
    avg_7d_milli   BIGINT,
    avg_30d_milli  BIGINT,
    volume_24h     INT,
    volume_7d      INT,
    volume_30d     INT,
    updated_at     TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE steam_item_nameid (
    hash_name    TEXT PRIMARY KEY,
    steam_nameid BIGINT NOT NULL,
    fetched_at   TIMESTAMPTZ DEFAULT now()
);
```

**Результат этапа:** Запускаем — через N часов store заполнен данными по всем CS2 предметам.

---

## Этап 3 — MarketCSGO Collector

**Цель:** Получать real-time минимальные цены с MarketCSGO.

- [ ] Перенести и адаптировать код из `price-checker`
  - `ws.go` — WebSocket клиент (Centrifuge)
  - `bootstrap.go` — загрузка full-export + ресинк
  - `tracker.go` — in-memory min-heap
- [ ] Адаптировать схему БД (добавить `marketcsgo_prices`)
- [ ] Запустить параллельно со Steam scraper

**Таблицы:**

```sql
CREATE TABLE marketcsgo_prices (
    name_id         BIGINT PRIMARY KEY,
    min_price_milli BIGINT NOT NULL,
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE item_dictionary (
    name_id   BIGINT PRIMARY KEY,
    hash_name TEXT NOT NULL UNIQUE
);
```

**Результат:** В store обновляются цены MarketCSGO в real-time.

---

## Этап 4 — Arbitrage Engine

**Цель:** Найти выгодные предметы для покупки.

- [ ] SQL-запрос для JOIN по hash_name (MarketCSGO ↔ Steam)
- [ ] Расчёт profit_ratio
- [ ] Применение фильтров (минимальный профит, ликвидность)
- [ ] Периодический запуск (например, каждые 30 секунд)
- [ ] Логирование найденных возможностей

**Пример SQL:**

```sql
SELECT
    d.name_id,
    d.hash_name,
    mp.min_price_milli                                         AS buy_price,
    si.min_sell_milli                                          AS steam_min_sell,
    si.avg_30d_milli                                           AS steam_avg_30d,
    si.volume_30d,
    ROUND(
        (si.min_sell_milli * 0.85 / mp.min_price_milli::float - 1) * 100, 2
    )                                                          AS profit_pct
FROM item_dictionary d
JOIN marketcsgo_prices mp ON mp.name_id   = d.name_id
JOIN steam_items       si ON si.hash_name = d.hash_name
WHERE
    mp.min_price_milli > 0
    AND si.min_sell_milli > 0
    AND si.avg_30d_milli >= si.min_sell_milli * 0.85
    AND si.volume_30d >= 10
    AND (si.min_sell_milli * 0.85 / mp.min_price_milli::float - 1) >= 0.30
ORDER BY profit_pct DESC;
```

**Результат:** Список предметов с профитом >= 30% выводится в лог / store.

---

## Этап 5 — Buyer Module

**Цель:** Автоматически покупать найденные предметы.

- [ ] MarketCSGO API — эндпоинт покупки
- [ ] Проверка баланса перед покупкой
- [ ] Лимиты безопасности (макс. сумма/день, макс. на предмет)
- [ ] Сохранение данных о покупке в store
- [ ] Dry-run режим (логировать, но не покупать)

**Таблица:**

```sql
CREATE TABLE purchases (
    id                      SERIAL PRIMARY KEY,
    hash_name               TEXT NOT NULL,
    buy_price_milli         BIGINT NOT NULL,
    quantity                INT NOT NULL DEFAULT 1,
    target_sell_price_milli BIGINT NOT NULL,
    expected_profit_pct     NUMERIC(5, 2),
    bought_at               TIMESTAMPTZ DEFAULT now(),
    status                  TEXT DEFAULT 'pending'
);
```

**Результат:** Бот покупает предметы, данные сохранены для ручной продажи в Steam.

---

## Этап 6 — Docker

**Цель:** Контейнеризация + веб-мониторинг.

- [ ] `docker/Dockerfile` — multi-stage build
- [ ] `docker/docker-compose.yml` с сервисами: `trade-manager`, `postgres`, `portainer`
- [ ] Volumes для персистентности данных
- [ ] Health checks для сервисов
- [ ] Конфигурация через `env_file`

**Portainer** доступен на `http://your-server:9000`.
Показывает логи контейнеров, статус, CPU/RAM, возможность перезапуска.

**Результат:** `docker compose up -d` — всё запущено, мониторинг доступен в браузере.

---

## Порядок приоритетов

```text
[1] Foundation         — скелет и store
[2] Steam Scraper      — НАЧИНАЕМ
[3] MarketCSGO WS      — перенос из price-checker
[4] Arbitrage Engine   — JOIN + фильтры
[5] Buyer Module       — покупки
[6] Docker             — деплой
```

---

## Ключевые решения

| Вопрос                | Решение                        | Причина                                       |
| --------------------- | ------------------------------ | --------------------------------------------- |
| MarketCSGO обновления | WebSocket (Centrifuge)         | уже работает в price-checker                  |
| Steam rate limit      | Proxy pool + per-proxy limiter | защита от 429                                 |
| Хранение              | PostgreSQL                     | единое место для всех данных и JOIN           |
| Архитектура           | Монолит с горутинами           | проще для одного разработчика                 |
| Покупка               | Только MarketCSGO API          | Steam покупка сложнее и требует иного подхода |

---

## Переменные окружения (.env.example)

```env
# MarketCSGO
MARKETCSGO_KEY=your_api_key_here

# Steam
STEAM_COOKIE=steamLoginSecure=your_cookie_here
PROXY_LIST=http://user:pass@host:port,http://...

# PostgreSQL
PG_DSN=postgres://user:password@localhost:5432/trade_manager

# Analytics
MIN_PROFIT_PCT=30
MIN_VOLUME_30D=10

# Buyer
MAX_BUY_AMOUNT_USD=100
DRY_RUN=true

# Logging
LOG_LEVEL=info
```