# Sarbon API — полная документация на русском языке

Руководство по использованию API Sarbon: авторизация, создание грузов, справочники, описание всех полей и примеры запросов.

---

## 1. Общие сведения

### Базовый URL и заголовки

Все запросы к API (кроме `/health`) должны содержать заголовки:

| Заголовок | Описание | Пример |
|-----------|----------|--------|
| **X-Device-Type** | Тип клиента | `web`, `ios`, `android` |
| **X-Language** | Язык | `ru`, `uz`, `en`, `tr`, `zh` |
| **X-Client-Token** | Токен приложения (если настроен на сервере) | строка из конфига |

Для защищённых методов дополнительно передаётся:

| Заголовок | Описание |
|-----------|----------|
| **X-User-Token** | JWT access_token после входа (водитель, диспетчер или админ) |

### Формат ответов

Успешный ответ: `{ "status": "ok", "data": { ... } }` или `{ "status": "ok", "data": { "id": "uuid" } }`.

Ошибка: `{ "status": "error", "code": 400, "description": "текст ошибки", "data": null }`.

---

## 2. Справочники (Reference)

Перед созданием грузов и работой с формами используйте справочники — оттуда берутся допустимые значения.

### Города (типы кузова и др.)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/reference/cities | **Все города мира.** Коды городов (TAS — Ташкент, SAM — Самарканд, DXB — Дубай и т.д.). Query: `country_code` (UZ, AE, RU…) — фильтр по стране. В ответе: `data.items` — массив `{ id, code, name_ru, name_en, country_code, lat, lng }`. |
| GET | /v1/reference/cargo | **Справочник по грузам.** Статусы груза (cargo_status), типы точек маршрута (route_point_type), статусы оффера (offer_status), **типы кузова (truck_type)**: REFRIGERATOR, TENT, FLATBED, TANKER, OTHER, статусы рейса (trip_status). Все значения в верхнем регистре; в запросах можно передавать в нижнем (tent, refrigerator). |

---

## 3. Создание груза (POST /api/cargo)

Используется диспетчером или админом. В заголовке передаётся **X-User-Token** (JWT) — тогда создатель записывается автоматически. Либо в теле можно передать **company_id** — груз будет от имени компании.

### Обязательные поля тела запроса

| Поле | Тип | Описание |
|------|-----|----------|
| **weight** | number | Вес груза в **тоннах**. Должно быть больше 0. |
| **volume** | number | Объём груза в **кубических метрах (м³)**. Должно быть больше 0. |
| **truck_type** | string | Тип кузова. Значения из справочника GET /v1/reference/cargo → truck_type: `tent`, `refrigerator`, `flatbed`, `tanker`, `other`. |
| **route_points** | array | Массив точек маршрута. **Минимум одна точка с type=load (погрузка) и одна с type=unload (выгрузка).** |

### Обязательные поля каждой точки маршрута (route_points)

| Поле | Тип | Описание |
|------|-----|----------|
| **type** | string | Тип точки: `load` (погрузка), `unload` (выгрузка), `customs` (таможня), `transit` (транзит). |
| **city_code** | string | Код города из GET /v1/reference/cities (например TAS, SAM, DXB). |
| **address** | string | Полный адрес: улица, дом, склад, промзона. |
| **lat** | number | Широта в градусах (для карты). |
| **lng** | number | Долгота в градусах (для карты). |
| **point_order** | integer | Порядок точки в маршруте: 1, 2, 3… |

### Необязательные поля точки маршрута

| Поле | Описание |
|------|----------|
| **orientir** | Ориентир для водителя: как найти место (въезд, КПП, рядом с чем). |
| **comment** | Произвольный комментарий к точке. |
| **is_main_load** | true — основная точка погрузки. |
| **is_main_unload** | true — основная точка выгрузки. |

### Необязательные поля груза

| Поле | Описание |
|------|----------|
| **ready_enabled** | true — указана дата готовности груза. |
| **ready_at** | Дата и время готовности (обязательна при ready_enabled=true). |
| **load_comment** | Комментарий к погрузке. |
| **temp_min**, **temp_max** | Температурный режим (°C); допустимо только при truck_type=refrigerator. |
| **adr_enabled** | true — опасный груз (ADR). |
| **adr_class** | Класс опасного груза (обязателен при adr_enabled=true). |
| **loading_types** | Массив способов погрузки. |
| **requirements** | Массив требований к перевозке. |
| **shipment_type** | FTL, LTL и т.д. |
| **belts_count** | Количество ремней крепления. |
| **documents** | Объект с флагами TIR, T1, CMR, Medbook, GLONASS, Seal, Permit (boolean). |
| **contact_name** | Имя контактного лица. |
| **contact_phone** | Телефон контакта (+998XXXXXXXXX). |
| **company_id** | UUID компании, от которой груз (опционально). |
| **payment** | Блок условий оплаты (см. ниже). |

При создании груза статус автоматически устанавливается в `created`. Чтобы перевести груз в поиск перевозчика, после создания вызовите PATCH /api/cargo/:id/status с телом `{ "status": "searching" }`.

