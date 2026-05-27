BEGIN;

ALTER TABLE followups
    DROP COLUMN IF EXISTS execution_count;

COMMIT;
