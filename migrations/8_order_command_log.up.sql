BEGIN;

CREATE TYPE "order".command_type AS ENUM ('place', 'cancel');

-- Commands accepted for the USD/KCN book whose effect hasn't reached "order"."order" yet.
-- One log per book: a book's commands form one ordered stream. Settlement deletes a command
-- in the transaction that applies it, so "order"."order" plus this table is the whole state
-- of the book. Commands are applied in id order (UUIDv7).
CREATE TABLE IF NOT EXISTS "order".command_log_usd_kcn (
    id UUID PRIMARY KEY,
    type "order".command_type NOT NULL,
    -- No foreign key: the order of a place command isn't in "order"."order" yet.
    order_id UUID NOT NULL,
    -- The order to place: user_id, pair_id, side, price, quantity. Set for place commands only.
    place JSONB,
    CHECK ((type = 'place') = (place IS NOT NULL))
);

COMMIT;
