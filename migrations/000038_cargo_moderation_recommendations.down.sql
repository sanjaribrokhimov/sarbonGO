DROP TABLE IF EXISTS cargo_driver_recommendations;
ALTER TABLE offers DROP COLUMN IF EXISTS rejection_reason;
ALTER TABLE cargo DROP COLUMN IF EXISTS moderation_rejection_reason;
