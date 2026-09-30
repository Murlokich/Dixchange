-- name: CreateOrder :exec
INSERT INTO "order"."order" (id, user_id, market_pair_id, side, quantity_ordered, price, status)
    VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: FillOrder :execrows
UPDATE "order"."order"
SET quantity_filled = quantity_filled + sqlc.arg(quantity)::bigint,
    status = CASE
        WHEN quantity_filled + sqlc.arg(quantity)::bigint >= quantity_ordered THEN 'filled'
        ELSE 'partially_filled'
    END,
    last_updated_at = NOW()
WHERE id = sqlc.arg(order_id);


-- name: GetUserOrdersByStatuses :many
SELECT * FROM "order"."order" WHERE user_id = $1 AND status = ANY($2::text[]) ORDER BY created_at DESC;

-- name: GetOrderCancelInfo :one
SELECT user_id, status FROM "order"."order" WHERE id = $1;

-- name: GetOrderForUpdate :one
SELECT * FROM "order"."order" WHERE id = $1 FOR UPDATE;

-- name: CancelOrder :execrows
UPDATE "order"."order"
SET status = 'cancelled',
    last_updated_at = NOW()
WHERE id = $1;
