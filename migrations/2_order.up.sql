BEGIN;

CREATE SCHEMA IF NOT EXISTS "order";
CREATE TYPE "order".action AS ENUM ('buy', 'sell');

CREATE TABLE IF NOT EXISTS "order"."order" (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.user(id),
    market_pair_id INT NOT NULL REFERENCES market.pair(id),
    action "order".action NOT NULL,
    quantity_ordered BIGINT NOT NULL,
    quantity_filled BIGINT DEFAULT 0,
    price BIGINT NOT NULL, 
    status TEXT NOT NULL CHECK (status IN ('open', 'filled', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_at TIMESTAMPTZ
);

COMMIT;