# Sarbon (Gin + Postgres + Redis)

## Run locally

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

## Notes

- Only one DB table is used: `drivers` (see `migrations/`).
- OTP is sent via Telegram Gateway API (configure `TELEGRAM_GATEWAY_*`).

