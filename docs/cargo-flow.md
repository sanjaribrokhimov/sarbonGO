# Cargo flow: логика, статусы и проверка через Swagger

---

## 1. Статусы: что реально работает в коде

В API, в БД и в справочнике везде используются значения **только в ВЕРХНЕМ регистре** (UPPERCASE). Справочник `GET /v1/reference/cargo` и все ответы API отдают статусы и enum-поля (truck_type, route_point type, payment types и т.д.) в UPPERCASE.

### 1.1 Статусы груза (cargo.status)

| Значение в API/БД | Кто выставляет | Описание |
|-------------------|----------------|----------|
| **PENDING_MODERATION** | Система при создании груза диспетчером | На модерации у админа |
| **REJECTED** | Админ (reject с обязательным reason) | Отклонён модерацией |
| **SEARCHING_ALL** | Админ (accept, search_visibility=all или по умолчанию) | Груз виден **всем** водителям; можно слать офферы |
| **SEARCHING_COMPANY** | Админ (accept, search_visibility=company; только для грузов компании) | Груз виден **только водителям своей компании**; офферы только от них |
| **ASSIGNED** | Система при принятии оффера или рекомендации | Перевозчик выбран, создан рейс |
| **IN_PROGRESS** | Система при смене рейса на LOADING | Водитель грузит / в пути |
| **COMPLETED** | Система при смене рейса на COMPLETED | Перевозка завершена |
| **CANCELLED** | Диспетчер/админ через PATCH cargo status | Отменён |
| **CREATED** | Не выставляется при текущем потоке | Оставлен для совместимости |
| **IN_TRANSIT** | Не выставляется автоматически; допустим для ручного PATCH | В пути (зарезервирован) |
| **DELIVERED** | Не выставляется автоматически; допустим для ручного PATCH | Доставлен (зарезервирован) |

Допустимые переходы статусов груза заданы в `internal/cargo/repo.go` (SetStatus). Текущий автоматический сценарий:  
`PENDING_MODERATION` → `SEARCHING_ALL` \| `SEARCHING_COMPANY` \| `REJECTED` → `SEARCHING_*` → `ASSIGNED` → (рейс LOADING) → `IN_PROGRESS` → (рейс COMPLETED) → `COMPLETED`.  
**Видимость при приёме модерации:** для грузов **компании** админ может выбрать `search_visibility`: **all** (SEARCHING_ALL) или **company** (SEARCHING_COMPANY). Для грузов **фриланс-диспетчера** всегда только **all** (выбора нет).

### 1.2 Статусы рейса (trip.status)

| Значение в API/БД | Кто выставляет | Описание |
|------------------|----------------|----------|
| **PENDING_DRIVER** | Система при создании рейса и назначении водителя | Ожидание подтверждения водителем |
| **ASSIGNED** | Водитель: POST .../trips/:id/confirm | Водитель принял рейс |
| **LOADING** | Водитель: PATCH .../trips/:id/status | Погрузка (груз → IN_PROGRESS) |
| **EN_ROUTE** | Водитель: PATCH .../trips/:id/status | В пути |
| **UNLOADING** | Водитель: PATCH .../trips/:id/status | Выгрузка |
| **COMPLETED** | Водитель: PATCH .../trips/:id/status | Рейс завершён (груз → COMPLETED) |
| **CANCELLED** | Водитель: PATCH .../trips/:id/status | Рейс отменён |

Водитель может передавать в PATCH только: `LOADING`, `EN_ROUTE`, `UNLOADING`, `COMPLETED`, `CANCELLED`.

### 1.3 Статусы оффера (offer.status)

| Значение в API/БД | Кто выставляет | Описание |
|-------------------|----------------|----------|
| **PENDING** | Система при создании оффера | На рассмотрении у диспетчера |
| **ACCEPTED** | Диспетчер (accept) или система при accept рекомендации | Принят |
| **REJECTED** | Диспетчер (reject) или система при принятии другого оффера | Отклонён |

### 1.4 Статусы рекомендации (cargo_driver_recommendations.status)

Внутренняя таблица, не в справочнике. Используются значения:

| Значение | Когда |
|----------|--------|
| **PENDING** | Диспетчер отправил рекомендацию |
| **ACCEPTED** | Водитель нажал «Принять» (после успешного создания оффера и принятия) |
| **DECLINED** | Водитель нажал «Отказать» |

---

## 2. Reference API (GET /v1/reference/cargo)

