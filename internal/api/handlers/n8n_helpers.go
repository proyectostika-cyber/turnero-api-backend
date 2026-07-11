package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/unknowncode44/appointments/internal/api/middleware"
)

// ExtractTenantFromN8NContext obtiene el tenant_id validado por ValidateTenantFromRequest middleware
func ExtractTenantFromN8NContext(c *fiber.Ctx) (uuid.UUID, error) {
	tenantID, ok := middleware.ExtractN8NTenant(c)
	if !ok {
		return uuid.Nil, errors.New("tenant_id not found in context")
	}
	return tenantID, nil
}

// GetTenantIDUnified intenta obtener tenant_id de N8N context o JWT payload
// Útil para handlers que pueden ser llamados por ambas rutas (N8N y JWT)
// Prioridad: 1) N8N context, 2) JWT payload
func GetTenantIDUnified(c *fiber.Ctx) (uuid.UUID, error) {
	// 1. Intentar N8N context primero (rutas /api/v1/n8n/*)
	if tenantID, ok := middleware.ExtractN8NTenant(c); ok {
		return tenantID, nil
	}

	// 2. Intentar JWT payload (rutas autenticadas normales /api/v1/*)
	payload, err := middleware.ExtractUserFromContext(c)
	if err == nil && payload.TenantID != nil {
		return *payload.TenantID, nil
	}

	return uuid.Nil, errors.New("tenant_id not found in N8N context or JWT payload")
}
