ALTER TABLE event_streams ALTER COLUMN "version" TYPE INTEGER;
ALTER TABLE events ALTER COLUMN "version" TYPE INTEGER;
ALTER TABLE aggregates ALTER COLUMN "version" TYPE INTEGER;
