-- Write your migrate up statements here
CREATE TABLE IF NOT EXISTS tasker.todo_categories(
    id uuid DEFAULT gen_random_uuid(),
    user_id varchar(128) NOT NULL,
    name varchar(32) NOT NULL,
    description text,
    metadata JSONB,
    is_deleted boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_todo_categories PRIMARY KEY (id),
    CONSTRAINT uniq_category_user_id_name UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_todo_categories_user_id ON tasker.todo_categories(user_id);
CREATE TRIGGER set_updated_at_todo_categories
    BEFORE UPDATE ON tasker.todo_categories
    FOR EACH ROW
    EXECUTE FUNCTION tasker.trigger_set_updated_at();
---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
