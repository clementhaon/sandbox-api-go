-- Drop indexes
DROP INDEX IF EXISTS idx_tasks_column_id;
DROP INDEX IF EXISTS idx_tasks_assignee_id;
DROP INDEX IF EXISTS idx_tasks_priority;
DROP INDEX IF EXISTS idx_tasks_deadline;
DROP INDEX IF EXISTS idx_tasks_order;

-- Remove board-related columns from tasks table
ALTER TABLE tasks
    DROP COLUMN IF EXISTS column_id,
    DROP COLUMN IF EXISTS "order",
    DROP COLUMN IF EXISTS priority,
    DROP COLUMN IF EXISTS assignee_id,
    DROP COLUMN IF EXISTS deadline,
    DROP COLUMN IF EXISTS estimated_time,
    DROP COLUMN IF EXISTS tracked_time,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS created_by;
