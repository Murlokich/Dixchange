BEGIN;

ALTER TABLE "order".command_log_usd_kcn RENAME COLUMN order_details TO place;
ALTER TYPE "order".command_type RENAME VALUE 'create' TO 'place';

COMMIT;
