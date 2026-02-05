# pgAdmin — управление PostgreSQL

После `docker-compose up -d` pgAdmin доступен по адресу: **http://localhost:5050** (порт можно изменить в .env: `PGADMIN_PORT`).

**Важно:** pgAdmin поднимается 1–2 минуты. Если страница не открывается — подожди и обнови.

## Если http://localhost:5050 не открывается

1. **Подожди 1–2 минуты** после `docker-compose up -d`.
2. Проверь, что контейнер запущен:
   ```bash
   docker-compose ps
   ```
   У `pgadmin` должно быть состояние `Up`.
3. Посмотри логи (ошибки при старте):
   ```bash
   docker-compose logs pgadmin
   ```
4. Перезапусти только pgAdmin:
   ```bash
   docker-compose restart pgadmin
   ```
   Снова подожди 1–2 минуты и открой http://localhost:5050.

## Первый вход

1. Открой в браузере: http://localhost:5050  
2. **Email:** значение из `PGADMIN_DEFAULT_EMAIL` (по умолчанию `admin@example.com`)  
3. **Password:** значение из `PGADMIN_DEFAULT_PASSWORD` (по умолчанию `admin`)

## Подключение к БД Sarbon (чтобы слева появилась база)

1. На экране приветствия нажми **Add New Server** (или правый клик по **Servers** в дереве слева → **Register** → **Server**).
2. Вкладка **General:** в поле **Name** введи `Sarbon` (любое имя для отображения).
3. Вкладка **Connection:**
   - **Host name/address:** `postgres` (имя сервиса в Docker; не localhost)
   - **Port:** `5432`
   - **Username:** `sarbon`
   - **Password:** из файла `deploy/docker/.env` — переменная `POSTGRES_PASSWORD` (по умолчанию `sarbon`). Поставь галочку **Save password**, чтобы не вводить каждый раз.
4. Нажми **Save**. В дереве слева под **Servers** появится **Sarbon** → внутри будет база **sarbon** и таблицы.


docker-compose down
docker-compose stop

docker-compose up -d

docker-compose restart app

docker-compose up -d --build app

docker-compose logs -f app

docker-compose ps


Остановить всё	: docker-compose down
Запустить всё:	docker-compose up -d
Перезапустить app:	docker-compose restart app
Пересобрать app после изменений кода:	docker-compose up -d --build app
Логи app:	docker-compose logs -f app
Статус:	docker-compose ps

