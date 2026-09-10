-- name: CreateTrades :exec
INSERT INTO trade.trade (id, buy_order_id, sell_order_id, price, quantity)
    SELECT id, buy_order_id, sell_order_id, price, quantity
    FROM jsonb_to_recordset(sqlc.arg(trades)::jsonb)
        AS t(id uuid, buy_order_id uuid, sell_order_id uuid, price bigint, quantity bigint);