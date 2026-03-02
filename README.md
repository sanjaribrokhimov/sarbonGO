# Sarbon (Gin + Postgres + Redis)

Stylish insurance page: open `docs/insurance.html` in browser (see below).

go run ./cmd/admin -login admin -password "Secret123" -name "Main Admin"

## Run locally
http://api.sarbon.me/docs
1) Start infra:

```bash
docker compose up -d
```

2) Configure env:

```bash
cp .env.example .env
```

3) Apply migrations:

```bash
go run ./cmd/migrate -direction up
```

4) Run API:

```bash
go run ./cmd/api
```

API: `http://localhost:8080`  
Swagger UI: `http://localhost:8080/docs`

## Run without Docker

Установи локально **PostgreSQL** и **Redis**, затем проверь доступ:

**PostgreSQL** (в `.env`: `DATABASE_URL=postgres://sarbon:sarbon@localhost:5432/sarbon?sslmode=disable`):

```bash
# Проверка: подключение к БД (пароль: sarbon)
psql -h localhost -p 5432 -U sarbon -d sarbon -c "SELECT 1"
```

Если БД или пользователь ещё не созданы:

```bash
psql -h localhost -p 5432 -U postgres -c "CREATE USER sarbon WITH PASSWORD 'sarbon';"
psql -h localhost -p 5432 -U postgres -c "CREATE DATABASE sarbon OWNER sarbon;"
```

**Redis** (в `.env`: `REDIS_ADDR=localhost:6379`):

```bash
# Проверка
redis-cli ping
# Ожидается: PONG
```

Дальше как с Docker: настрой `.env`, выполни миграции и запусти API:

```bash
cp .env.example .env   # и поправь DATABASE_URL / REDIS_ADDR при необходимости
go run ./cmd/api       # миграции применятся при старте
```

Проверка API: `curl http://localhost:8080/health` → `{"status":"ok",...}`

## Notes

- Only one DB table is used: `drivers` (see `migrations/`).
- OTP is sent via Telegram Gateway API (configure `TELEGRAM_GATEWAY_*`).

