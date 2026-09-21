-- name: CreateOrder :exec
INSERT INTO "order"."order" (id, user_id, market_pair_id, action, quantity_ordered, price, status)
    VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: FillOrder :execrows
UPDATE "order"."order"
SET quantity_filled = COALESCE(quantity_filled, 0) + sqlc.arg(quantity)::bigint,
    status = CASE
        WHEN COALESCE(quantity_filled, 0) + sqlc.arg(quantity)::bigint >= quantity_ordered THEN 'filled'
        ELSE status
    END,
    last_updated_at = NOW()
WHERE id = sqlc.arg(order_id);
