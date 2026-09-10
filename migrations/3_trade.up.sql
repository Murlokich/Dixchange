BEGIN;

CREATE SCHEMA IF NOT EXISTS trade;

CREATE TABLE IF NOT EXISTS trade.trade (
    id UUID PRIMARY KEY,
    buy_order_id UUID NOT NULL REFERENCES "order"."order"(id),
    sell_order_id UUID NOT NULL REFERENCES "order"."order"(id),
    price BIGINT NOT NULL,
    quantity BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;