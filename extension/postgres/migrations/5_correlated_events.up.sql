CREATE TABLE correlated_events (
    correlation_id        TEXT    NOT NULL,
    event_stream_type     TEXT    NOT NULL,
    event_stream_id       TEXT    NOT NULL,
    event_stream_version  INTEGER NOT NULL,


    FOREIGN KEY (event_stream_type, event_stream_id, event_stream_version)
        REFERENCES events(stream_type, stream_id, version)  ON DELETE CASCADE
);

CREATE MATERIALIZED VIEW correlated_events_view AS
    SELECT ce.correlation_id, e.*
    FROM correlated_events ce
        INNER JOIN events e ON e.stream_type = ce.event_stream_type
            AND e.stream_id = ce.event_stream_id
            AND e.version = ce.event_stream_version;

CREATE OR REPLACE FUNCTION project_correlated_event()
RETURNS TRIGGER
AS $$
DECLARE
    event_correlation_id TEXT;
BEGIN

    event_correlation_id = NEW.metadata->>'Correlation-Id';
    IF event_correlation_id IS NULL THEN
        RETURN NEW;
    END IF;

    INSERT INTO correlated_events
    VALUES (event_correlation_id, NEW.stream_type, NEW.stream_id, NEW.version);

    RETURN NEW;

END;
$$ LANGUAGE PLPGSQL;

CREATE TRIGGER project_correlated_event_on_append
    AFTER INSERT
    ON events
    FOR EACH ROW
    EXECUTE PROCEDURE project_correlated_event();
