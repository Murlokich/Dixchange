BEGIN;

ALTER TABLE market.pair DROP COLUMN IF EXISTS price_precision;

COMMIT;
