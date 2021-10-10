CREATE TABLE subscriptions_checkpoints (
    subscription_id      TEXT    PRIMARY KEY,
    last_sequence_number INTEGER NOT NULL     DEFAULT 0
);

CREATE OR REPLACE FUNCTION get_or_create_subscription_checkpoint(
    _subscription_id  TEXT
)
RETURNS TABLE (
    last_sequence_number INTEGER
) AS $$

    INSERT INTO subscriptions_checkpoints (subscription_id)
    VALUES (_subscription_id)
        -- Perform update to force row returning.
        ON CONFLICT (subscription_id)
        DO UPDATE SET subscription_id=EXCLUDED.subscription_id
    RETURNING last_sequence_number;

$$ LANGUAGE SQL;
