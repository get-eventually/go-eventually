-- Global ordering is not really working well (due to SERIAL not being transactionally safe),
-- and in general is not a scalable feature. Hence, with v0.1.0 it's being deprecated :)
ALTER TABLE events
    DROP COLUMN IF EXISTS global_sequence_number;

-- Hardcoded Event Subscriptions are deprecated in v0.1.0.
-- To create Event Subscriptions using PostgreSQL, take a look at Debezium: https://debezium.io/
DROP FUNCTION get_or_create_subscription_checkpoint;
DROP TABLE subscriptions_checkpoints;
