-- 000001_create_tables.down.sql
-- Drops: followups, tickets, users (reverse order of creation)

BEGIN;

DROP TABLE IF EXISTS followups;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS users;

COMMIT;