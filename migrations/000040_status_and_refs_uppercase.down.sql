-- Revert constraints to allow lowercase (data remains UPPERCASE; app must support chosen convention).
ALTER TABLE cargo DROP CONSTRAINT IF EXISTS cargo_status_check;
ALTER TABLE offers DROP CONSTRAINT IF EXISTS offers_status_check;
ALTER TABLE trips DROP CONSTRAINT IF EXISTS trips_status_check;
