-- All status and enum values in API/DB: UPPERCASE only.
-- Order: DROP constraint first, then UPDATE data, then ADD new constraint (else UPDATE would violate old check).

-- 1) Cargo: drop status and created_by_type checks, update data, re-add checks
DO $$
DECLARE conname text;
BEGIN
  -- drop status check
  SELECT c.conname INTO conname FROM pg_constraint c JOIN pg_class t ON c.conrelid = t.oid
  WHERE t.relname = 'cargo' AND c.contype = 'c' AND pg_get_constraintdef(c.oid) LIKE '%status%';
  IF conname IS NOT NULL THEN EXECUTE format('ALTER TABLE cargo DROP CONSTRAINT %I', conname); END IF;
  -- drop created_by_type check (so UPDATE to UPPERCASE does not violate it)
  SELECT c.conname INTO conname FROM pg_constraint c JOIN pg_class t ON c.conrelid = t.oid
  WHERE t.relname = 'cargo' AND c.contype = 'c' AND pg_get_constraintdef(c.oid) LIKE '%created_by_type%';
  IF conname IS NOT NULL THEN EXECUTE format('ALTER TABLE cargo DROP CONSTRAINT %I', conname); END IF;
END$$;

UPDATE cargo SET status = UPPER(status) WHERE status IS NOT NULL AND status != UPPER(status);
UPDATE cargo SET created_by_type = UPPER(created_by_type) WHERE created_by_type IS NOT NULL AND created_by_type != UPPER(created_by_type);

ALTER TABLE cargo ADD CONSTRAINT cargo_status_check CHECK (status IN (
  'CREATED', 'PENDING_MODERATION', 'REJECTED', 'SEARCHING', 'ASSIGNED', 'IN_TRANSIT', 'DELIVERED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'
));
ALTER TABLE cargo ADD CONSTRAINT cargo_created_by_type_check CHECK (
  created_by_type IS NULL OR created_by_type IN ('ADMIN', 'DISPATCHER', 'COMPANY')
);

-- 2) Offers: drop check, update data, re-add check
DO $$
DECLARE conname text;
BEGIN
  SELECT c.conname INTO conname FROM pg_constraint c JOIN pg_class t ON c.conrelid = t.oid
  WHERE t.relname = 'offers' AND c.contype = 'c' AND pg_get_constraintdef(c.oid) LIKE '%status%';
  IF conname IS NOT NULL THEN EXECUTE format('ALTER TABLE offers DROP CONSTRAINT %I', conname); END IF;
END$$;
UPDATE offers SET status = UPPER(status) WHERE status IS NOT NULL;
ALTER TABLE offers ADD CONSTRAINT offers_status_check CHECK (status IN ('PENDING', 'ACCEPTED', 'REJECTED'));

-- 3) Trips: drop check, update data, re-add check
ALTER TABLE trips DROP CONSTRAINT IF EXISTS trips_status_check;
UPDATE trips SET status = CASE
  WHEN status = 'pending_driver' THEN 'PENDING_DRIVER'
  WHEN status = 'assigned' THEN 'ASSIGNED'
  WHEN status = 'loading' THEN 'LOADING'
  WHEN status = 'en_route' THEN 'EN_ROUTE'
  WHEN status = 'unloading' THEN 'UNLOADING'
  WHEN status = 'completed' THEN 'COMPLETED'
  WHEN status = 'cancelled' THEN 'CANCELLED'
  ELSE status
END WHERE status IS NOT NULL;
ALTER TABLE trips ADD CONSTRAINT trips_status_check CHECK (status IN (
  'PENDING_DRIVER', 'ASSIGNED', 'LOADING', 'EN_ROUTE', 'UNLOADING', 'COMPLETED', 'CANCELLED'
));

-- 4) Cargo driver recommendations: status
UPDATE cargo_driver_recommendations SET status = UPPER(status) WHERE status IS NOT NULL AND status != UPPER(status);
