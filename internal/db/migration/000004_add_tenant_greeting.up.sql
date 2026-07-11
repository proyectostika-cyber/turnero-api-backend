-- Add greeting_message column to tenants table
ALTER TABLE tenants
ADD COLUMN greeting_message TEXT;

-- Set default greeting message for existing tenants
UPDATE tenants
SET greeting_message = '¡Hola! ¿Dime en qué puedo ayudarte?'
WHERE greeting_message IS NULL;

-- Make the column NOT NULL after populating it
ALTER TABLE tenants
ALTER COLUMN greeting_message SET NOT NULL;
