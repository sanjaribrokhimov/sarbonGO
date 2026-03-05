-- Добавить last_online_at для диспетчеров (обновляется при каждом запросе, как у водителей).

ALTER TABLE freelance_dispatchers
  ADD COLUMN IF NOT EXISTS last_online_at TIMESTAMP NULL;

ALTER TABLE deleted_freelance_dispatchers
  ADD COLUMN IF NOT EXISTS last_online_at TIMESTAMP NULL;
