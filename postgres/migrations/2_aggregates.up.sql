CREATE TABLE aggregates (
    aggregate_id  TEXT    NOT NULL,
    "type"       TEXT    NOT NULL,
    "version"    INTEGER NOT NULL CHECK ("version" > 0),
    "state"      BYTEA   NOT NULL,

    PRIMARY KEY (aggregate_id, "type")
);

CREATE INDEX aggregate_id_idx ON aggregates (aggregate_id);
