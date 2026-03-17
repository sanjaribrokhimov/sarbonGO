## Cargo: итоговый отчёт по доработкам

### 1. Что реализовано в этом цикле

- **Модель Cargo и репозиторий**
  - Приведена структура к ТЗ:
    - Добавлены поля: `Name`, `Packaging`, `Dimensions`, `Photos`, `CapacityRequired`, `PlaceID`, `CargoTypeID`.
    - Строгая типизация: `ShipmentType` (FTL/LTL/…), `CargoStatus` (CREATED, PENDING_MODERATION, SEARCHING_ALL, SEARCHING_COMPANY, REJECTED, ASSIGNED, IN_PROGRESS, IN_TRANSIT, DELIVERED, COMPLETED, CANCELLED).
    - Обновлены структуры `RoutePointInput` и `Documents` (TIR, T1, CMR, Medbook, GLONASS, Seal, Permit).
  - Валидация на уровне `CreateParams`:
    - `Weight`, `CapacityRequired` — `required, gt=0`.
    - `RoutePoints` — `min=2, dive`.
    - `ADRClass` обязательна при `ADREnabled=true`.
    - Координаты и адрес — через `required_with`.

- **Handlers Cargo (HTTP API)**
  - `CreateCargoReq` / `UpdateCargoReq` синхронизированы с моделью:
    - Есть поля `packaging`, `dimensions`, `photos`, `capacity_required`, `cargo_type_id`, `place_id` в `route_points` и т.д.
    - Обязательные поля помечены `binding:"required"` (вес, грузоподъёмность, тип кузова, маршрут, ключевые координаты).
  - Дополнительная бизнес‑валидация:
    - Проверки по справочникам: `truck_type`, `shipment_type`, `loading_types`, валюты, `prepayment_type`, `remaining_type`.
    - Проверяется минимум одна точка `LOAD` и одна `UNLOAD`.
    - `temp_min/temp_max` разрешены только при `REFRIGERATOR`.
    - `ready_enabled=true` → обязателен `ready_at`.
  - Маппинги `toCreateParams` / `toUpdateParams`:
    - Переводят строковые enum‑поля в UPPERCASE и строгие типы (`ShipmentType`, `CargoStatus`).
    - Прокидывают новые поля (упаковка, габариты, фото, грузоподъёмность, place_id, тип груза) в слой репозитория.

- **Статусы и переходы**
  - В `internal/cargo/model.go` и `internal/cargo/repo.go`:
    - Описан полный набор статусов груза в UPPERCASE.
    - Функция `IsSearching` учитывает `SEARCHING_ALL` и `SEARCHING_COMPANY`.
    - `SetStatus` ограничивает переходы допустимыми (created/pending_moderation → searching_* → assigned → in_progress → completed и др.).

### 2. Swagger / OpenAPI для Cargo

- **Схемы запросов/ответов**
  - `CargoCreateRequest` содержит все поля из `CreateCargoReq`:
    - Груз: `name`, `weight`, `volume`, `packaging`, `dimensions`, `photos`.
    - Готовность: `ready_enabled`, `ready_at`, `load_comment`.
    - Транспорт: `truck_type`, `capacity_required`, `temp_min`, `temp_max`, `adr_enabled`, `adr_class`, `loading_types`, `requirements`, `shipment_type`, `belts_count`, `documents`.
    - Контакты: `contact_name`, `contact_phone`.
    - Связи: `cargo_type_id`, `company_id`.
    - Маршрут: `route_points` (минимум 2).
    - Оплата: `payment`.
  - Обязательные поля: `weight`, `volume`, `truck_type`, `capacity_required`, `route_points`.
  - Числовые ограничения:
    - `weight`, `volume`, `capacity_required` — `minimum: 0, exclusiveMinimum: true` (строго > 0).
  - `RoutePointInput`:
    - Обязательные: `type`, `city_code`, `address`, `lat`, `lng`, `point_order`.
    - `type` — enum `[LOAD, UNLOAD, CUSTOMS, TRANSIT]`.
    - Есть `place_id` для интеграции с картами.

