ALTER TABLE event_streams ALTER COLUMN "version" TYPE BIGINT;
ALTER TABLE events ALTER COLUMN "version" TYPE BIGINT;
ALTER TABLE aggregates ALTER COLUMN "version" TYPE BIGINT;