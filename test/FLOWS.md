# Описание flow и проверок тестов

Запуск всех тестов: из корня проекта  
`go test -v ./test/...`  
или по отдельности:  
`go test -v ./test/driver/`  
`go test -v ./test/dispatcher/`  
`go test -v ./test/cargo/`

---

## 1. Driver flow (водитель)

**Какой правильный flow:**  
1. Мобильное приложение получает номер телефона и отправляет OTP (вне текущих тестов).  
2. После успешного OTP бэкенд создаёт сессию (session_id в Redis).  
3. Клиент вызывает **POST /v1/registration/start** с телом: `session_id`, `name`, `oferta_accepted: true`.  
4. Ответ: `data.status` = `registered` или `login`, `data.tokens.access_token`, `data.tokens.refresh_token`, `data.driver`.  
5. Все последующие запросы к защищённым эндпоинтам выполняются с заголовком **X-User-Token** = `access_token`.  
6. Дополнительно: PATCH /v1/profile/driver (name, work_status), PATCH /v1/registration/geo-push, PATCH /v1/registration/transport-type.

**Что проверяется в тестах:**

| Файл | Тест | Проверка |
|------|------|----------|
| `driver/health_test.go` | Health | GET /health без заголовков → 200, data.status=ok |
| | BaseHeadersRequired | GET /v1/transport-options с X-Device-Type, X-Language, X-Client-Token → 200 |
| `driver/registration_test.go` | RegistrationStart_Positive | Session → start → токены (access+refresh) и data.driver; GET /v1/profile с токеном → тот же driver |
| `driver/profile_test.go` | ProfilePatch_Positive | PATCH profile/driver → data.event=updated, data.driver.name и work_status |
| | GeoPush_Positive | PATCH registration/geo-push с координатами → 200 |
| | TransportType_Positive | PATCH registration/transport-type → data.driver.driver_type |
| `driver/negative_test.go` | MissingClientToken | Без X-Client-Token → 400, "missing required headers" |
| | InvalidClientToken | Неверный X-Client-Token (при CLIENT_TOKEN_EXPECTED) → 401, "invalid X-Client-Token" |
| | MissingBaseHeaders | Без заголовков → 400 |
| | InvalidDeviceType | X-Device-Type=desktop → 400, "invalid X-Device-Type" |
| | RegistrationStart_InvalidSession | Несуществующий session_id → 401, "session" |
| | RegistrationStart_OfertaNotAccepted | oferta_accepted=false → 400 |
| | Profile_NoToken | GET /v1/profile без X-User-Token → 401, "missing X-User-Token" |
| | Profile_InvalidToken | Невалидный JWT в X-User-Token → 401, "invalid X-User-Token" |
| | ProfilePatch_InvalidWorkStatus | work_status=invalid_status → 400, "work_status" |
| | TransportType_InvalidDriverType | driver_type=invalid_type → 400, "driver_type" |

