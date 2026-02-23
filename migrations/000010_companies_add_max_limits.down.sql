ALTER TABLE companies
  DROP COLUMN IF EXISTS max_vehicles,
  DROP COLUMN IF EXISTS max_drivers,
  DROP COLUMN IF EXISTS max_top_dispatchers;

