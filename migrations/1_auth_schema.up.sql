BEGIN;

CREATE SCHEMA auth;
CREATE TABLE IF NOT EXISTS auth.user (
    id CHARACTER(36)
    email CHARACTER(254) -- https://errata.rfc-editor.org/eid1690/ --
    password_hash ...
    token CHARACTER ...
)

COMMIT;