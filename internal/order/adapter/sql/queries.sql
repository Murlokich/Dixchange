-- name: CreateOrder :exec
INSERT INTO "order"."order" (id, user_id, market_pair_id, action, quantity_ordered, price, status)
    VALUES ($1, $2, $3, $4, $5, $6, $7);