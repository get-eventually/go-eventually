CREATE TABLE event_streams (
    event_stream_id TEXT    NOT NULL PRIMARY KEY,
    "version"       INTEGER NOT NULL CHECK ("version" > 0)
);

CREATE TABLE events (
    event_stream_id  TEXT    NOT NULL,
    "type"           TEXT    NOT NULL,
    "version"        INTEGER NOT NULL CHECK ("version" > 0),
    "event"          BYTEA   NOT NULL,
    metadata         JSONB,

    PRIMARY KEY (event_stream_id, "version"),
    FOREIGN KEY (event_stream_id) REFERENCES event_streams (event_stream_id) ON DELETE CASCADE
);

CREATE INDEX event_stream_id_idx ON events (event_stream_id);
