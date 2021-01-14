CREATE TABLE streams (
    id          TEXT    NOT NULL,
    stream_type TEXT    NOT NULL,
    "version"   INTEGER NOT NULL  DEFAULT 0,

    PRIMARY KEY (stream_type, id)
);

CREATE TABLE events (
    global_sequence_number SERIAL,
    stream_type            TEXT    NOT NULL,
    stream_id              TEXT    NOT NULL,
    event_type             TEXT    NOT NULL,
    "version"              INTEGER NOT NULL,
    "event"                JSONB   NOT NULL,
    metadata               JSONB   NOT NULL,

    PRIMARY KEY (stream_type, stream_id, "version"),
    FOREIGN KEY (stream_type, stream_id) REFERENCES streams(stream_type, id) ON DELETE CASCADE
);
