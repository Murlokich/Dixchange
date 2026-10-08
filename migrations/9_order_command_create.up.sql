BEGIN;

ALTER TYPE "order".command_type RENAME VALUE 'place' TO 'create';
ALTER TABLE "order".command_log_usd_kcn RENAME COLUMN place TO order_details;

COMMIT;
