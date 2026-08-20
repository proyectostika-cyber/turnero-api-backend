package dto

import "github.com/google/uuid"

// N8NWhatsAppSendRequest is the payload to send a WhatsApp text via Evolution API.
// tenant_id can be provided in the body or via X-Tenant-ID header (handled by middleware).
type N8NWhatsAppSendRequest struct {
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	Instance string     `json:"instance" validate:"required"`
	Number   string     `json:"number" validate:"required"`
	Text     string     `json:"text" validate:"required"`
}
