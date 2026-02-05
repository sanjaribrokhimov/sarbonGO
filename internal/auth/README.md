# Auth Driver — OTP через Telegram Gateway

## Принципы

- **Login и Register не разделяются** — один flow.
- Единственный идентификатор пользователя — **номер телефона**.
- OTP подтверждает номер, а не регистрацию.
- Токен выдаётся **только после успешной проверки OTP**.
- После OTP токен выдаётся всегда (существующий или новый пользователь).
- Пользовательские данные (drivers) **не создаются до подтверждения OTP**.
- OTP и токены не логируются.
- OTP хранится в таблице **otp_codes**, не в users/drivers.

## Сущности

- **users**: id, phone (unique), role=driver, status (new | active | blocked), created_at
- **otp_codes**: id, phone, code, expires_at, used_at, attempts_count
- **auth_tokens**: id, user_id, access_token, refresh_token, expires_at
- **drivers**: id, user_id, first_name, last_name, passport_series, passport_number, rating, account_status, language, platform, dispatcher_type, created_at, updated_at, deleted_at

## API

1. **POST /api/v1/auth/otp/send** — тело `{ "phone": "+..." }`. Генерация OTP, сохранение в БД (TTL 3–5 мин), сброс старых OTP по номеру, отправка через Telegram Gateway. Ответ не раскрывает существование пользователя.
2. **POST /api/v1/auth/otp/verify** — тело `{ "phone", "code" }`. Валидация OTP; при успехе — поиск/создание user, выдача access_token и refresh_token; в ответе `is_new: true/false`.
3. **PUT /api/v1/drivers/profile** — заголовок `Authorization: Bearer <access_token>`, тело — профиль водителя. Только для role=driver. Создание/обновление driver, привязка к user; user.status = active, driver.account_status = pending.

## Безопасность

- OTP TTL не более 5 минут.
- Максимум 3–5 попыток ввода кода.
- Rate limit на отправку OTP по номеру.
- Заблокированные пользователи не проходят verify.
- Токены ротируются при повторном логине (старые refresh удаляются).
