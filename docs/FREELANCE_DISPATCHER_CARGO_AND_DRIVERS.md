# Отчёт: API фриланс-диспетчера — добавление груза и приглашения водителей

В Swagger в категории **Freelance Dispatchers** в первую очередь вынесены: **добавление груза** и **отправка приглашений водителям**. Все нужные API уже реализованы, ниже — как ими пользоваться и где они в документации.

---

## 1. Категории в Swagger для фриланс-диспетчера

В **docs/openapi.yaml** для фриланс-диспетчера добавлены две отдельные группы (теги):

| Тег | Назначение |
|-----|------------|
| **Freelance Dispatchers / Добавление груза** | Создание груза, список, карточка, редактирование, статус, офферы, принятие оффера, рейсы по грузу |
| **Freelance Dispatchers / Приглашения водителей** | Отправка приглашения водителю (по телефону), список моих водителей |

При выборе этих тегов в Swagger отображаются только методы, относящиеся к добавлению груза и приглашениям водителей.

---

## 2. Добавление груза (API)

Все запросы к грузам идут на **/api/***. Обязательные заголовки: X-Device-Type, X-Language, X-Client-Token. Для записи создателя груза диспетчером нужен **X-User-Token** (JWT после входа фриланс-диспетчера).

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /api/cargo | **Создать груз** — тело: weight, volume (м³), truck_type, route_points (минимум load + unload), опционально payment. Статус при создании автоматически `created`; смена через PATCH …/status. С X-User-Token (dispatcher) в груз пишется created_by_type=dispatcher. |
| GET | /api/cargo | Список грузов (фильтры: status, weight_min/max, truck_type, page, limit). |
| GET | /api/cargo/:id | Карточка груза (с route_points и payment). |
| PUT | /api/cargo/:id | Обновить груз (до перехода в assigned и далее — ограничения). |
| PATCH | /api/cargo/:id/status | Сменить статус (например created → searching). |
| DELETE | /api/cargo/:id | Мягкое удаление груза. |
| GET | /api/cargo/:id/offers | Список офферов по грузу. |
| POST | /api/cargo/:id/offers | Создать оффер (как перевозчик; body: carrier_id, price, currency, comment). |
| POST | /api/offers/:id/accept | Принять оффер — груз переходит в assigned, создаётся рейс (trip). |
| GET | /api/trips?cargo_id= | Рейс по грузу. |
| GET | /api/trips/:id | Рейс по ID. |

После принятия оффера диспетчер назначает водителя на рейс: **PATCH /v1/dispatchers/trips/:id/assign-driver** (body: driver_id). Водитель подтверждает: **POST /v1/trips/:id/confirm** (с водительским JWT).

---

## 3. Приглашения водителей (API)

Фриланс-диспетчер приглашает водителей **без компании**. Водитель при принятии привязывается к диспетчеру (freelancer_id).

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /v1/dispatchers/driver-invitations | **Отправить приглашение водителю.** Тело: `{"phone": "+998901234567"}`. В ответе — token; его нужно передать водителю (чат/ссылка). Обязателен X-User-Token (dispatcher). |
| GET | /v1/dispatchers/drivers | Список водителей, привязанных к текущему диспетчеру (freelancer_id = me). Query: limit (по умолчанию 100). X-User-Token (dispatcher). |

Водитель принимает приглашение: **POST /v1/driver-invitations/accept** (body: token) с **X-User-Token** (водитель). Телефон водителя должен совпадать с приглашением; после принятия у водителя выставляется freelancer_id = диспетчер. В ответе приходит `freelancer_id` (или `company_id`, если приглашение было от компании).

---

## 4. Где это в Swagger

1. Откройте Swagger UI (например `/docs` или хост из конфига).
2. В списке тегов найдите:
   - **Freelance Dispatchers / Добавление груза** — все методы по грузам и рейсам для диспетчера.
   - **Freelance Dispatchers / Приглашения водителей** — отправка приглашения и список моих водителей.
3. Выберите тег — отобразятся только операции этой группы.
4. Для вызова методов укажите в Authorize: X-Device-Type, X-Language, X-Client-Token и при необходимости **X-User-Token** (JWT фриланс-диспетчера или водителя).

---

## 5. Краткий сценарий для фриланс-диспетчера

1. **Вход:** POST /v1/dispatchers/auth/phone → POST /v1/dispatchers/auth/otp/verify (или login/password) → получить access_token.
2. **Добавить груз:** POST /api/cargo с X-User-Token, телом (weight, volume, route_points, payment и т.д.) → получить id груза.
3. **Опубликовать:** PATCH /api/cargo/:id/status, body: `{"status": "searching"}`.
4. **Пригласить водителей:** POST /v1/dispatchers/driver-invitations, body: `{"phone": "+998..."}` → token отправить водителю.
5. **Мои водители:** GET /v1/dispatchers/drivers.
6. Когда приходят офферы: GET /api/cargo/:id/offers → POST /api/offers/:id/accept → PATCH /v1/dispatchers/trips/:id/assign-driver (driver_id из списка моих водителей).

Все перечисленные API реализованы и описаны в Swagger под категориями фриланс-диспетчера.
