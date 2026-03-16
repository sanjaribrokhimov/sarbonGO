DO $$
DECLARE conname text;
BEGIN
  SELECT c.conname INTO conname FROM pg_constraint c
  JOIN pg_class t ON c.conrelid = t.oid
  WHERE t.relname = 'route_points' AND c.contype = 'c' AND pg_get_constraintdef(c.oid) LIKE '%type%';
  IF conname IS NOT NULL THEN
    EXECUTE format('ALTER TABLE route_points DROP CONSTRAINT %I', conname);
  END IF;
END$$;

UPDATE route_points SET type = LOWER(type) WHERE type IS NOT NULL;

ALTER TABLE route_points ADD CONSTRAINT route_points_type_check CHECK (type IN ('load', 'unload', 'customs', 'transit'));
