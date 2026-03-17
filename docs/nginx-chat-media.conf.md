## Nginx: X-Accel-Redirect для chat media

### Цель

Сделать отдачу файлов чата максимально быстрой и дешёвой по ресурсам:
- Go проверяет доступ (участник диалога),
- Nginx отдаёт файл (Range, кеш, буферизация).

---

### 1) Переменные окружения для API

- `CHAT_USE_X_ACCEL=1`
- `CHAT_X_ACCEL_PREFIX=/_protected` (можно не задавать — это default)
- `CHAT_STORAGE_DIR=storage` (default: `storage`)

API будет отвечать:
- `X-Accel-Redirect: /_protected/storage/...`
- `Cache-Control: public, max-age=31536000, immutable`
- `ETag: "<sha256>"`

---

### 2) Пример конфигурации Nginx

Добавь в конфиг сервера:

```nginx
# Важно: internal location — нельзя дергать напрямую снаружи.
location /_protected/ {
    internal;
    # Путь к корню проекта, где лежит папка storage.
    # Пример: /var/www/sarbonGO/
    alias /var/www/sarbonGO/;

    # Range requests (видео), отдача больших файлов
    aio on;
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;

    # Кеширование
    expires 365d;
    add_header Cache-Control "public, max-age=31536000, immutable" always;
    add_header Accept-Ranges "bytes" always;
}

# Публичный API (проксирование на Go)
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

> `alias` должен указывать на директорию, которая содержит папку `storage/`.
> Если проект лежит в другом месте — меняешь `alias`.

---

### 3) Как это работает в рантайме

1. Клиент вызывает `GET /v1/chat/files/{attachment_id}`.
2. Go:
   - проверяет, что пользователь участник диалога,
   - выставляет `X-Accel-Redirect`.
3. Nginx:
   - отдаёт файл напрямую,
   - поддерживает Range, caching и fast download.

