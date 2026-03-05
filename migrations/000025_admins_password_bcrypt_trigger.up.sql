-- Hash plain-text passwords on admins when saved from GoAdmin (pgcrypto bcrypt).
-- API and cmd/admin already store bcrypt; plain text is only from GoAdmin form.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION admins_hash_password_trigger()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
  -- If password looks like bcrypt ($2a$, $2b$, etc.), leave as is (from API/cmd/admin).
  IF NEW.password IS NOT NULL AND length(trim(NEW.password)) > 0 AND NEW.password NOT LIKE '$2%' THEN
    NEW.password := crypt(NEW.password, gen_salt('bf'));
  END IF;
  -- On update, empty string usually means "don't change"; restore old hash.
  IF TG_OP = 'UPDATE' AND (NEW.password IS NULL OR trim(NEW.password) = '') THEN
    NEW.password := OLD.password;
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS admins_before_save_password ON admins;
CREATE TRIGGER admins_before_save_password
  BEFORE INSERT OR UPDATE OF password ON admins
  FOR EACH ROW
  EXECUTE PROCEDURE admins_hash_password_trigger();
