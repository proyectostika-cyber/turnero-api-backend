-- Rollback: Remove greeting_message column from tenants table
ALTER TABLE tenants
DROP COLUMN IF EXISTS greeting_message;
