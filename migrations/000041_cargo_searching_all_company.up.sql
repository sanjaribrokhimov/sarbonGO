-- Split SEARCHING into SEARCHING_ALL (visible to all) and SEARCHING_COMPANY (visible only to company drivers).
-- Companies can choose visibility when admin accepts; freelance dispatchers always get SEARCHING_ALL.

DO $$
DECLARE conname text;
BEGIN
  SELECT c.conname INTO conname FROM pg_constraint c JOIN pg_class t ON c.conrelid = t.oid
  WHERE t.relname = 'cargo' AND c.contype = 'c' AND pg_get_constraintdef(c.oid) LIKE '%status%';
  IF conname IS NOT NULL THEN EXECUTE format('ALTER TABLE cargo DROP CONSTRAINT %I', conname); END IF;
END$$;

UPDATE cargo SET status = 'SEARCHING_ALL' WHERE status = 'SEARCHING';

ALTER TABLE cargo ADD CONSTRAINT cargo_status_check CHECK (status IN (
  'CREATED', 'PENDING_MODERATION', 'REJECTED', 'SEARCHING_ALL', 'SEARCHING_COMPANY', 'ASSIGNED', 'IN_TRANSIT', 'DELIVERED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'
));
