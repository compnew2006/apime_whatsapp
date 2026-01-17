ALTER TABLE instances
    ADD COLUMN IF NOT EXISTS history_sync_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS history_sync_cycle_id UUID,
    ADD COLUMN IF NOT EXISTS history_sync_updated_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS whatsapp_history_syncs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    instance_id UUID NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    payload_type TEXT NOT NULL,
    payload BYTEA NOT NULL,
    cycle_id UUID,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_history_sync_instance ON whatsapp_history_syncs(instance_id);
CREATE INDEX IF NOT EXISTS idx_history_sync_status ON whatsapp_history_syncs(status);
CREATE INDEX IF NOT EXISTS idx_history_sync_cycle ON whatsapp_history_syncs(cycle_id);