- **cargo_status** — все 11 статусов груза (CREATED, PENDING_MODERATION, SEARCHING_ALL, SEARCHING_COMPANY, REJECTED, ASSIGNED, IN_PROGRESS, IN_TRANSIT, DELIVERED, COMPLETED, CANCELLED) с label и description на 5 языках (X-Language: en, ru, uz, tr, zh).
- **trip_status** — PENDING_DRIVER, ASSIGNED, LOADING, EN_ROUTE, UNLOADING, COMPLETED, CANCELLED.
- **offer_status** — PENDING, ACCEPTED, REJECTED.

Справочник **полностью совпадает** с константами в коде (`internal/cargo/model.go`, `internal/trips/model.go`, офферы в БД). Менять reference не требуется.

---

## 3. Пошаговый поток (как работает сейчас)

**Фриланс-диспетчер универсален:** может и добавить на себя водителей (и подбирать им грузы), и добавить груз (и искать ему водителя). Порядок не фиксирован.

### Шаг 1: Диспетчер создаёт груз

- **Кто:** фриланс-диспетчер (X-User-Token, role=dispatcher).
- **API:** `POST /api/cargo` (weight, volume, truck_type, route_points, payment и др.).
- **Результат:** груз создаётся со статусом **pending_moderation**. Лимит грузов из env; валидация по справочникам.

### Шаг 2: Админ модерирует

- **Кто:** админ (X-User-Token, role=admin).
- **API:**  
  - `GET /v1/admin/cargo/moderation` — список грузов на модерации.  
  - `POST /v1/admin/cargo/{id}/moderation/accept` — принять. Тело опционально: `{"search_visibility": "all" | "company"}`. По умолчанию **all**.  
    - **all** → статус **SEARCHING_ALL** (груз виден всем водителям).  
    - **company** → статус **SEARCHING_COMPANY** (груз виден только водителям компании). Для грузов, созданных **фриланс-диспетчером**, всегда применяется только **all**.  
  - `POST /v1/admin/cargo/{id}/moderation/reject` — отклонить (body: `{"reason": "..."}` обязательно) → статус **rejected**.

### Шаг 3: Водители видят грузы

- Грузы в статусе **SEARCHING_ALL** видны **всем** водителям. Грузы **SEARCHING_COMPANY** видны только водителям **той же компании** (при запросе с X-User-Token водителя список автоматически фильтруется).
- Фильтр: `GET /api/cargo?status=SEARCHING_ALL,SEARCHING_COMPANY` (с JWT водителя вернутся только доступные ему грузы).
- Создание оффера разрешено по грузу в статусе **SEARCHING_ALL** или **SEARCHING_COMPANY**. Для **SEARCHING_COMPANY** оффер может отправить только водитель той же компании (иначе 403 `cargo_visible_only_to_company_drivers`).

### Шаг 3.1: Фриланс-диспетчер — универсальный сценарий

Фриланс-диспетчер может работать **в обе стороны**:

- **Добавить на себя водителя и подбирать ему грузы:** диспетчер приглашает водителя (`POST /v1/dispatchers/driver-invitations`, phone или driver_id). Водитель принимает → попадает в «мои водители». Список водителей с фильтрами: `GET /v1/dispatchers/drivers` (query: phone, work_status, truck_type, page, limit, sort). Дальше диспетчер отправляет **таклиф** (приглашение на груз) выбранному водителю: `POST /v1/dispatchers/cargo/{id}/recommend` (body: driver_id). Водитель видит детали груза в `GET /v1/driver/recommended-cargo` и принимает или отказывает.
- **Добавить груз и искать ему водителя:** диспетчер создаёт груз (`POST /api/cargo`), после модерации груз в поиске. Водители сами присылают офферы (`POST /api/cargo/{id}/offers`), либо диспетчер отправляет приглашение конкретному водителю (recommend). Диспетчер принимает оффер или водитель принимает рекомендацию — рейс создаётся.

Связку можно начать и с водителя: водитель «добавляет на себя» диспетчера (`POST /v1/driver/dispatcher-invitations`, body: phone). Диспетчер принимает (`POST /v1/dispatchers/invitations-from-drivers/accept`) — у водителя проставляется freelancer_id, диспетчер видит его в своих водителях и может подбирать ему грузы.

### Шаг 4: Назначение водителя — только через приглашения (офферы и рекомендации)

**Диспетчер не назначает (assign) водителя** — он только отправляет приглашения. Водитель назначается на груз только путём **принятия оффера** или **принятия рекомендации**. Для грузов фриланс-диспетчера вызов `PATCH /v1/dispatchers/trips/:id/assign-driver` запрещён (400).

