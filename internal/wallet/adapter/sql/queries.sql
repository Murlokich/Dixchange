-- name: DepositUSD :exec
UPDATE wallet.balance SET main = main + $1 WHERE user_id = $2 AND symbol_id = $3;

-- name: CreateBalanceRecord :exec
INSERT INTO wallet.balance (id, user_id, symbol_id) VALUES ($1, $2, $3);

-- name: ReserveFunds :execrows
UPDATE wallet.balance 
SET main = main - $1,
    reserved = reserved + $1
WHERE user_id = $2 AND symbol_id = $3 AND main >= $1;