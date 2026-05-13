CREATE TABLE release_hashes (
    id          UUID PRIMARY KEY,
    release_id  UUID NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    filename    VARCHAR(500) NOT NULL,
    algorithm   VARCHAR(10) NOT NULL,
    hash        VARCHAR(128) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_release_hash UNIQUE (release_id, filename, algorithm)
);
