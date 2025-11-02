-- Rollback user columns
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_role;

ALTER TABLE users
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS first_name;
