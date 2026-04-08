CREATE TABLE IF NOT EXISTS media (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    object_key VARCHAR(500) NOT NULL,
    bucket_name VARCHAR(100) NOT NULL DEFAULT 'user-uploads',
    original_filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_user_id ON media(user_id);
CREATE INDEX IF NOT EXISTS idx_media_object_key ON media(object_key);
CREATE INDEX IF NOT EXISTS idx_media_created_at ON media(created_at);
