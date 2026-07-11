-- Remove unique constraint from conversation_state
DROP INDEX IF EXISTS idx_conversation_state_tenant_customer;
