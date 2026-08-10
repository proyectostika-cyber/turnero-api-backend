ALTER TABLE webhook_logs
ADD COLUMN tenant_channel_id UUID REFERENCES tenant_channels(id) ON DELETE SET NULL,
ADD COLUMN evolution_message_id TEXT;

CREATE UNIQUE INDEX idx_webhook_logs_evolution_message
ON webhook_logs (tenant_channel_id, evolution_message_id)
WHERE tenant_channel_id IS NOT NULL AND evolution_message_id IS NOT NULL;
