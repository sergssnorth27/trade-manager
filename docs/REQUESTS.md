
# Примеры запросов

## MarketCSGO

### Price list (все активные лоты)

Публичный эндпоинт, авторизация не требуется.

Шаг 1 — получить индекс файлов:

`GET https://market.csgo.com/api/full-export/USD.json`

Доступные валюты: `USD`, `EUR`, `RUB`.

Формат ответа:

```json
{
  "success": true,
  "time": 1724318846,     // unix timestamp генерации выгрузки
  "currency": "USD",
  "format": [...],        // описание полей каждого item (см. ниже)
  "items": [              // список имён chunk-файлов для скачивания
    "17243188645717.json",
    "17243188646993.json"
  ]
}
```

Шаг 2 — скачать каждый chunk:

`GET https://market.csgo.com/api/full-export/{filename}`

Каждый файл — массив лотов. Каждый лот — позиционный массив, порядок полей задан полем `format` из индекса:

```text
[0]  price            — текущая цена в копейках/центах (int)
[1]  id               — уникальный ID лота на market.csgo.com (int)
[2]  market_hash_name — название предмета (Steam market hash name)
[3]  classid          — Steam classid
[4]  instanceid       — Steam instanceid
[5]  real_instance    — Steam instanceid (дубль, обычно совпадает)
[6]  asset            — Steam assetid конкретного предмета
[7]  old_price        — предыдущая цена в копейках/центах (int)
[8]  stamp            — время создания лота "YYYY-MM-DD HH:MM:SS"
[9]  base_id          — ID базового предмета (группировка одинаковых скинов)
[10] float            — float value (износ), string или null
[11] phase            — фаза (для Doppler: "Ruby", "Sapphire", ...) или ""
[12] paintseed        — paint seed, string или null
[13] paintindex       — paint index, string или null
[14] stickers         — ID стикеров через "|", или null
[15] type             — тип предмета ("Rifle", "Pistol", "Knife", ...)
[16] chance_to_transfer — вероятность переноса (int, 0-100)
[17] source           — источник лота (int, платформа)
```

Пример двух лотов:

```json
[
  [623, 5649300794, "Negev | Dazzle (Minimal Wear)", 5075999874, 188530139, 188530139,
   38691476335, 5700, "2024-08-22 12:25:02", 2023, "0.14540666341782", "",
   "391", "610", "11289205989|11289205989|11289205989|11289205989", "Machinegun", 80, 1],

  [7609, 5649300802, "Glock-18 | Neo-Noir (Well-Worn)", 4141779994, 188530139, 188530139,
   38426905739, 69900, "2024-08-22 12:25:02", 82040, "0.40996930003166", "",
   "332", "988", null, "Pistol", 85, 1]
]
```

## STEAM

### Market Search

Публичный эндпоинт, авторизация не требуется.

`GET https://steamcommunity.com/market/search/render/`

Параметры запроса:

- `appid` — ID игры (730 = CS2)
- `norender=1` — вернуть JSON вместо HTML (обязательно)
- `start` — смещение для пагинации (default: 0)
- `count` — количество результатов на страницу (max: 100)
- `query` — поисковая строка (опционально)
- `search_descriptions=1` — искать также в описаниях (опционально)

Пагинация: повторять запросы увеличивая `start` на `count` пока `start < total_count`.

Формат ответа:

```json
{
  "success": true,
  "start": 0,
  "pagesize": 10,
  "total_count": 31350,
  "searchdata": { ... },
  "results": [ ... ]
}
```

`results[]` — один предмет (уникальный скин, не лот):

```json
{
  "name": "Dreams & Nightmares Case",
  "hash_name": "Dreams & Nightmares Case",  // = market_hash_name для других запросов
  "sell_listings": 245097,                   // кол-во активных лотов на продажу
  "sell_price": 223,                         // минимальная цена продажи в центах (int)
  "sell_price_text": "$2.23",               // то же, форматированная строка
  "sale_price_text": "$2.14",               // цена покупателя (после вычета комиссии Steam ~13%)
  "app_icon": "url",
  "app_name": "Counter-Strike 2",
  "asset_description": { ... }
}
```

`asset_description`:

```json
{
  "appid": 730,
  "classid": "4717330486",
  "background_color": "393b3e",
  "icon_url": "string",        // относительный путь, базовый URL: https://community.akamai.steamstatic.com/economy/image/
  "tradable": 1,               // 1 = можно трейдить, 0 = нельзя
  "name": "Dreams & Nightmares Case",
  "name_color": "b0c3d9",      // hex цвет названия (определяет редкость)
  "type": "Base Grade Container",
  "market_name": "Dreams & Nightmares Case",
  "market_hash_name": "Dreams & Nightmares Case",
  "commodity": 1               // 1 = все экземпляры одинаковы (кейсы, патчи), 0 = уникальные (скины с float)
}
```

### Price History

Требует авторизации (cookies активной сессии Steam).

`GET https://steamcommunity.com/market/pricehistory/`

Параметры запроса:

- `appid` — ID игры (730 = CS2)
- `market_hash_name` — URL-encoded название предмета

Формат ответа:

```json
{
  "success": true,
  "price_prefix": "$",   // символ валюты слева
  "price_suffix": "",    // символ валюты справа (например "руб." для RU)
  "prices": [...]
}
```

`prices` — массив записей вида `[timestamp, price, volume]`:

```json
[
  "Aug 22 2025 01: +0",  // [0] timestamp: "MMM DD YYYY HH: +0", UTC
  4.793,                  // [1] средняя цена за период (float)
  "3138"                  // [2] количество продаж за период (string)
]
```

Гранулярность данных зависит от давности записи:

Дневная агрегация (исторические данные, ~старше 30 дней):

- одна запись на календарный день
- час в timestamp всегда "01"
- volume — суммарное количество за сутки (обычно сотни–тысячи)

Пример: ["Aug 22 2025 01: +0", 4.793, "3138"]

Почасовая детализация (свежие данные, ~последние 30 дней):

- до 24 записей на календарный день (по одной на каждый час)
- час в timestamp варьируется от "00" до "23"
- volume — количество продаж за конкретный час (обычно десятки)

Пример: ["Apr 07 2026 14: +0", 3.108, "56"]

Как различать при парсинге:

- Сгруппировать записи по дате (без часа)
- Если на дату одна запись → дневная агрегация
- Если на дату несколько записей → почасовая детализация
- Альтернатива: смотреть разницу между соседними timestamp (~1ч vs ~24ч)
