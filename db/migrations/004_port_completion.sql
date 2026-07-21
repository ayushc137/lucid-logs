-- Port-completion columns required by the Go domain models.
-- The activities domain model carries task-default and quick-log fields that the
-- initial schema deferred as "opaque options". Retrospectives split their
-- `responses` payload into auto_summary (generated) and user_content (authored).

ALTER TABLE activities ADD COLUMN default_duration INTEGER NOT NULL DEFAULT 0;
ALTER TABLE activities ADD COLUMN default_emotion_id TEXT;
ALTER TABLE activities ADD COLUMN default_priority INTEGER NOT NULL DEFAULT 3;
ALTER TABLE activities ADD COLUMN default_completed INTEGER NOT NULL DEFAULT 1;
ALTER TABLE activities ADD COLUMN quantity_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE activities ADD COLUMN quantity_default REAL;
ALTER TABLE activities ADD COLUMN quantity_step REAL;
ALTER TABLE activities ADD COLUMN quantity_unit_id TEXT;
ALTER TABLE activities ADD COLUMN default_impact TEXT NOT NULL DEFAULT 'positive';
ALTER TABLE activities ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

ALTER TABLE tasks ADD COLUMN source TEXT NOT NULL DEFAULT 'manual';

ALTER TABLE categories ADD COLUMN updated_by TEXT REFERENCES users(id);

ALTER TABLE retrospectives ADD COLUMN auto_summary TEXT;
ALTER TABLE retrospectives ADD COLUMN user_content TEXT;

-- activity_logs is a generic entity event log (entity_type/entity_id/event + display
-- helpers entity_title/entity_icon). The initial schema only had event_type + metadata.
ALTER TABLE activity_logs ADD COLUMN entity_title TEXT;
ALTER TABLE activity_logs ADD COLUMN entity_icon TEXT;
