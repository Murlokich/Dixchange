BEGIN;

ALTER TABLE "order"."order" RENAME COLUMN side TO action;
ALTER TYPE "order".side RENAME TO action;

COMMIT;
