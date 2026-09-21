-- name: Deposit :execrows
UPDATE wallet.balance SET main = main + $1 WHERE user_id = $2 AND symbol_id = $3;

-- name: CreateBalanceRecord :exec
INSERT INTO wallet.balance (id, user_id, symbol_id) VALUES ($1, $2, $3);

-- name: ReserveFunds :execrows
UPDATE wallet.balance
SET main = main - $1,
    reserved = reserved + $1
WHERE user_id = $2 AND symbol_id = $3 AND main >= $1;

-- name: ReleaseReservedFunds :execrows
UPDATE wallet.balance
SET reserved = reserved - $1,
    main = main + $1
WHERE user_id = $2 AND symbol_id = $3 AND reserved >= $1;

-- name: TransferFunds :execrows
WITH debited AS (
    UPDATE wallet.balance payer
    SET reserved = payer.reserved - sqlc.arg(amount)
    WHERE payer.user_id = sqlc.arg(from_user_id)
      AND payer.symbol_id = sqlc.arg(symbol_id)
      AND payer.reserved >= sqlc.arg(amount)
    RETURNING payer.user_id
)
UPDATE wallet.balance payee
SET main = payee.main + sqlc.arg(amount)
WHERE payee.user_id = sqlc.arg(to_user_id)
  AND payee.symbol_id = sqlc.arg(symbol_id)
  AND EXISTS (SELECT 1 FROM debited);
