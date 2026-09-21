-- name: CreateTrade :exec
INSERT INTO trade.trade (id, buy_order_id, sell_order_id, price, quantity)
    VALUES ($1, $2, $3, $4, $5);
