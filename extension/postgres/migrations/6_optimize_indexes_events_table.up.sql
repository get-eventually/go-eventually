-- Create a unique index for the global_sequence_number, since it is not allowed
-- to have multiple events with the same sequence number.
CREATE UNIQUE INDEX IF NOT EXISTS events_unique_sequence_number
    ON events (global_sequence_number);

-- This index should help speed up lookups when streaming events by their stream type.
CREATE INDEX IF NOT EXISTS events_by_stream_type_sequence_number
    ON events (stream_type, global_sequence_number);
