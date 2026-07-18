BEGIN;

CREATE SCHEMA IF NOT EXISTS auth;
CREATE TABLE IF NOT EXISTS auth.user (
    id UUID PRIMARY KEY,
    email VARCHAR(254) NOT NULL UNIQUE, -- https://errata.rfc-editor.org/eid1690/ --
    password_hash VARCHAR(256) NOT NULL
);

CREATE SCHEMA IF NOT EXISTS balance;
CREATE TABLE IF NOT EXISTS balance.main (
    user_id UUID PRIMARY KEY,
    usd NUMERIC(15, 6) NOT NULL DEFAULT 1000,
    kostyaCoin NUMERIC(15, 6) NOT NULL DEFAULT 0,
    reserved_usd NUMERIC(15, 6) NOT NULL DEFAULT 0,
    reserved_kostya_coin NUMERIC(15, 6) NOT NULL DEFAULT 0
);

COMMIT;