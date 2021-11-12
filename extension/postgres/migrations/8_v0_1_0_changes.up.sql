-- correlated_events_view is using the old event column. The quickest way
-- to solve the conflicting issue is to drop the materialized view and rebuild it.
DROP MATERIALIZED VIEW correlated_events_view;

-- Global ordering is not really working well (due to SERIAL not being transactionally safe),
-- and in general is not a scalable feature. Hence, with v0.1.0 it's being deprecated :)
ALTER TABLE events
    DROP COLUMN IF EXISTS global_sequence_number;

-- Hardcoded Event Subscriptions are deprecated in v0.1.0.
-- To create Event Subscriptions using PostgreSQL, take a look at Debezium: https://debezium.io/
DROP FUNCTION get_or_create_subscription_checkpoint;
DROP TABLE subscriptions_checkpoints;

-- Recreate the materialized view.
CREATE MATERIALIZED VIEW correlated_events_view AS
    SELECT ce.correlation_id, e.*
    FROM correlated_events ce INNER JOIN events e
        ON e.stream_type = ce.event_stream_type
        AND e.stream_id = ce.event_stream_id
        AND e.version = ce.event_stream_version;

-- These are some unused triggers from the previous abstraction of the Event Store
-- which haven't been removed yet, but must be removed now due to the reference
-- to the global_sequence_number.
DROP TRIGGER IF EXISTS notify_stream_type_on_append ON events;
DROP TRIGGER IF EXISTS notify_all_on_append ON events;

DROP FUNCTION IF EXISTS notify_stream_type;
DROP FUNCTION IF EXISTS notify_all;