**Два вида приглашений (в обе стороны можно отправить «пустое» приглашение — готов везти на указанную сумму — или приглашение с другой суммой):**

**Вариант A — Оффер от водителя (приглашение везти груз на сумму)**

1. Водитель: `POST /api/cargo/{id}/offers` (carrier_id, price, currency, comment) — приглашение перевезти груз на указанную сумму (или с комментарием «цена договорная»).
2. Диспетчер: `GET /api/cargo/{id}/offers` → принять или отклонить.
3. Принять: `POST /api/offers/{id}/accept` → создаётся рейс, груз → **assigned**, водитель привязывается к рейсу.
4. Отклонить: `POST /v1/dispatchers/offers/{id}/reject` (body: reason — необязательно).

**Вариант B — Рекомендация от диспетчера (приглашение водителю везти груз)**

1. Диспетчер: `POST /v1/dispatchers/cargo/{id}/recommend` (body: driver_id). Груз должен быть в статусе **searching** и принадлежать диспетчеру. Это приглашение водителю перевезти груз (по умолчанию на сумму из условий груза).
2. Водитель: `GET /v1/driver/recommended-cargo` — список приглашений.
3. Принять: `POST /v1/driver/recommended-cargo/{cargoId}/accept` → создаётся оффер по цене груза, оффер принимается, рейс создаётся, груз → **assigned**.
4. Отказать: `POST /v1/driver/recommended-cargo/{cargoId}/decline`.

В обоих вариантах можно отправить приглашение «на указанную сумму» (из груза или явно) или с другой суммой (водитель в оффере указывает свою цену; при рекомендации по умолчанию берётся сумма из груза).

### Шаг 5: Рейс и статус груза

- После принятия оффера или рекомендации: груз **assigned**, рейс создаётся в статусе **pending_driver**, водитель уже привязан к рейсу.
- Водитель подтверждает: `POST /v1/driver/trips/{id}/confirm` → рейс **assigned**.
- Водитель меняет этап: `PATCH /v1/driver/trips/{id}/status` (body: `{"status": "loading" | "en_route" | "unloading" | "completed" | "cancelled"}`).
- **Синхронизация с грузом:**  
  - рейс **loading** → груз переводится в **in_progress**;  
  - рейс **completed** → груз переводится в **completed**.

---

## 4. Как проверить в Swagger (разделы и API)

| Шаг | Действие | Раздел в Swagger | API |
|-----|----------|-------------------|-----|
| 0a | Водитель: пригласить диспетчера (добавить на себя) | Drivers / Invite dispatcher | POST /v1/driver/dispatcher-invitations (body: phone) |
| 0b | Диспетчер: принять приглашение от водителя | Freelance Dispatchers / Invitations from drivers | POST /v1/dispatchers/invitations-from-drivers/accept (body: token) |
| 0c | Диспетчер: пригласить водителя | Freelance Dispatchers / Приглашения водителей | POST /v1/dispatchers/driver-invitations (phone или driver_id) |
| 0d | Водитель: принять приглашение диспетчера | Drivers / Driver invitations | POST /v1/driver/driver-invitations/accept (body: token) |
| 1 | Создать груз (диспетчер) | Freelance Dispatchers / Добавление груза, Cargo — Диспетчер | POST /api/cargo |
| 2 | Список на модерации | Admin / Cargo moderation | GET /v1/admin/cargo/moderation |
| 3 | Принять груз (админ) | Admin / Cargo moderation | POST /v1/admin/cargo/{id}/moderation/accept (body: search_visibility all \| company) |
| 3' | Отклонить груз (админ) | Admin / Cargo moderation | POST /v1/admin/cargo/{id}/moderation/reject |
| 4 | Список грузов для водителя | Cargo — Водитель | GET /api/cargo?status=SEARCHING_ALL,SEARCHING_COMPANY (с X-User-Token — только доступные) |
| 5a | Водитель: оффер | Cargo — Водитель, Freelance Dispatchers / Добавление груза | POST /api/cargo/{id}/offers |
| 5b | Диспетчер: офферы по грузу | Freelance Dispatchers / Добавление груза | GET /api/cargo/{id}/offers |
| 5c | Диспетчер: принять оффер | Freelance Dispatchers / Добавление груза | POST /api/offers/{id}/accept |
| 5d | Диспетчер: отклонить оффер | Freelance Dispatchers / Добавление груза | POST /v1/dispatchers/offers/{id}/reject |
| 6a | Диспетчер: рекомендовать груз | Freelance Dispatchers / Добавление груза | POST /v1/dispatchers/cargo/{id}/recommend |
| 6b | Водитель: рекомендованные грузы | Drivers / Trips, Cargo — Водитель | GET /v1/driver/recommended-cargo |
| 6c | Водитель: принять рекомендацию | Drivers / Trips, Cargo — Водитель | POST /v1/driver/recommended-cargo/{cargoId}/accept |
| 6d | Водитель: отказать | Drivers / Trips, Cargo — Водитель | POST /v1/driver/recommended-cargo/{cargoId}/decline |
| 7 | Водитель: статус рейса | Drivers / Trips | PATCH /v1/driver/trips/{id}/status |
| — | Справочник статусов (все значения) | Reference / Cargo | GET /v1/reference/cargo |

