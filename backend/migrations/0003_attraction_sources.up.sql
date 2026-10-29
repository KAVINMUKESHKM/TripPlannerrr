ALTER TABLE attractions ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE attractions ADD COLUMN IF NOT EXISTS source_id TEXT;
ALTER TABLE attractions ADD COLUMN IF NOT EXISTS source_url TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS attractions_source_identity_idx
    ON attractions (source, source_id)
    WHERE source IS NOT NULL AND source_id IS NOT NULL;
