BEGIN;

CREATE SCHEMA IF NOT EXISTS auth;
CREATE TABLE IF NOT EXISTS auth.user (
    id UUID PRIMARY KEY,
    email VARCHAR(254) NOT NULL UNIQUE, -- https://errata.rfc-editor.org/eid1690/ --
    password_hash VARCHAR(256) NOT NULL
);

CREATE SCHEMA IF NOT EXISTS market;
CREATE TABLE IF NOT EXISTS market.symbol (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code CHARACTER(3) NOT NULL UNIQUE,
    precision SMALLINT NOT NULL
);

INSERT INTO market.symbol (code, precision) VALUES ('USD', 2), ('KCN', 0);

CREATE TABLE IF NOT EXISTS market.pair (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    base_currency INT NOT NULL REFERENCES market.symbol(id),
    quote_currency INT NOT NULL REFERENCES market.symbol(id),
    lot_size BIGINT NOT NULL,
    price_tick BIGINT NOT NULL,
    UNIQUE(base_currency, quote_currency)
);

INSERT INTO market.pair (base_currency, quote_currency, lot_size, price_tick) VALUES (
   (SELECT id FROM market.symbol WHERE code = 'USD'),
   (SELECT id FROM market.symbol WHERE code = 'KCN'),
   100,
   1
);

CREATE SCHEMA IF NOT EXISTS wallet;
CREATE TABLE IF NOT EXISTS wallet.balance (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.user(id),
    symbol_id INT NOT NULL REFERENCES market.symbol(id),
    main BIGINT NOT NULL DEFAULT 0,
    reserved BIGINT NOT NULL DEFAULT 0, 
    UNIQUE(user_id, symbol_id)
);

COMMIT;