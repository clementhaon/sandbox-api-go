-- Create columns table for board
CREATE TABLE columns (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    "order" INTEGER NOT NULL DEFAULT 0,
    color VARCHAR(7) DEFAULT '#2196F3',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_columns_order ON columns("order");

-- Insert default column for existing tasks
INSERT INTO columns (title, "order", color) VALUES ('Backlog', 0, '#9E9E9E');
