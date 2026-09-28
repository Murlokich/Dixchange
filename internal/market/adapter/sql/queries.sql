-- name: GetPairs :many
SELECT
    p.id AS pair_id,
    p.base_currency AS base_currency_id,
    b.code AS base_currency_code,
    p.quote_currency AS quote_currency_id,
    q.code AS quote_currency_code,
    p.lot_size AS lot_size,
    p.price_tick AS price_tick,
    p.price_precision AS price_precision,
    b.precision AS base_precision,
    q.precision AS quote_precision
FROM 
    market.pair p   
JOIN 
    market.symbol b ON p.base_currency = b.id 
JOIN 
    market.symbol q ON p.quote_currency = q.id;