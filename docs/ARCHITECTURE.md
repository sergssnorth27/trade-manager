# Trade Manager v1 — Архитектура

## Цель системы

Арбитраж CS2-предметов: покупка на MarketCSGO, продажа на Steam Community Market.
USD - единственная поддерживаемая валюта
Главное конкурентное преимущество — **скорость получения минимальной цены** с MarketCSGO.

---

## Схема системы

```
┌─────────────────────────────────────────────────────────────┐
│                        trade-manager                        │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  MarketCSGO  │    │    Steam     │    │  Arbitrage   │  │
│  │  Collector   │    │   Scraper    │    │   Engine     │  │
│  │              │    │              │    │              │  │
│  │  WebSocket   │    │  HTTP+Proxy  │    │  Price diff  │  │
│  │  (real-time) │    │  (periodic)  │    │  Profit calc │  │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘  │
│         │                   │                   │           │
│         └───────────────────┼───────────────────┘           │
│                             │                               │
│                    ┌────────▼────────┐                      │
│                    │   PostgreSQL    │                      │
│                    │    (store)      │                      │
│                    └────────┬────────┘                      │
│                             │                               │
│                    ┌────────▼────────┐                      │
│                    │  Buyer Module   │                      │
│                    │ (MarketCSGO API)│                      │
│                    └─────────────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

---

## Модули

### 1. MarketCSGO Collector

**Задача:** максимально быстро получать актуальную минимальную цену.

**Подход:**

- WebSocket через Centrifuge (`wss://wsprice.csgo.com/connection/websocket`)
- Channel: `public:items:730:usd`
- Bootstrap: HTTP full-export при старте + ресинк каждые 2 минуты
- In-memory min-heap для мгновенного доступа к минимальной цене

**Данные:**

- `name_id` — числовой ID предмета на MarketCSGO
- `hash_name` — текстовое название (для матчинга со Steam)
- `min_price_milli` — минимальная цена покупки в миллидолларах

---

### 2. Steam Scraper

**Задача:** собирать данные о ценах, объёме и ликвидности предметов.

**Подход:**

- HTTP с ротацией прокси (защита от 429)
- Периодическое обновление (не нужен real-time)
- История цен — раз в час, минимальная цена — раз в 15 минут

**Ключевые endpoints Steam Market:**

```text
# Список всех предметов (пагинация)
GET /market/search/render/?appid=730&norender=1&start=0&count=100

# История цен (нужен логин/cookie)
GET /market/pricehistory/?appid=730&market_hash_name={name}

# Стакан ордеров (нужен item_nameid из HTML страницы предмета)
GET /market/itemordershistogram?country=US&language=english&currency=1&item_nameid={id}

# Минимальная цена продажи (текущий листинг)
GET /market/listings/730/{name}/render?start=0&count=3&country=US&language=english&currency=1
```

**Данные:**

- Минимальная цена продажи (sell listings)
- Средняя цена за 24ч / 7д / 30д
- Объём торгов за 24ч / 7д / 30д
- Buy orders / Sell orders (стакан)

**Прокси-пул:**

- Список прокси с per-proxy rate limiter
- Rate: ~1 req/3s на прокси без авторизации, больше с cookie
- Retry с другим прокси при 429
- Healthcheck — исключение мёртвых прокси

---

### 3. Arbitrage Engine

**Задача:** найти предметы с прибылью >= 30%.

**Формула:**

```text
Покупаем на MarketCSGO:  buy_price
Продаём в Steam:         steam_listing_price

Steam fee (CS2): 15% (13% Steam + 2% Valve)
Получаем в Steam:        steam_listing_price * 0.85

profit_ratio = (steam_listing_price * 0.85 / buy_price) - 1
```

**Условия отбора:**

```text
1. profit_ratio >= 0.30          — минимум 30% прибыли
2. steam_avg_30d >= target_sell  — средняя цена за 30д подтверждает цену продажи
3. steam_volume_30d >= MIN_VOLUME — минимальная ликвидность (например, 10 продаж/мес)
4. marketcsgo_min_price > 0      — предмет доступен на MarketCSGO
```

**Почему steam_avg_30d важна:**
Минимальная цена в Steam может быть единственным выставленным лотом по завышенной цене.
Средняя за 30 дней показывает реальную цену, по которой рынок торгует.

---

### 4. Buyer Module

**Задача:** автоматически покупать выгодные предметы с MarketCSGO.

**Логика:**

- Получает список предметов от Arbitrage Engine
- Проверяет баланс на MarketCSGO
- Проверяет лимиты (макс. сумма за покупку, макс. кол-во предметов)
- Выполняет покупку через MarketCSGO API
- Сохраняет данные о покупке в store

