-- name: DepositUSD :exec
UPDATE balance.main SET usd = usd + $1 WHERE user_id = $2;

-- name: CreateBalanceRecord :exec
INSERT INTO balance.main (user_id) VALUES ($1);