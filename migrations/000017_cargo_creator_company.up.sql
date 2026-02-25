-- Кто создал груз и от какой компании (диспетчер/админ + опционально компания).

ALTER TABLE cargo
  ADD COLUMN IF NOT EXISTS created_by_type VARCHAR NULL CHECK (created_by_type IN ('admin', 'dispatcher')),
  ADD COLUMN IF NOT EXISTS created_by_id UUID NULL,
  ADD COLUMN IF NOT EXISTS company_id UUID NULL;

CREATE INDEX IF NOT EXISTS idx_cargo_created_by_id ON cargo (created_by_id) WHERE created_by_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cargo_company_id ON cargo (company_id) WHERE company_id IS NOT NULL;

-- company_id можно привязать к companies
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cargo_company_id') THEN
    ALTER TABLE cargo ADD CONSTRAINT fk_cargo_company_id
      FOREIGN KEY (company_id) REFERENCES companies(id);
  END IF;
END$$;
