-- Add board-related columns to tasks table
ALTER TABLE tasks
    ADD COLUMN column_id INTEGER REFERENCES columns(id),
    ADD COLUMN "order" INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN priority VARCHAR(10) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    ADD COLUMN assignee_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN deadline TIMESTAMP,
    ADD COLUMN estimated_time INTEGER DEFAULT 0,
    ADD COLUMN tracked_time INTEGER DEFAULT 0,
    ADD COLUMN tags TEXT[] DEFAULT '{}',
    ADD COLUMN created_by INTEGER REFERENCES users(id);

-- Assign existing tasks to default Backlog column
UPDATE tasks SET
    column_id = (SELECT id FROM columns WHERE title = 'Backlog' LIMIT 1),
    created_by = user_id
WHERE column_id IS NULL;

-- Make column_id required after migration
ALTER TABLE tasks ALTER COLUMN column_id SET NOT NULL;

-- Create indexes for performance
CREATE INDEX idx_tasks_column_id ON tasks(column_id);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_deadline ON tasks(deadline);
CREATE INDEX idx_tasks_order ON tasks("order");
