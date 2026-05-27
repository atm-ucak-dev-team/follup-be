-- Remove ticket metadata columns from tickets table
ALTER TABLE tickets DROP COLUMN IF EXISTS title;
ALTER TABLE tickets DROP COLUMN IF EXISTS stakeholder;
ALTER TABLE tickets DROP COLUMN IF EXISTS status;
