# Документация: создатель груза, компания и статусы по умолчанию

## 1. Автоматическое определение создателя груза

При создании груза (**POST /v1/api/cargo**) система **автоматически** записывает, кто создал груз: админ, диспетчер или компания.

### Правила заполнения полей `created_by_type` и `created_by_id`

| Условие | Результат |
|--------|-----------|
| В запросе передан заголовок **X-User-Token** с валидным JWT и ролью **admin** | `created_by_type = "admin"`, `created_by_id` = UUID из таблицы `admins`. Опционально в теле можно передать `company_id` — груз будет привязан к компании. |
| В запросе передан заголовок **X-User-Token** с валидным JWT и ролью **dispatcher** | `created_by_type = "dispatcher"`, `created_by_id` = UUID из таблицы `freelance_dispatchers`. В теле можно передать `company_id` для привязки груза к компании. |
| В теле запроса передан **company_id**, а JWT отсутствует или роль не admin/dispatcher | `created_by_type = "company"`, `created_by_id` = переданный `company_id`, `company_id` = тот же UUID (создатель — компания из таблицы `companies`). |

### Поля в ответах API

В ответах **GET /v1/api/cargo** (список) и **GET /v1/api/cargo/:id** (карточка) каждый груз содержит при наличии:

- **created_by_type** — `"admin"` | `"dispatcher"` | `"company"`
- **created_by_id** — UUID создателя (соответствует таблице: admins, freelance_dispatchers или companies)
- **company_id** — UUID компании, от которой груз (опционально; при `created_by_type=company` совпадает с `created_by_id`)
- **created_at** — дата и время создания записи

### Примеры

**Создание от имени админа (с JWT):**
```http
POST /v1/api/cargo
X-User-Token: <JWT с role=admin>
Content-Type: application/json

{ "title": "...", "weight": 10, "truck_type": "tent", "capacity": 20, "route_points": [...], "company_id": "uuid-компании-опционально" }
```
→ В БД: `created_by_type=admin`, `created_by_id=<id админа>`, `company_id` — если передан.

**Создание от имени диспетчера (с JWT):**
```http
POST /v1/api/cargo
X-User-Token: <JWT с role=dispatcher>
Content-Type: application/json

{ "title": "...", "weight": 10, "truck_type": "tent", "capacity": 20, "route_points": [...], "company_id": "uuid-компании-опционально" }
```
→ В БД: `created_by_type=dispatcher`, `created_by_id=<id диспетчера>`, `company_id` — если передан.

**Создание от имени компании (без JWT, только company_id в теле):**
```http
POST /v1/api/cargo
Content-Type: application/json

{ "title": "...", "weight": 10, "truck_type": "tent", "capacity": 20, "route_points": [...], "company_id": "uuid-компании" }
```
→ В БД: `created_by_type=company`, `created_by_id=company_id`, `company_id=company_id`.

---

## 2. Статусы по умолчанию при создании сущностей

### Компании (companies)

- При создании компании (**POST /v1/admin/companies**) поле **status** по умолчанию устанавливается в **active**, если в теле не передан другой статус.
- Допустимые значения: `active`, `inactive`, `blocked`, `pending`.
- После создания статус можно изменить (например, на **inactive**) через обновление записи в админке или API.

### Фриланс-диспетчеры (freelance_dispatchers)

- При регистрации диспетчера автоматически устанавливаются:
  - **account_status** = **active**
  - **work_status** = **available**
- Дальнейшее изменение — через обновление профиля/статуса.

### Грузы (cargo)

- При создании груза (**POST /v1/api/cargo**) статус по умолчанию — **created** (если в теле не передан иной, например `searching`).
- Жизненный цикл: `created` → `searching` → `assigned` → `in_transit` → `delivered`; возможна отмена (`cancelled`) из состояний created, searching, assigned.
- Смена статуса — через **PATCH /v1/api/cargo/:id/status** с телом `{"status": "searching"}` и т.д.

---

## 3. Связь с OpenAPI (Swagger)

Полное описание API грузов, схем запросов/ответов и параметров см. в **OpenAPI (Swagger)**:

- Схема **Cargo** — поля груза, в том числе `created_by_type`, `created_by_id`, `company_id`, `created_at`.
- Схема **CargoCreateRequest** — тело POST /api/cargo, описание автоматического заполнения создателя и опционального `company_id`.
- Эндпоинты: POST/GET/PUT/DELETE /api/cargo, PATCH /api/cargo/:id/status, офферы и приём оффера.

Swagger UI доступен по адресу, указанному в конфигурации приложения (обычно `/swagger/` или аналогичный путь к UI OpenAPI).

---

## 4. Краткая сводка

| Сущность | Что записывается автоматически при создании |
|----------|---------------------------------------------|
| **Груз** | Создатель: admin / dispatcher / company и его ID; дата создания; опционально привязка к компании. |
| **Компания** | Статус по умолчанию **active**. |
| **Фриланс-диспетчер** | Статус **active**, work_status **available**. |
| **Груз** | Статус по умолчанию **created**; далее смена через PATCH status. |

Все изменения отражены в миграциях (cargo: created_by_type с учётом company), в коде (handler, repo, model) и в OpenAPI-документации.
