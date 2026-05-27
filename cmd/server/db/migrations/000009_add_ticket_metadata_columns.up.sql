-- Add ticket metadata columns to tickets table
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS stakeholder TEXT;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS status TEXT;
