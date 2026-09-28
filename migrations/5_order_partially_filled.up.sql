BEGIN;

UPDATE "order"."order" SET quantity_filled = 0 WHERE quantity_filled IS NULL;
ALTER TABLE "order"."order" ALTER COLUMN quantity_filled SET NOT NULL;

ALTER TABLE "order"."order" DROP CONSTRAINT order_status_check;
ALTER TABLE "order"."order" ADD CONSTRAINT order_status_check
    CHECK (status IN ('open', 'partially_filled', 'filled', 'cancelled'));

-- Orders filled in part before this status existed stayed 'open'.
UPDATE "order"."order" SET status = 'partially_filled'
WHERE status = 'open' AND quantity_filled > 0;

COMMIT;
