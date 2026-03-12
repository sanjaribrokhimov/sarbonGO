# Отчёт: единая структура ответа API и локализация description по X-Language

## Что сделано

### 1. Структура ответа (уже была)
Во всех ответах используется общая обёртка:
- **status** — всегда на английском: `"success"` или `"error"`
- **code** — HTTP-код
- **description** — человекочитаемое описание (теперь переводится по заголовку **X-Language**)
- **data** — тело ответа или `null`

### 2. Локализация description (ru, uz, en, tr, zh)
- Добавлен пакет **internal/server/resp/i18n.go**: словарь ключей сообщений и переводы на 5 языков.
- В **internal/server/resp/resp.go** добавлены:
  - **OKLang(c, messageKey, data)** — успех 200, description по ключу и X-Language
  - **SuccessLang(c, httpCode, messageKey, data)** — успех с произвольным кодом
  - **ErrorLang(c, httpCode, messageKey)** — ошибка, description по ключу и X-Language
  - **Lang(c)** — чтение X-Language из запроса (по умолчанию `"en"`)

### 3. Обработанные API (переведены на OKLang / ErrorLang / SuccessLang)

| Группа / файл | Эндпоинты | Статус |
|---------------|-----------|--------|
| **Middleware** (mw/headers.go) | Проверка X-Device-Type, X-Language, X-Client-Token | ✅ ErrorLang |
| **Driver auth** (auth.go) | SendOTP, VerifyOTP, Refresh, Logout | ✅ OKLang, ErrorLang |
| **Driver profile** (profile.go) | Get, PatchDriver, Heartbeat, PhoneChange, PatchPower, PatchTrailer, Delete, UploadPhoto, GetPhoto, DeletePhoto | ✅ OKLang, ErrorLang |
| **Admin auth** (admin_auth.go) | LoginPassword | ✅ OKLang, ErrorLang |
| **Dispatcher profile** (dispatcher_profile.go) | Get, Patch, ChangePassword, PhoneChange, UploadPhoto, GetPhoto, Delete | ✅ OKLang, ErrorLang |
| **Dispatcher auth** (dispatcher_auth.go) | SendOTP, VerifyOTP, Complete, LoginPassword, ResetPassword, Refresh, Logout | ✅ OKLang, ErrorLang |
| **Registration** (registration.go) | NameOferta, GeoPush, TransportType | ✅ OKLang, ErrorLang |
| **Reference** (reference.go) | GetReferenceDrivers, Admin, Dispatchers, Cargo, Company, Cities, Countries | ✅ OKLang, ErrorLang |
| **Cargo** (cargo.go) | List, Get, Create, Update, Delete, PatchStatus, ListOffers, AcceptOffer | ✅ OKLang, ErrorLang |
| **Trips** (trips.go) | Get, List, AssignDriver, DriverConfirm, DriverReject, PatchStatus | ✅ OKLang, ErrorLang |
| **Company TZ** (company_tz.go) | CreateCompany, ListCompanies, SwitchCompany, CreateInvitation, AcceptInvitation, ListUsers, UpdateRole, RemoveUser | ✅ OKLang, ErrorLang, SuccessLang |

### 4. Дополнительно переведённые API (все ответы по X-Language)

| Группа / файл | Эндпоинты | Статус |
|---------------|-----------|--------|
| dispatcher_registration.go | Complete (логин/регистрация диспетчера) | ✅ OKLang, ErrorLang |
| otp_send.go | WriteOTPSendError (ошибки OTP) | ✅ ErrorLang (internal_error / invalid_payload / otp_rate_limited) |
| admin_companies.go | CreateCompany, SetOwner, SearchOwners | ✅ OKLang, ErrorLang |
| company_user_auth.go | SendOTP, VerifyOTP (company users) | ✅ OKLang, ErrorLang |
| dispatcher_companies.go | CreateCompany, SwitchCompany | ✅ OKLang, ErrorLang, SuccessLang |
| driver_invitations.go | Create, List, ListMy, Unlink, SetDriverPower, SetDriverTrailer, Cancel, Accept, Decline, ListInvitations | ✅ OKLang, ErrorLang, SuccessLang |
| kyc.go | Submit KYC | ✅ OKLang, ErrorLang |
| chat.go | ListConversations, GetOrCreateConversation, ListMessages, SendMessage, EditMessage, DeleteMessage, GetPresence | ✅ OKLang, ErrorLang |
| company_user_registration.go | Complete (регистрация пользователя компании) | ✅ OKLang, ErrorLang |
| dispatcher_invitations.go | Create, Accept, Decline | ✅ SuccessLang, ErrorLang |
| mw/auth.go, mw/admin.go, mw/app_user.go | Ответы при невалидном/отсутствующем JWT | ✅ ErrorLang (missing_user_token, invalid_user_token, missing_user_token_or_id) |
| router.go | Health/ok | ✅ OKLang |
| swaggerui/swaggerui.go | 404 openapi.yaml | ✅ ErrorLang (openapi_not_found) |

**Итог:** во всех API ответы используют единую структуру; **status** всегда на английском (`success` | `error`), **description** — по заголовку **X-Language** (ru, uz, en, tr, zh).

### 5. Использование

- Клиент передаёт заголовок **X-Language** со значением: `ru`, `uz`, `en`, `tr` или `zh`.
- В ответе поле **description** всегда приходит на выбранном языке (при отсутствии ключа или языка возвращается ключ или fallback en → ru).
- **status** в JSON всегда на английском: `"success"` или `"error"`.

Пример ответа (X-Language: tr):
```json
{
  "status": "success",
  "code": 200,
  "description": "Başarılı",
  "data": { ... }
}
```

Пример ошибки (X-Language: ru):
```json
{
  "status": "error",
  "code": 400,
  "description": "Некорректные данные запроса",
  "data": null
}
```
