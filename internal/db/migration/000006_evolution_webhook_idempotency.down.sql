DROP INDEX IF EXISTS idx_webhook_logs_evolution_message;

ALTER TABLE webhook_logs
DROP COLUMN IF EXISTS evolution_message_id,
DROP COLUMN IF EXISTS tenant_channel_id;
