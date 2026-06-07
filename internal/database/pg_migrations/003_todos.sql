-- Write your migrate up statements here
CREATE TABLE IF NOT EXISTS tasker.todos (
    id UUID,
    user_id VARCHAR(128) NOT NULL,
    category_id UUID,
    title VARCHAR(128) NOT NULL,
    description TEXT,
    status VARCHAR(32) NOT NULL,
    priority INTEGER NOT NULL,
    due_date TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    parent_id uuid,
    metadata JSONB,
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '')) || 
        to_tsvector('english', coalesce(description, ''))
    ) STORED,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT pk_todos PRIMARY KEY (id),
    CONSTRAINT chk_todo_no_self_parent CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT fk_todo_category FOREIGN KEY (category_id) REFERENCES tasker.todo_categories(id) ON DELETE SET NULL,
    CONSTRAINT fk_todo_parent FOREIGN KEY (parent_id) REFERENCES tasker.todos(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_todos_user_id ON tasker.todos(user_id);
CREATE INDEX IF NOT EXISTS idx_todos_category_id ON tasker.todos(category_id);
CREATE INDEX IF NOT EXISTS idx_todos_status ON tasker.todos(status);
CREATE INDEX IF NOT EXISTS idx_todos_search_vector ON tasker.todos USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_todos_parent_id ON tasker.todos(parent_id);
CREATE INDEX IF NOT EXISTS idx_todos_user_id_is_deleted ON tasker.todos(user_id, is_deleted);

CREATE TRIGGER set_updated_at_todos
    BEFORE UPDATE ON tasker.todos
    FOR EACH ROW
    EXECUTE FUNCTION tasker.trigger_set_updated_at();

---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
