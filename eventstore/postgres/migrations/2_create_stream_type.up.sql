CREATE OR REPLACE FUNCTION create_stream_type(stream_type_id TEXT)
RETURNS VOID
AS $$

    INSERT INTO stream_types (id) VALUES (stream_type_id)
    ON CONFLICT (id)
    DO NOTHING;

$$ LANGUAGE SQL;
