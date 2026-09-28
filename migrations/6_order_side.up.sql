BEGIN;

ALTER TYPE "order".action RENAME TO side;
ALTER TABLE "order"."order" RENAME COLUMN action TO side;

COMMIT;
