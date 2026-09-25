BEGIN;

-- Existing pairs stored prices at the quote precision. For USD/KCN that is 0,
-- which is also the pair price precision, so stored prices stay valid.
ALTER TABLE market.pair ADD COLUMN price_precision SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE market.pair ALTER COLUMN price_precision DROP DEFAULT;

COMMIT;