**Заголовки:**  
- **Base headers** (обязательны для /v1/*): `X-Device-Type` (ios | android | web), `X-Language` (ru | uz | en | tr | zh), `X-Client-Token`.  
- **User token:** для защищённых эндпоинтов — `X-User-Token` = JWT access_token.

---

## 2. Freelance Dispatcher flow (диспетчер)

**Какой правильный flow:**  
1. **Регистрация нового диспетчера:**  
   - **POST /v1/dispatchers/auth/phone** с `phone` → otp_sent (при необходимости Telegram).  
   - **POST /v1/dispatchers/auth/otp/verify** с `phone`, `otp` → для нового номера: `status=register`, `session_id`.  
   - **POST /v1/dispatchers/registration/complete** с `session_id`, `name`, `password`, `passport_series`, `passport_number`, `pinfl` → `status=registered`, `tokens`, `dispatcher`.  
2. **Вход существующего диспетчера:**  
   - **POST /v1/dispatchers/auth/login/password** с `phone`, `password` → `status=login`, `tokens`.  
3. Все запросы к защищённым эндпоинтам — с **X-User-Token** = access_token.  
4. **GET /v1/dispatchers/profile** — текущий диспетчер; **PATCH /v1/dispatchers/profile** — обновление name, passport и т.д.

**Что проверяется в тестах:**

| Файл | Тест | Проверка |
|------|------|----------|
| `dispatcher/registration_test.go` | RegistrationComplete_Positive | Создание session в тесте → complete → tokens + data.dispatcher; GET /v1/dispatchers/profile с токеном |
| `dispatcher/auth_test.go` | LoginPassword_Positive | Регистрация → логин по phone+password → tokens; GET profile по токену |
| `dispatcher/profile_test.go` | ProfilePatch_Positive | После регистрации PATCH profile (name) → data.status=ok, data.dispatcher.name |
| `dispatcher/negative_test.go` | MissingClientToken | Запрос без X-Client-Token → 400 |
| | RegistrationComplete_InvalidSession | complete с несуществующим session_id → 401, "session" |
| | Profile_NoToken | GET profile без X-User-Token → 401, "missing X-User-Token" |
| | Profile_InvalidToken | Невалидный X-User-Token → 401, "invalid X-User-Token" |
| | LoginPassword_InvalidCredentials | Неверный пароль → 401, "invalid phone or password" |

**Примечание:** сценарии с отправкой OTP (auth/phone, auth/otp/verify) требуют настройки Telegram и в этих тестах не вызываются; проверяются complete (с session, созданной в тесте), login/password и profile.

---

## 3. Cargo + Chat flow (диспетчер создаёт груз → водитель видит → оффер → согласие/отказ + чат)

**Роли:** диспетчер (создаёт груз, принимает оффер), водитель (видит груз, подаёт оффер или отказывается). Чат — между водителем и диспетчером (conversation, messages).

### 3.1 Cargo API (все эндпоинты в тестах)

| Шаг | Кто | Метод | Эндпоинт | Описание |
|-----|-----|--------|----------|----------|
| 1 | Диспетчер | POST | `/api/cargo` | Создать груз (title, weight, route_points load/unload, truck_type, capacity, payment). Заголовок X-User-Token = dispatcher JWT → created_by_type=dispatcher. |
| 2 | Водитель | GET | `/api/cargo` | Список грузов (page, limit, status, with_offers и т.д.). Видит созданный груз. |
| 3 | Водитель | GET | `/api/cargo/:id` | Детали груза (route_points, payment). |
| 4 | Водитель | POST | `/api/cargo/:id/offers` | Подать оффер: body `carrier_id` (id водителя из GET /v1/profile), `price`, `currency`, `comment`. Ответ: id оффера. |
| 5 | Диспетчер | GET | `/api/cargo/:id/offers` | Список офферов по грузу. |
| 6 | Диспетчер | POST | `/api/offers/:id/accept` | Принять оффер → статус груза `assigned`, остальные офферы по этому грузу → `rejected`. |
| — | Любой | PATCH | `/api/cargo/:id/status` | Смена статуса (created → searching → assigned → in_transit → delivered или cancelled). |
| — | Создатель | PUT | `/api/cargo/:id` | Редактирование груза (до назначения). |
| — | — | DELETE | `/api/cargo/:id` | Удаление груза (soft). |

**Ситуации в тестах:**  
- **Согласие:** диспетчер создал груз → водитель увидел (list/get) → водитель подал оффер → диспетчер принял оффер → груз в статусе assigned.  
- **Отказ:** водитель видит груз, но не подаёт оффер (просто list/get без CreateOffer); или два водителя подали офферы — диспетчер принял один, второй автоматически rejected.  
- **Негатив:** неверный cargo_id/offer_id, неверный body (нет load/unload, неверный carrier_id и т.д.), offer не в pending.

### 3.2 Chat API (в тестах)

| Шаг | Кто | Метод | Эндпоинт | Описание |
|-----|-----|--------|----------|----------|
| 1 | Водитель или диспетчер | GET | `/v1/chat/conversations` | Список своих диалогов (X-User-Token = driver или dispatcher JWT). |
| 2 | Водитель | POST | `/v1/chat/conversations` | Создать/получить диалог: body `peer_id` = id диспетчера (из GET /v1/dispatchers/profile или из регистрации). |
| 3 | Диспетчер | POST | `/v1/chat/conversations` | Аналогично: `peer_id` = id водителя. |
| 4 | Любой | POST | `/v1/chat/conversations/:id/messages` | Отправить сообщение (body text). |
| 5 | Любой | GET | `/v1/chat/conversations/:id/messages` | Список сообщений (limit, cursor). |

**user_id в чате:** для driver JWT = driver.id, для dispatcher JWT = dispatcher.id; GetOrCreateConversation(driver_id, dispatcher_id) — один диалог на пару.

**Что проверяется в тестах (cargo/):**  
- `cargo_flow_test.go`: полный сценарий create → list → get → create offer → list offers → accept; сценарий «водитель видит груз, не подаёт оффер»; сценарий два оффера → accept один.  
- `chat_flow_test.go`: driver/dispatcher получают или создают conversation (peer_id = второй участник), отправляют сообщения, получают list messages.  
- `negative_test.go`: cargo create с невалидным body; get/offers по несуществующему id; accept несуществующего/уже принятого оффера; chat — неверный peer_id, тот же user.

---

## Структура тестов и Flow в каждом файле

В **начале каждого тестового файла** есть блок **«--- Flow этого файла (как проходит тест) ---»**: пошагово описано, что делает файл и как проходит проверка. Так ты видишь flow по каждому направлению (driver/health, driver/registration, driver/profile, driver/negative, dispatcher/registration, dispatcher/auth, dispatcher/profile, dispatcher/negative).

- **test/common/** — общие хелперы: DecodeEnvelope, AssertSuccess, AssertError, TokensFromData, DriverFromData, DispatcherFromData.  
- **test/driver/** — setup_test.go (TestMain, baseHeaders, req), health_test.go, registration_test.go, profile_test.go, negative_test.go (в каждом — свой блок Flow).  
- **test/dispatcher/** — setup_test.go, registration_test.go, auth_test.go, profile_test.go, negative_test.go (в каждом — свой блок Flow).  
- **test/cargo/** — setup_test.go (driver + dispatcher токены, id из profile), cargo_flow_test.go (cargo create → list → get → offer → accept; отказ/два оффера), chat_flow_test.go (conversations, messages), negative_test.go (cargo/chat ошибки).  
- **test/flows_test.go** — один тест, указывающий на этот документ.  
- **test/FLOWS.md** — этот файл (общее описание flow и проверок).

Один общий запуск:  
`go test -v ./test/...`

---

## Покрытие API: что проверено и что нет

**Полный API в приложении не покрыт тестами.** Сейчас проверяется только flow водителя (Driver) и фриланс-диспетчера (Freelance Dispatcher) в объёме ниже.

### Проверено тестами (driver + dispatcher)

| Эндпоинт | Метод | Где тест |
|----------|--------|----------|
| `/health` | GET | driver/health_test.go |
| `/v1/transport-options` | GET | driver/health_test.go, driver/negative_test.go |
| `/v1/registration/start` | POST | driver/registration_test.go, driver/profile_test.go, driver/negative_test.go |
| `/v1/profile` | GET | driver/registration_test.go, driver/negative_test.go |
| `/v1/profile/driver` | PATCH | driver/profile_test.go, driver/negative_test.go |
| `/v1/registration/geo-push` | PATCH | driver/profile_test.go |
| `/v1/registration/transport-type` | PATCH | driver/profile_test.go, driver/negative_test.go |
| `/v1/dispatchers/registration/complete` | POST | dispatcher/*.go |
| `/v1/dispatchers/auth/login/password` | POST | dispatcher/auth_test.go, dispatcher/negative_test.go |
| `/v1/dispatchers/profile` | GET, PATCH | dispatcher/registration_test.go, auth_test.go, profile_test.go, negative_test.go |
| **Cargo** | | **test/cargo/** |
| `/api/cargo` | POST, GET, GET/:id, PUT/:id, PATCH/:id/status, DELETE/:id | cargo/cargo_flow_test.go, cargo/negative_test.go |
| `/api/cargo/:id/offers` | POST, GET | cargo/cargo_flow_test.go |
| `/api/offers/:id/accept` | POST | cargo/cargo_flow_test.go |
| **Chat** | | **test/cargo/** |
| `/v1/chat/conversations` | GET, POST | cargo/chat_flow_test.go |
| `/v1/chat/conversations/:id/messages` | GET, POST | cargo/chat_flow_test.go |

### Не проверено тестами

- **Driver (водитель):**  
  `/v1/auth/phone`, `/v1/auth/otp/verify`, `/v1/auth/refresh`, `/v1/auth/logout`  
  `/v1/profile` — PUT heartbeat, POST phone-change (request/verify), PATCH power/trailer, DELETE  
  `/v1/registration/...` — только start, geo-push, transport-type; остальное не трогаем  
  `/v1/kyc` — PATCH (KYC submit)

- **Справочники (reference):**  
  GET `/v1/reference/drivers`, `/reference/cargo`, `/reference/company`, `/reference/admin`, `/reference/dispatchers` — не вызываются в тестах.

- **Freelance Dispatcher:**  
  `/v1/dispatchers/auth/phone`, `/v1/dispatchers/auth/otp/verify`  
  `/v1/dispatchers/auth/reset-password/request`, `/v1/dispatchers/auth/reset-password/confirm`  
  PUT `/v1/dispatchers/profile/password`, POST phone-change (request/verify), DELETE `/v1/dispatchers/profile`

- **User auth (app user):**  
  POST `/v1/auth/register`, `/v1/auth/login` — отдельный от driver flow, не тестируется.

- **Admin:**  
  POST `/v1/admin/auth/login/password`, POST `/v1/admin/companies` — не тестируется.

- **Company TZ (компании, инвайты, роли):**  
  GET `/v1/auth/companies`, POST `/v1/auth/switch-company`, POST `/v1/companies`, invitations, users, role, remove, POST `/v1/invitations/accept` — не тестируется.

- **Chat:**  
  PATCH/DELETE message, GET presence, GET `/v1/chat/ws` — не тестируются (conversations и messages — в cargo/chat_flow_test.go).

**Итог:** проверены сценарии: водитель (регистрация → профиль → geo, transport-type); диспетчер (регистрация, login/password → профиль); **cargo** (диспетчер создаёт груз → водитель list/get → водитель оффер → диспетчер accept; отказ/два оффера); **chat** (conversations, messages между водителем и диспетчером). Остальной API (auth по OTP, refresh, logout, admin, company TZ, reference, KYC, смена пароля/телефона, chat presence/ws) в тестах не покрыт.
