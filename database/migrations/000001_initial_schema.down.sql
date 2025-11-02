-- Rollback initial schema
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_tasks_user_id;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS users;
