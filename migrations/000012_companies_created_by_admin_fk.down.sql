DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_companies_created_by_admins') THEN
    ALTER TABLE companies DROP CONSTRAINT fk_companies_created_by_admins;
  END IF;
END$$;

