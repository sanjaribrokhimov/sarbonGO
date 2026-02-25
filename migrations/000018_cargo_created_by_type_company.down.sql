-- Вернуть CHECK только admin/dispatcher (как в 000017).

DO $$
DECLARE
  cname text;
BEGIN
  FOR cname IN
    SELECT conname FROM pg_constraint
    WHERE conrelid = 'cargo'::regclass AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%created_by_type%'
  LOOP
    EXECUTE format('ALTER TABLE cargo DROP CONSTRAINT %I', cname);
  END LOOP;
  ALTER TABLE cargo ADD CONSTRAINT cargo_created_by_type_check
    CHECK (created_by_type IS NULL OR created_by_type IN ('admin', 'dispatcher'));
END$$;
