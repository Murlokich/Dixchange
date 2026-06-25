BEGIN;

CREATE SCHEMA auth;
CREATE TABLE IF NOT EXISTS auth.user (
    id UUID PRIMARY KEY,
    email CHARACTER(254) NOT NULL UNIQUE, -- https://errata.rfc-editor.org/eid1690/ --
    password_hash CHARACTER(256) NOT NULL
);

COMMIT;