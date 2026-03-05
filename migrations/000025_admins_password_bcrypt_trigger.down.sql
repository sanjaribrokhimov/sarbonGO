DROP TRIGGER IF EXISTS admins_before_save_password ON admins;
DROP FUNCTION IF EXISTS admins_hash_password_trigger();
