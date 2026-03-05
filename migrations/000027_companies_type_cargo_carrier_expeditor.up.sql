-- Add company types: CargoOwner (грузовладелец), Carrier (перевозчик), Expeditor (экспедитор).

ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_type_check;

ALTER TABLE companies ADD CONSTRAINT companies_type_check
  CHECK (
    company_type IS NULL OR company_type = ''
    OR company_type IN (
      'Shipper', 'Broker', 'Fleet', 'OwnerOperator',
      'CargoOwner', 'Carrier', 'Expeditor'
    )
  );