---

## 5. Отчёт: проверка статусов и reference

- **Код:** В коде используются те же значения, что и в БД (UPPERCASE). Константы: `internal/cargo/model.go` (cargo: в т.ч. SEARCHING_ALL, SEARCHING_COMPANY), `internal/trips/model.go` (trip), офферы и рекомендации в соответствующих репозиториях.
- **Reference API:** В `GET /v1/reference/cargo` возвращаются все актуальные cargo_status, trip_status, offer_status (в верхнем регистре для UI). Локализация (label, description) в `internal/reference/i18n.go` — все ключи для статусов груза/рейса/оффера присутствуют.
- **Изменения в reference не требуются.** Текущий справочник соответствует коду и потоку.
- **Уточнения в потоке:** В текущей реализации груз автоматически переходит только в **in_progress** (при loading) и **completed** (при completed рейса). Статусы **in_transit** и **delivered** допустимы в PATCH и в переходах, но системой не выставляются (зарезервированы на будущее или ручное управление).

Ответы API поддерживают 5 языков (X-Language: en, ru, uz, tr, zh). Ключи ошибок — в `internal/server/resp/i18n.go`.

---

## 6. Два вида «поиска»: SEARCHING_ALL и SEARCHING_COMPANY

- **SEARCHING_ALL** — груз виден **всем** водителям (как раньше SEARCHING). По умолчанию при приёме модерации и для грузов фриланс-диспетчера.
- **SEARCHING_COMPANY** — груз виден **только водителям своей компании**. Доступен при приёме модерации для грузов, созданных компанией (body: `search_visibility: "company"`).
- Водитель при запросе списка с X-User-Token видит только доступные ему грузы: все SEARCHING_ALL + SEARCHING_COMPANY своей компании. Оффер по грузу SEARCHING_COMPANY может отправить только водитель той же компании (иначе 403).

---

## 7. Что изменилось (миграции и запуск)

### Запуск приложения

- **Команда:** из корня репозитория `go run ./cmd/api` или из `cmd/api` — `go run main.go`.
- **Миграции** выполняются **автоматически** при старте: поднимаются все неприменённые миграции из `migrations/`.
- Если БД в состоянии **Dirty** (миграция когда-то упала), приложение само сбрасывает версию на предыдущую и повторяет `Up()` один раз (только для упавшей миграции).

### Изменения в миграциях

| Миграция | Что сделано |
|----------|--------------|
| **000035** (driver_powers / driver_trailers) | Идемпотентность: перенос данных из `drivers` и DROP колонок выполняются **только если** в `drivers` ещё есть колонка `power_plate_number`. Если схема уже разнесена или колонок не было — блок пропускается, ошибок нет. |
| **000040** (status/refs UPPERCASE) | Для таблицы `cargo` перед UPDATE теперь снимаются **оба** check: и по `status`, и по `created_by_type`. После UPDATE заново вешаются `cargo_status_check` и `cargo_created_by_type_check` с значениями в UPPERCASE (`ADMIN`, `DISPATCHER`, `COMPANY`). Раньше падало на `cargo_created_by_type_check`, т.к. старый check допускал только lowercase. |
| **000041** (SEARCHING_ALL / SEARCHING_COMPANY) | Статус `SEARCHING` заменён на два: `SEARCHING_ALL` (виден всем) и `SEARCHING_COMPANY` (только водителям компании). В БД существующие записи с `SEARCHING` обновляются на `SEARCHING_ALL`. |

### Итог

- Запуск одного `go run main.go` (или `go run ./cmd/api`) достаточен: миграции применяются сами, при единичном сбое — авто-повтор после сброса dirty-версии.
