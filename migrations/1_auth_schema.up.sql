BEGIN;

CREATE SCHEMA auth;
CREATE TABLE IF NOT EXISTS auth.user (
    id UUID PRIMARY KEY,
    email VARCHAR(254) NOT NULL UNIQUE, -- https://errata.rfc-editor.org/eid1690/ --
    password_hash VARCHAR(256) NOT NULL
);

COMMIT;