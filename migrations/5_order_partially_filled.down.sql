BEGIN;

-- The old constraint does not allow 'partially_filled', and those orders were 'open' before.
UPDATE "order"."order" SET status = 'open' WHERE status = 'partially_filled';

ALTER TABLE "order"."order" DROP CONSTRAINT order_status_check;
ALTER TABLE "order"."order" ADD CONSTRAINT order_status_check
    CHECK (status IN ('open', 'filled', 'cancelled'));

ALTER TABLE "order"."order" ALTER COLUMN quantity_filled DROP NOT NULL;

COMMIT;
