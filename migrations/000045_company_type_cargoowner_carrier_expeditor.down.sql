ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_type_check;

-- Вернуть предыдущее ограничение (Shipper, Carrier, Broker)
ALTER TABLE companies ADD CONSTRAINT companies_type_check
  CHECK (
    company_type IS NULL OR company_type = ''
    OR company_type IN ('Shipper', 'Carrier', 'Broker')
  );

