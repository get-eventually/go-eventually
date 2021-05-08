CREATE OR REPLACE FUNCTION append_to_store(
    _stream_type  TEXT,
    stream_id     TEXT,
    version_check INTEGER,
    event_name    TEXT,
    event_payload JSONB,
    metadata      JSONB
) RETURNS TABLE (
    "version" INTEGER
) AS $$
DECLARE
    last_stream_version INTEGER;
BEGIN

    -- Retrieve the latest stream version for the specified stream.
    SELECT s."version"
    INTO last_stream_version
    FROM streams s
    WHERE id = stream_id AND s.stream_type = _stream_type;

    IF NOT FOUND THEN
        -- Create a new entry for the desired stream.
        INSERT INTO streams (id, stream_type)
        VALUES (stream_id, _stream_type);

        -- Make sure to initialize the stream version in this case.
        last_stream_version = 0;
    END IF;

    -- Perform optimistic concurrency check.
    IF version_check <> -1 AND version_check <> last_stream_version THEN
        RAISE EXCEPTION 'stream version check failed, expected: %, current: %', version_check, last_stream_version;
    END IF;

    -- Increment the stream version prior to inserting the new event.
    last_stream_version = last_stream_version + 1;

    -- Add a recorded_at timestamp in the metadata.
    metadata = metadata || ('{"Recorded-At": ' || to_json(NOW()) || '}')::JSONB;

    -- Insert the event into the events table.
    -- Version numbers should start from 1.
    INSERT INTO events (
        stream_type,
        stream_id,
        "version",
        event_type,
        "event",
        metadata
    ) VALUES (
        _stream_type,
        stream_id,
        last_stream_version,
        event_name,
        event_payload,
        metadata
    );

    -- Update the stream with the latest version computed.
    UPDATE streams s
    SET "version" = last_stream_version
    WHERE id = stream_id AND s.stream_type = _stream_type;

    RETURN QUERY
        SELECT last_stream_version;

END;
$$ LANGUAGE PLPGSQL;
