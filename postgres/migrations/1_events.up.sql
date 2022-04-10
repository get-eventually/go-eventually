CREATE TABLE events (
    event_stream_id  TEXT    NOT NULL,
    "type"           TEXT    NOT NULL,
    "version"        INTEGER NOT NULL CHECK ("version" > 0),
    "event"          BYTEA   NOT NULL,
    metadata         JSONB,

    PRIMARY KEY (event_stream_id, "version")
);

CREATE INDEX event_stream_id_idx ON events (event_stream_id);
