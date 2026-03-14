-- Restore original cargo status check (from 000015).

ALTER TABLE cargo DROP CONSTRAINT IF EXISTS cargo_status_check;

ALTER TABLE cargo ADD CONSTRAINT cargo_status_check CHECK (
  status IN ('created', 'searching', 'assigned', 'in_transit', 'delivered', 'cancelled')
);
