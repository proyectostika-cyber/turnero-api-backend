-- Add unique constraint to conversation_state (tenant_id, customer_id)
-- This ensures one conversation state per customer per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_state_tenant_customer
ON conversation_state (tenant_id, customer_id);