**Данные о покупке:**

```text
- item_name (hash_name)
- buy_price_milli
- quantity
- target_sell_price_milli (рекомендованная цена продажи)
- expected_profit_ratio
- bought_at (timestamp)
- status (pending / sold / cancelled)
```

---

### 5. Store (PostgreSQL)

**Таблицы:**

```sql
-- Справочник предметов (матчинг между платформами)
item_dictionary (name_id, hash_name)

-- Актуальные цены MarketCSGO
marketcsgo_prices (name_id, min_price_milli, updated_at)

-- Данные Steam по предмету
steam_items (hash_name, min_sell_milli, avg_24h_milli, avg_7d_milli,
             avg_30d_milli, volume_24h, volume_7d, volume_30d, updated_at)

-- Стакан ордеров Steam
steam_orders (hash_name, order_type, price_milli, quantity, updated_at)

-- История покупок
purchases (id, hash_name, buy_price_milli, quantity,
           target_sell_price_milli, expected_profit_ratio, bought_at, status)
```

---

## Комиссии

| Платформа      | Комиссия | Считается                        |
| -------------- | -------- | -------------------------------- |
| MarketCSGO     | ~5%      | уже включена в цену листинга     |
| Steam (CS2)    | 15%      | вычитается при продаже           |

**Пример расчёта:**

- Купили на MarketCSGO: $1.00
- Цена листинга в Steam: $1.53
- Получим после комиссии Steam: $1.53 × 0.85 = $1.30
- Прибыль: 30%

---

## Технологический стек

| Компонент        | Технология                  | Причина                              |
| ---------------- | --------------------------- | ------------------------------------ |
| Язык             | Go                          | производительность, горутины         |
| БД               | PostgreSQL                  | надёжность, сложные JOIN-запросы     |
| MarketCSGO WS    | centrifugal/centrifuge-go   | проверено в боевых условиях          |
| HTTP клиент      | net/http + proxy            | стандартная библиотека               |
| Миграции         | golang-migrate              | простота                             |
| Конфиг           | godotenv + env vars         | привычный подход                     |
| Логирование      | slog (stdlib Go 1.21+)      | встроен, отдельный пакет не нужен    |
| Docker           | Docker Compose + Portainer  | контейнеризация + веб-мониторинг     |

---

## Docker-окружение

```yaml
# docker-compose.yml
services:
  trade-manager:  # основное приложение
  postgres:       # база данных
  portainer:      # веб-интерфейс для управления контейнерами
```

**Portainer** — веб-панель для управления Docker-контейнерами.
Доступна на `http://localhost:9000`. Показывает логи, статус контейнеров, потребление ресурсов.

---

## Структура проекта

```text
trade-manager-v1/
├── cmd/
│   └── trade-manager/
│       └── main.go               # точка входа + wiring всех компонентов
├── internal/
│   ├── domain/                   # общие типы, которые нужны нескольким пакетам
│   │   └── item.go               # Item, Price, Opportunity
│   ├── config/                   # конфигурация из env-переменных
│   │   └── config.go
│   ├── marketcsgo/               # всё про MarketCSGO
│   │   ├── ws.go                 # WebSocket клиент (Centrifuge)
│   │   ├── bootstrap.go          # загрузка full-export + ресинк
│   │   └── tracker.go            # in-memory min-heap трекер цен
│   ├── steam/                    # всё про Steam
│   │   ├── client.go             # HTTP клиент с поддержкой прокси
│   │   ├── proxy.go              # пул прокси + per-proxy rate limiter
│   │   └── scraper.go            # парсинг ответов Steam API
│   ├── arbitrage/                # бизнес-логика поиска арбитража
│   │   └── analyzer.go           # сравнение цен, фильтрация, отбор
│   ├── buyer/                    # покупка предметов на MarketCSGO
│   │   └── buyer.go
│   └── store/                    # работа с PostgreSQL
│       ├── postgres.go           # подключение, миграции
│       └── queries.go            # SQL-запросы
├── migrations/
│   ├── 001_init.sql
│   └── 002_purchases.sql
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/
│   ├── ARCHITECTURE.md           # этот файл
│   └── IMPLEMENTATION_PLAN.md
├── .env.example
├── go.mod
└── go.sum
```

**Принципы структуры:**

- `domain` — единственный пакет без зависимостей на другие внутренние пакеты. Только типы.
- Каждый пакет отвечает за одну область. Типы живут рядом с кодом, который их использует.
- Логирование через `slog` из stdlib — отдельный пакет не нужен.
- `store` — единая точка доступа к БД для всех модулей.
