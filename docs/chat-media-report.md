## Chat: медиа (фото / голос / геолокация / видео / video-note) — отчёт

### 1. Цель

Добавить в существующий чат поддержку:
- **Фото**
- **Голосовых сообщений** (с транскодированием «как Telegram» → OGG/Opus)
- **Видео**
- **Кружков (video note)**
- **Геолокации**

С учётом требования «быстро и экономно» (есть только сервер + Postgres) выбран подход:
- **файлы храним на диске сервера**;
- **в Postgres храним только метаданные** и связь с сообщениями;
- для voice/video делаем **оптимизацию через ffmpeg**.

---

### 2. Изменения в БД (миграция)

Добавлена миграция:
- `migrations/000046_chat_attachments_and_message_types.up.sql`
- `migrations/000046_chat_attachments_and_message_types.down.sql`
 - `migrations/000047_chat_media_files_dedup.up.sql`
 - `migrations/000047_chat_media_files_dedup.down.sql`

Что делает `up`:
- в `chat_messages`:
  - `body` стал nullable (для не-текстовых сообщений),
  - добавлены поля:
    - `type` (TEXT/PHOTO/VOICE/VIDEO/VIDEO_NOTE/LOCATION),
    - `payload` (JSONB, type-specific).
- создана таблица `chat_attachments`:
  - хранит метаданные вложений (kind, mime, size, path, thumb_path, duration, width/height),
  - `message_id` связывается после создания сообщения.
 - создана таблица `media_files`:
   - хранит уникальные файлы по `content_hash` (sha256),
   - используется для дедупликации медиа (один файл — много сообщений).
 - в `chat_attachments` добавлены ссылки:
   - `media_file_id`, `thumb_media_file_id`.

---

### 3. Изменения в коде (чат)

#### 3.1 Модель сообщения

Файл: `internal/chat/model.go`
- `Message.Body` → `*string` (опционально)
- добавлены поля:
  - `Message.Type`
  - `Message.Payload` (Raw JSON)

#### 3.2 Репозиторий

Файл: `internal/chat/repo.go`
- `CreateTextMessage` — создание текстового сообщения.
- `CreateMessage` — универсальное создание сообщения с `type/body/payload`.
- `ListMessages`, `GetMessageByID`, `UpdateMessage` — обновлены под новые поля.
- Добавлены методы для вложений:
  - `CreateAttachment`
  - `LinkAttachment`
  - `GetAttachmentForUser` (проверка доступа: пользователь должен быть участником диалога)

#### 3.3 Handler’ы и маршруты

Файл: `internal/server/router.go`
- добавлены маршруты:
  - `POST /v1/chat/conversations/:id/messages/media`
  - `GET /v1/chat/files/:id`

Файл: `internal/server/handlers/chat.go`
- `POST /v1/chat/conversations/:id/messages` теперь создаёт текст через `CreateTextMessage`.
- Новый handler **`SendMediaMessage`**:
  - принимает `multipart/form-data`:
    - `type` = PHOTO/VOICE/VIDEO/VIDEO_NOTE
    - `body` (опционально, подпись)
    - `file` (binary)
  - сохраняет оригинал во временный файл,
  - оптимизирует через ffmpeg:
    - VOICE → OGG/Opus (моно, 48kHz, ~32kbps)
    - VIDEO → MP4 (H.264/AAC, faststart) + thumbnail
    - VIDEO_NOTE → квадратный MP4 (под UI “кружок”) + thumbnail
    - PHOTO → JPEG (оптимизация)
  - создаёт запись `chat_attachments`,
  - создаёт сообщение `chat_messages` с `type` и `payload`, где есть:
    - `attachment_id`
    - `url` = `/v1/chat/files/{attachment_id}`
    - `thumb_url` (если доступно)
- Новый handler **`GetFile`**:
  - отдаёт файл вложения по `attachment_id`,
  - проверяет права через `GetAttachmentForUser`,
  - поддерживает `?thumb=1` для превью.

---

### 4. Swagger / OpenAPI

Файл: `docs/openapi.yaml`

Добавлено:
- новые схемы:
  - `ChatMessage`
  - `ChatMediaPayload`
  - `ChatLocationPayload`
  - `ChatPresence`
- расширена документация Chat:
  - новый endpoint `POST /v1/chat/conversations/{id}/messages/media` (multipart, примеры)
  - новый endpoint `GET /v1/chat/files/{id}` (+ thumb=1)
  - ответы обновлены на `ChatMessage`/`ChatPresence` вместо “Message”.

---

### 5. Как тестировать (быстро)

1) Применить миграции до `000046`.
2) Подключиться к Swagger (`GET /docs`) и вызвать:
   - `POST /v1/chat/conversations` → получить `conversation_id`
   - `POST /v1/chat/conversations/{id}/messages/media` → отправить voice/video/photo
   - `GET /v1/chat/conversations/{id}/messages` → увидеть сообщение с `type` и `payload.url`
   - открыть `payload.url` (это `/v1/chat/files/{attachment_id}`) → скачать файл

---

### 6. Примечания

- Геолокация (`LOCATION`) логически поддержана схемами (`ChatLocationPayload`), но для неё можно добавить отдельный JSON-эндпоинт или расширить текущий text endpoint, чтобы принимал `type=LOCATION` и `payload`.
- Для продакшена рекомендуется вынести `CHAT_STORAGE_DIR` в конфиг (сейчас берётся из env, fallback: `storage`).
 - Для максимальной производительности включите `CHAT_USE_X_ACCEL=1` и настройте Nginx `X-Accel-Redirect` (см. `docs/nginx-chat-media.conf.md`).
 - Для очистки диска включите GC:
   - `CHAT_MEDIA_GC_ENABLED=1`
   - `CHAT_MEDIA_GC_DAYS=30` (или другое значение)

