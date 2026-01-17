ALTER TABLE instances
    ADD COLUMN history_sync_status TEXT NOT NULL DEFAULT 'pending';

ALTER TABLE instances
    ADD COLUMN history_sync_cycle_id TEXT;

ALTER TABLE instances
    ADD COLUMN history_sync_updated_at TEXT;

CREATE TABLE IF NOT EXISTS whatsapp_history_syncs (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    payload_type TEXT NOT NULL,
    payload BLOB NOT NULL,
    cycle_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    processed_at TEXT,
    FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_history_sync_instance ON whatsapp_history_syncs(instance_id);
CREATE INDEX IF NOT EXISTS idx_history_sync_status ON whatsapp_history_syncs(status);
CREATE INDEX IF NOT EXISTS idx_history_sync_cycle ON whatsapp_history_syncs(cycle_id);