### Блок оплаты (payment)

Передаётся при необходимости. Один блок на груз.

| Поле | Описание |
|------|----------|
| **is_negotiable** | true — цена договорная. |
| **price_request** | true — запрос цены (без фиксированной суммы); перевозчик предлагает сам. Если false и блок payment передан — нужно указать **total_amount**. |
| **total_amount** | Общая сумма оплаты. |
| **total_currency** | Валюта: USD, UZS и т.д. |
| **with_prepayment** | Есть предоплата. |
| **without_prepayment** | Оплата без предоплаты. |
| **prepayment_amount**, **prepayment_currency** | Сумма и валюта предоплаты. |
| **remaining_amount**, **remaining_currency** | Остаток к оплате. |

### Пример тела запроса (создание груза)

```json
{
  "weight": 20,
  "volume": 40,
  "truck_type": "tent",
  "route_points": [
    {
      "type": "load",
      "city_code": "TAS",
      "address": "ул. Амира Темура 1, склад №5",
      "orientir": "въезд со стороны метро Хамза",
      "lat": 41.311081,
      "lng": 69.240562,
      "point_order": 1,
      "is_main_load": true,
      "is_main_unload": false
    },
    {
      "type": "unload",
      "city_code": "SAM",
      "address": "промзона, терминал А",
      "orientir": "за КПП 2",
      "lat": 39.654167,
      "lng": 66.959722,
      "point_order": 2,
      "is_main_load": false,
      "is_main_unload": true
    }
  ],
  "payment": {
    "is_negotiable": false,
    "price_request": false,
    "total_amount": 1500,
    "total_currency": "USD",
    "with_prepayment": true,
    "without_prepayment": false,
    "prepayment_amount": 500,
    "prepayment_currency": "USD"
  },
  "contact_name": "Иван Петров",
  "contact_phone": "+998901234567"
}
```

Ответ: `{ "status": "ok", "data": { "id": "uuid-созданного-груза" } }`.

---

## 4. Авторизация фриланс-диспетчера

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /v1/dispatchers/auth/phone | Отправить OTP на телефон (тело: `{ "phone": "+998901234567" }`). |
| POST | /v1/dispatchers/auth/otp/verify | Подтвердить OTP (тело: phone, otp). В ответе — tokens при status=login или session_id при status=register. |
| POST | /v1/dispatchers/auth/login/password | Вход по паролю (тело: phone, password). В ответе — access_token, refresh_token, expires_at, refresh_expires_at (Unix мс). |
| POST | /v1/dispatchers/auth/refresh | Обновить пару токенов (тело: `{ "refresh_token": "..." }`). |
| POST | /v1/dispatchers/auth/logout | Выход. Тело: `{ "refresh_token": "..." }` — отзыв одной сессии, или `{ "access_token": "..." }` — отзыв всех сессий диспетчера. При невалидном токене — 401. |

---

## 5. Авторизация водителя

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /v1/auth/phone | Отправить OTP (тело: phone). |
| POST | /v1/auth/otp/verify | Подтвердить OTP → tokens или session_id при регистрации. |
| POST | /v1/auth/refresh | Обновить токены (тело: refresh_token). |
| POST | /v1/auth/logout | Выход (тело: refresh_token или access_token). При невалидном токене — 401. |

---

## 6. Грузы: список, карточка, статус, офферы

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /api/cargo | Список грузов. Параметры: status, weight_min, weight_max, truck_type, page, limit, sort. |
| GET | /api/cargo/:id | Карточка груза с route_points и payment. |
| PUT | /api/cargo/:id | Обновить груз (до перехода в assigned действуют ограничения). |
| PATCH | /api/cargo/:id/status | Сменить статус (тело: `{ "status": "searching" }` и т.д.). |
| GET | /api/cargo/:id/offers | Список офферов по грузу. |
| POST | /api/cargo/:id/offers | Создать оффер (тело: carrier_id, price, currency, comment). |
| POST | /api/offers/:id/accept | Принять оффер → груз переходит в assigned, создаётся рейс (trip). |

---

## 7. Рейсы (trips)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /api/trips?cargo_id= | Рейс по грузу. |
| GET | /api/trips/:id | Рейс по ID. |
| PATCH | /v1/dispatchers/trips/:id/assign-driver | Назначить водителя на рейс (тело: driver_id). Требуется X-User-Token диспетчера. |
| POST | /v1/trips/:id/confirm | Водитель подтверждает назначение. |
| POST | /v1/trips/:id/reject | Водитель отклоняет назначение. |
| PATCH | /v1/trips/:id/status | Водитель меняет статус рейса (loading, en_route, unloading, completed, cancelled). |

---

Дополнительные разделы (водители, регистрация, KYC, профиль, админ, компании, чат) описаны в **Swagger UI** (`/docs`) и в файле **docs/openapi.yaml** — там же приведены все теги, схемы и русские пояснения к полям.
