-- Разрешить created_by_type = 'company' (кто создал груз: admin, dispatcher или company).

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
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'cargo'::regclass AND conname = 'cargo_created_by_type_check') THEN
    ALTER TABLE cargo ADD CONSTRAINT cargo_created_by_type_check
      CHECK (created_by_type IS NULL OR created_by_type IN ('admin', 'dispatcher', 'company'));
  END IF;
END$$;
