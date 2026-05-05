
# Примеры запросов

## MarketCSGO

Для получения всех предметов (Price list (all offers))
<https://market.csgo.com/api/full-export/USD.json>
Sample answer:
{
  "success": true,
  "time": 1724318846,
  "currency": "USD",
  "format": [
    "price",
    "id",
    "market_hash_name",
    "classid",
    "instanceid",
    "real_instance",
    "asset",
    "old_price",
    "stamp",
    "base_id",
    "float",
    "phase",
    "paintseed",
    "paintindex",
    "stickers",
    "type",
    "chance_to_transfer",
    "source"
  ],
  "items": [
    "17243188645717.json",
    "17243188646993.json",
    "17243188648538.json",
    "17243188649954.json",
    "17243188651414.json",
    "17243188653044.json",
    "17243188654487.json",
    "17243188655857.json",
    "17243188657132.json"
  ]
}

Потом мы обращаемся к каждому json из items, чтобы получить все предметы
<https://market.csgo.com/api/full-export/17243188645717.json>
[
  [
    623,
    5649300794,
    "Negev | Dazzle (Minimal Wear)",
    5075999874,
    188530139,
    188530139,
    38691476335,
    5700,
    "2024-08-22 12:25:02",
    2023,
    "0.14540666341782",
    "",
    "391",
    "610",
    "11289205989|11289205989|11289205989|11289205989",
    "Machinegun",
    80
  ],
  [
    7609,
    5649300802,
    "Glock-18 | Neo-Noir (Well-Worn)",
    4141779994,
    188530139,
    188530139,
    38426905739,
    69900,
    "2024-08-22 12:25:02",
    82040,
    "0.40996930003166",
    "",
    "332",
    "988",
    null,
    "Pistol",
    85
  ]
]

## STEAM
<https://steamcommunity.com/market/search/render/?appid=730&norender=1&start=0&count=10>

Root объект
{
  "success": true,
  "start": 0,
  "pagesize": 10,
  "total_count": 31350,
  "searchdata": { ... },
  "results": [ ... ]
}

searchdata
{
  "query": "",
  "search_descriptions": false,
  "total_count": 31350,
  "pagesize": 10,
  "prefix": "searchResults",
  "class_prefix": "market"
}

results[] (item)
{
  "name": "Dreams & Nightmares Case",
  "hash_name": "Dreams & Nightmares Case",
  "sell_listings": 245097,
  "sell_price": 223,
  "sell_price_text": "$2.23",
  "sale_price_text": "$2.14",
  "app_icon": "url",
  "app_name": "Counter-Strike 2",
  "asset_description": { ... }
}

🧩 asset_description
{
  "appid": 730,
  "classid": "4717330486",
  "background_color": "393b3e",
  "icon_url": "string",
  "tradable": 1,
  "name": "Dreams & Nightmares Case",
  "name_color": "b0c3d9",
  "type": "Base Grade Container",
  "market_name": "Dreams & Nightmares Case",
  "market_hash_name": "Dreams & Nightmares Case",
  "commodity": 1,
  "market_bucket_group_name": "string",
  "market_bucket_group_id": "string"
}
