-- Driver can invite dispatcher (by phone). Dispatcher accepts/declines; on accept driver.freelancer_id = dispatcher_id.

CREATE TABLE IF NOT EXISTS driver_to_dispatcher_invitations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  token VARCHAR(64) UNIQUE NOT NULL,
  driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
  dispatcher_phone VARCHAR(32) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_d2d_invitations_token ON driver_to_dispatcher_invitations (token);
CREATE INDEX IF NOT EXISTS idx_d2d_invitations_driver ON driver_to_dispatcher_invitations (driver_id);
CREATE INDEX IF NOT EXISTS idx_d2d_invitations_dispatcher_phone ON driver_to_dispatcher_invitations (replace(replace(replace(trim(dispatcher_phone), ' ', ''), '-', ''), '+', ''));
