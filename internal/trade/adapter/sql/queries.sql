-- name: CreateTrade :exec
INSERT INTO trade.trade (id, buy_order_id, sell_order_id, buyer_id, seller_id, pair_id, price, quantity)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8);


-- name: GetUserTrades :many
-- Trade IDs are UUIDv7, so ordering by ID orders trades by creation time.
SELECT * FROM trade.trade
WHERE (buyer_id = $1 OR seller_id = $1) AND id < $2
ORDER BY id DESC
LIMIT $3;
