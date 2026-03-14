-- Allow new cargo statuses: pending_moderation, rejected, in_progress, completed (from 000038).
-- Drop the old CHECK that only allowed created, searching, assigned, in_transit, delivered, cancelled.

DO $$
DECLARE
  conname text;
BEGIN
  SELECT c.conname INTO conname
  FROM pg_constraint c
  JOIN pg_class t ON c.conrelid = t.oid
  WHERE t.relname = 'cargo'
    AND c.contype = 'c'
    AND pg_get_constraintdef(c.oid) LIKE '%status%';
  IF conname IS NOT NULL THEN
    EXECUTE format('ALTER TABLE cargo DROP CONSTRAINT %I', conname);
  END IF;
END$$;

ALTER TABLE cargo ADD CONSTRAINT cargo_status_check CHECK (
  status IN (
    'created',
    'pending_moderation',
    'rejected',
    'searching',
    'assigned',
    'in_transit',
    'delivered',
    'in_progress',
    'completed',
    'cancelled'
  )
);