- **Пример запроса `POST /api/cargo`**
  - В `application/json.example` указан полный реальный кейс:
    - Заполнены **все поля**: `name`, упаковка, габариты, фото, вес/объём, тип кузова, грузоподъёмность, готовность, комментарий, ADR, способы погрузки, требования, тип отправки, количество ремней, документы, контакты, `cargo_type_id`, `company_id`.
    - Маршрут с двумя точками (`LOAD` и `UNLOAD`), включающими `place_id`.
    - Блок `payment` с предоплатой/остатком и типами оплаты.

- **Описание статусов и справочников**
  - В разделе `Reference / Cargo` описаны:
    - `cargo_status`, `trip_status`, `offer_status`.
    - `route_point_type`, `truck_type`, `shipment_type`, `currency`, `prepayment_type`, `remaining_type`, `loading_type`, `created_by_type`.
  - Потоки статусов синхронизированы с кодом и данными в БД.

### 3. Справочник типов груза (cargo_types) и hint‑API

- **Таблица `cargo_types`**
  - Создана миграцией `000043_cargo_name_and_types.up.sql`:
    - Поля: `id`, `code`, `name_ru`, `name_uz`, `name_en`, `name_tr`, `name_zh`, `created_at`.
    - Связана с таблицей `cargo` через `cargo.cargo_type_id` (FK).
  - Список типов груза (с твоего ТЗ) подготавливается в виде `INSERT ... ON CONFLICT (code) DO NOTHING`:
    - `code` — латиницей в UPPERCASE (например `MOROZHENOE`, `FARSH`, `PESOK` и т.д.).
    - `name_ru` — оригинальное русское название.
    - `name_uz`, `name_en`, `name_tr`, `name_zh` — на первом этапе могут дублировать русский/код (переводы можно постепенно улучшать).

- **Hint‑эндпоинт `GET /v1/reference/cargo-types/hint`**
  - Реальный маршрут в `router.go`:
    - `v1.GET("/reference/cargo-types/hint", handlers.HintCargoTypes(deps.PG))`.
  - Логика handler’а:
    - Берёт язык из `X-Language` (`ru/uz/en/tr/zh`) и выбирает соответствующую колонку `name_*`.
    - Параметр `q` (опциональный) — фильтр по подстроке в выбранной колонке.
    - Возвращает до 50 записей, отсортированных по названию.
    - Формат ответа:
      ```json
      {
        "status": "success",
        "code": 200,
        "description": "... на X-Language ...",
        "data": {
          "items": [
            { "id": "UUID", "code": "MOROZHENOE", "name": "Мороженое" },
            ...
          ]
        }
      }
      ```
    - При отсутствии совпадений массив `items` пустой (`[]`), не `null`.

- **Swagger для cargo_types**
  - Добавлен путь `/v1/reference/cargo-types/hint`:
    - Tag: `Reference / Cargo`.
    - `q` — query‑параметр с примером и описанием.
    - Описание, что результат зависит от `X-Language` и используется как подсказка для выбора `cargo_type_id` в `POST /api/cargo`.
    - Ответ описан через общий `Envelope` + `data.items[] {id, code, name}`.
  - В `CargoCreateRequest` поле `cargo_type_id` описано как:
    - `UUID типа груза (подбор через GET /v1/reference/cargo-types/hint)`.

### 4. Локализация ответов (5 языков)

- Все новые и изменённые handler’ы используют:
  - `resp.OKLang`, `resp.SuccessLang`, `resp.ErrorLang`.
- Фразы (успех/ошибки) берутся из `internal/server/resp/i18n.go`:
  - Есть переводы на **ru, uz, en, tr, zh** для ключей, связанных с грузами и reference:
    - `failed_to_get_cargo`, `cargo_not_found`, `failed_to_create_cargo`, `failed_to_update_cargo`, `failed_to_delete_cargo`, `failed_to_list` и др.
  - Для `HintCargoTypes` используется ключ `failed_to_list` (тоже с переводами на 5 языков).

### 5. Краткая сводка

Раздел | Что сделано
------ | -----------
Cargo (модель/репо) | Структура доведена до ТЗ: упаковка, габариты, фото, грузоподъёмность, place_id, тип груза, строгие enum‑типы и валидация.
Cargo API (handlers) | Обновлены структуры запросов/ответов, добавлены теги валидации, проверка по справочникам, маппинг всех полей в слой репозитория.
Swagger (Cargo) | Схемы и примеры полностью соответствуют коду и БД, пример `POST /api/cargo` показывает заполнение всех полей.
Справочник `cargo_types` | Таблица в БД + hint‑эндпоинт с учётом X-Language, описан в Swagger и привязан к полю `cargo_type_id`.
Локализация | Все новые ответы и ошибки по Cargo и Reference локализованы на 5 языков через единую систему `resp` и i18n.

