-- Add additional columns to users table
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS first_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS last_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(255),
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'admin'));

-- Create index on role for efficient filtering
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
