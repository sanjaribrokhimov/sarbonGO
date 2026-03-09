# Отчёт: API для User Flow фриланс-диспетчера

Реализованы все API, необходимые для описанного User Flow фриланс-диспетчера. Документация по категориям — в Swagger (`docs/openapi.yaml`).

---

## 1. Кто такой фриланс-диспетчер

**Диспетчер не создаёт компанию — только аккаунт.** Два типа:
- **Фриланс** — добавляет грузы, ищет грузы для водителей, приглашает водителей (без компании); водители привязаны к диспетчеру через `freelancer_id`.
- **Диспетчер компании** — через приглашение от компании (по телефону); после принятия видит компанию в списке и может переключать контекст.

---

## 2. Регистрация и выбор пути

**Регистрация:**  
`POST /v1/dispatchers/auth/phone` → `POST /v1/dispatchers/auth/otp/verify` (status=register) → `POST /v1/dispatchers/registration/complete` (name, password, passport, pinfl).

**Два пути после регистрации:**

| Путь | API |
|------|-----|
| Фриланс (без компании) | Пригласить водителей: `POST /v1/dispatchers/driver-invitations` (phone). Список водителей: `GET /v1/dispatchers/drivers`. Грузы: POST/GET /api/cargo, офферы, принять оффер, рейсы, назначить водителя. |
| Диспетчер компании | Ожидание приглашения → `POST /v1/dispatchers/invitations/accept` (token) или decline. Список компаний: `GET /v1/dispatchers/companies`. Переключение: `POST /v1/dispatchers/auth/switch-company`. Пригласить водителя в компанию: `POST /v1/dispatchers/companies/:companyId/driver-invitations`. |

---

## 3. Путь 1: Диспетчер компании (через приглашение)

- Компания создаёт приглашение: `POST /v1/dispatchers/companies/:companyId/invitations` (phone, role=dispatcher|top_dispatcher) → token.
- Диспетчер: `POST /v1/dispatchers/invitations/accept` (token) — телефон должен совпадать с аккаунтом.
- Список компаний: `GET /v1/dispatchers/companies` — в списке компании, куда приглашён.

---

## 4. Путь 2: Фриланс (без компании)

- Пригласить водителя (без компании): `POST /v1/dispatchers/driver-invitations` (phone) → token. Водитель: `POST /v1/driver-invitations/accept` (token) → у водителя устанавливается `freelancer_id` = диспетчер.
- Список моих водителей: `GET /v1/dispatchers/drivers`.
- Если диспетчер также приглашён в компанию, может приглашать водителей в компанию: `POST /v1/dispatchers/companies/:companyId/driver-invitations` (phone) → при принятии у водителя `company_id`.

---

## 5. Вход и контекст компании

- Вход: `POST /v1/dispatchers/auth/phone` + `POST /v1/dispatchers/auth/otp/verify` или `POST /v1/dispatchers/auth/login/password` → tokens.
- Список компаний: `GET /v1/dispatchers/companies`.
- Выбор компании (токен с company_id): `POST /v1/dispatchers/auth/switch-company` (company_id) → новый access_token с company_id в claims. Дальнейшие запросы могут использовать этот контекст.

---

## 6. Основные рабочие сценарии

### 6.1 Диспетчер как заказчик (создание груза)

| Действие | API |
|----------|-----|
| Создать груз | `POST /api/cargo` (маршрут, вес, даты, тип ТС, оплата и т.д.). При X-User-Token создатель записывается как dispatcher. |
| Опубликовать / статус searching | `PATCH /api/cargo/:id/status` (status=searching) |
| Список ставок | `GET /api/cargo/:id/offers` |
| Принять ставку | `POST /api/offers/:id/accept` → груз assigned, создаётся **рейс** (trip) со статусом pending_driver, в ответе cargo_id, offer_id, **trip_id** |
| Отклонить | Другие офферы остаются pending; при принятии другого оффера они помечаются rejected |

### 6.2 Диспетчер как перевозчик (поиск грузов и ставки)

| Действие | API |
|----------|-----|
| Поиск грузов | `GET /api/cargo?status=searching` |
| Создать ставку | `POST /api/cargo/:id/offers` (carrier_id, price, currency, comment) |
| После принятия ставки грузовладельцем | Создаётся рейс pending_driver. Назначить водителя: `PATCH /v1/dispatchers/trips/:id/assign-driver` (driver_id) |
| Водитель подтверждает | `POST /v1/trips/:id/confirm` (driver JWT) → рейс assigned |
| Водитель отклоняет | `POST /v1/trips/:id/reject` → диспетчер может назначить другого водителя |
| Водитель меняет статусы рейса | `PATCH /v1/trips/:id/status` (loading → en_route → unloading → completed) |

---

## 7. Рейсы (trips)

| Метод | Путь | Назначение |
|-------|------|------------|
| GET | /api/trips?cargo_id= | Рейс по грузу (базовые заголовки) |
| GET | /api/trips/:id | Рейс по ID |
| GET | /v1/trips | Список рейсов водителя (X-User-Token driver) |
| PATCH | /v1/dispatchers/trips/:id/assign-driver | Назначить водителя (dispatcher) |
| POST | /v1/trips/:id/confirm | Водитель подтверждает |
| POST | /v1/trips/:id/reject | Водитель отклоняет |
| PATCH | /v1/trips/:id/status | Водитель: loading, en_route, unloading, completed, cancelled |

**Статусы рейса:** pending_driver → assigned → loading → en_route → unloading → completed (или cancelled).

---

## 8. Swagger: категории и теги

В Swagger добавлены теги и пути:

- **Freelance Dispatchers / Companies** — создание компании Broker, список компаний, switch-company.
- **Freelance Dispatchers / Invitations** — приглашение диспетчера по телефону, accept/decline.
- **Freelance Dispatchers / Driver invitations** — приглашение водителя в компанию.
- **Freelance Dispatchers / Trips** — назначение водителя на рейс (assign-driver).
- **Drivers / Trips** — список рейсов водителя, confirm, reject, смена статуса.
- **Drivers / Driver invitations** — принять приглашение в компанию.

В разделе **Cargo** обновлён ответ `POST /api/offers/:id/accept` (добавлен trip_id), добавлены `GET /api/trips` и `GET /api/trips/:id`.

---

## 9. Миграция и схема

**Миграция 000030_freelance_dispatcher_flow:**

- `companies.owner_dispatcher_id` (FK → freelance_dispatchers).
- Таблица `dispatcher_company_roles` (dispatcher_id, company_id, role, invited_by, accepted_at).
- Таблица `dispatcher_invitations` (token, company_id, role, phone, invited_by, expires_at).
- Таблица `driver_invitations` (token, company_id, phone, invited_by, expires_at).
- Таблица `trips` (id, cargo_id, offer_id, driver_id, status, created_at, updated_at). Статусы: pending_driver, assigned, loading, en_route, unloading, completed, cancelled.

Приложение при старте выполняет миграции вверх (в т.ч. 030), если настроен `DATABASE_URL`.

---

## 10. Завершение работы

- Выход: `POST /v1/dispatchers/auth/logout` (refresh_token) — инвалидация refresh-токена.

Все ключевые действия используют JWT (X-User-Token) и при необходимости контекст компании (company_id в токене после switch-company). Рейсы создаются при принятии оффера; подтверждение водителем обязательно — диспетчер не может «принудительно» назначить водителя без вызова водителем `POST /v1/trips/:id/confirm`.
