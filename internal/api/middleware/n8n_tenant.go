package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	db "github.com/unknowncode44/appointments/internal/db/sqlc"
)

// N8NTenantKey es la clave del context donde se almacena el tenant_id validado
const N8NTenantKey = "n8n_tenant_id"

// ValidateTenantFromRequest extrae y valida el tenant_id del request.
// Busca en este orden: X-Tenant-ID header, query param tenant_id, body JSON.
// Valida que el tenant existe en DB y está activo.
func ValidateTenantFromRequest(store *db.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tenantIDStr string

		// 1. Intentar header X-Tenant-ID (recomendado para N8N, especialmente GET)
		tenantIDStr = c.Get("X-Tenant-ID")

		// 2. Intentar query param tenant_id (para GET requests sin header)
		if tenantIDStr == "" {
			tenantIDStr = c.Query("tenant_id")
		}

		// 3. Intentar body JSON (para POST/PATCH requests)
		if tenantIDStr == "" {
			var body map[string]interface{}
			if err := c.BodyParser(&body); err == nil {
				if v, ok := body["tenant_id"].(string); ok {
					tenantIDStr = v
				}
			}
		}

		// Tenant ID es requerido para rutas protegidas N8N
		if tenantIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "tenant_id is required (header X-Tenant-ID, query param, or body field)",
			})
		}

		// Parsear UUID
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid tenant_id format, must be valid UUID",
			})
		}

		// Validar que el tenant existe en DB
		tenant, err := store.GetTenant(c.Context(), tenantID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "tenant not found",
			})
		}

		// Verificar que el tenant está activo
		if !tenant.Active {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "tenant is not active",
			})
		}

		// Almacenar tenant_id validado en el contexto para uso posterior
		c.Locals(N8NTenantKey, tenantID)

		return c.Next()
	}
}

// ExtractN8NTenant obtiene el tenant_id del contexto (inyectado por ValidateTenantFromRequest)
func ExtractN8NTenant(c *fiber.Ctx) (uuid.UUID, bool) {
	tenantID, ok := c.Locals(N8NTenantKey).(uuid.UUID)
	return tenantID, ok
}
