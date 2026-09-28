BEGIN;

-- Trade participants and pair are copied from the orders, so trades can be read
-- without joining them. Orders never change owner or pair and trades are never
-- updated, so the copies can't go out of sync.
ALTER TABLE trade.trade
    ADD COLUMN buyer_id UUID REFERENCES auth.user(id),
    ADD COLUMN seller_id UUID REFERENCES auth.user(id),
    ADD COLUMN pair_id INT REFERENCES market.pair(id);

UPDATE trade.trade t
SET buyer_id = buy_order.user_id,
    seller_id = sell_order.user_id,
    pair_id = buy_order.market_pair_id
FROM "order"."order" buy_order, "order"."order" sell_order
WHERE buy_order.id = t.buy_order_id AND sell_order.id = t.sell_order_id;

ALTER TABLE trade.trade
    ALTER COLUMN buyer_id SET NOT NULL,
    ALTER COLUMN seller_id SET NOT NULL,
    ALTER COLUMN pair_id SET NOT NULL;

COMMIT;