### 6. Таблица всех API по грузам и водителям

Метод | Путь | Кто вызывает | Назначение / комментарий
----- | ---- | ------------ | ------------------------
POST | `/api/cargo` | Фриланс‑диспетчер, компания, админ | Создание груза: все данные по грузу, маршруту (`route_points`), условиям оплаты (`payment`), контактам и типу груза (`cargo_type_id`).
GET | `/api/cargo` | Любая сторона (фильтры) | Список грузов с фильтрами по статусу, дате создания, весу, типу кузова, наличию офферов и видимости для водителя.
GET | `/api/cargo/:id` | Любая сторона | Карточка одного груза: основные поля + вложенные `route_points` и `payment`.
PUT | `/api/cargo/:id` | Создатель груза (до назначения) | Полное/частичное редактирование груза (поля груза, маршрут, условия оплаты), с ограничениями после назначения.
DELETE | `/api/cargo/:id` | Создатель груза | Мягкое удаление груза (`deleted_at`).
PATCH | `/api/cargo/:id/status` | Админ / диспетчер | Смена статуса груза с проверкой допустимых переходов (`CREATED/PENDING_MODERATION/SEARCHING_*` → `ASSIGNED/IN_PROGRESS/COMPLETED/...`).
POST | `/api/cargo/:id/offers` | Водитель | Создание оффера по грузу (цена, валюта, комментарий); для `SEARCHING_COMPANY` — только водители нужной компании.
GET | `/api/cargo/:id/offers` | Диспетчер / владелец груза | Список офферов по грузу.
POST | `/api/offers/:id/accept` | Диспетчер / владелец груза | Принятие оффера: груз → `ASSIGNED`, создание рейса, остальные офферы по грузу → `REJECTED`.
POST | `/v1/dispatchers/offers/:id/reject` | Фриланс‑диспетчер (создатель груза) | Отклонение оффера с опциональной причиной (только по своим грузам).
GET | `/v1/driver/recommended-cargo` | Водитель | Список рекомендованных грузов (приглашения от диспетчера).
POST | `/v1/driver/recommended-cargo/:cargoId/accept` | Водитель | Принять рекомендацию: создаётся оффер, он принимается, создаётся рейс, груз → `ASSIGNED`.
POST | `/v1/driver/recommended-cargo/:cargoId/decline` | Водитель | Отказ от рекомендации по грузу.
POST | `/v1/dispatchers/cargo/:id/recommend` | Фриланс‑диспетчер (владелец груза) | Отправить конкретный груз выбранному водителю (создание рекомендации).
GET | `/v1/driver/trips` | Водитель | Список рейсов водителя (включая созданные из офферов по грузам).
POST | `/v1/driver/trips/:id/confirm` | Водитель | Подтвердить рейс (после назначения по принятому офферу/рекомендации).
POST | `/v1/driver/trips/:id/reject` | Водитель | Отказаться от рейса.
PATCH | `/v1/driver/trips/:id/status` | Водитель | Изменить статус рейса (`LOADING`, `EN_ROUTE`, `UNLOADING`, `COMPLETED`, `CANCELLED`) с синхронизацией статуса груза.
GET | `/v1/reference/cargo` | Любой клиент | Полный справочник по грузам: статусы груза/рейса/оффера, `truck_type`, `route_point_type`, `shipment_type`, валюты и типы оплат.
GET | `/v1/reference/cargo` | Любой клиент | Дополнительно: `packaging_type` — справочник типов упаковки для UI (BAG, PALLETS, BULK, BOXES, HEAP) с переводами на 5 языков.
GET | `/v1/reference/cargo-types/hint` | Любой клиент | Подсказка по типам груза (`cargo_types`) с учётом `X-Language`, используется для выбора `cargo_type_id`.
GET | `/v1/reference/cities` | Любой клиент | Справочник городов для `city_code` в `route_points`.
GET | `/v1/driver/transport-options` | Водитель | Справочник типов ТС (тягачи/прицепы) для профиля водителя, используемый при подборе грузов.


