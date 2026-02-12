CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    language TEXT NOT NULL,
    components TEXT NOT NULL, -- JSON
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE
);

CREATE INDEX idx_templates_instance_id ON templates(instance_id);
CREATE UNIQUE INDEX idx_templates_name_lang ON templates(instance_id, name, language);
