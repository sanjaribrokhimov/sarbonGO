-- Ограничить company_type только: Shipper, Carrier, Broker.
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_type_check;

-- Привести существующие недопустимые значения к NULL (или к Shipper), чтобы новый CHECK прошёл
UPDATE companies
SET company_type = NULL
WHERE company_type IS NOT NULL AND TRIM(company_type) <> ''
  AND company_type NOT IN ('Shipper', 'Carrier', 'Broker');

ALTER TABLE companies ADD CONSTRAINT companies_type_check
  CHECK (
    company_type IS NULL OR company_type = ''
    OR company_type IN ('Shipper', 'Carrier', 'Broker')
  );
