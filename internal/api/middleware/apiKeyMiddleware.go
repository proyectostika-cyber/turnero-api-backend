package middleware

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
	"github.com/unknowncode44/appointments/internal/util"
)

// APIKeyAuthKey is the context key for API key authentication status.
const APIKeyAuthKey = "api_key_auth"

// APIKeyMiddleware validates the X-API-Key header against the configured N8N API key.
// Uses constant-time comparison to prevent timing attacks.
// This middleware only handles authentication. Tenant validation should be done
// separately using ValidateTenantFromRequest middleware for multi-tenant endpoints.
func APIKeyMiddleware(config util.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")

		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing X-API-Key header",
			})
		}

		if !validateAPIKey(apiKey, config.N8NAPIKey) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid API key",
			})
		}

		// Store authentication status in context
		c.Locals(APIKeyAuthKey, true)
		return c.Next()
	}
}

// validateAPIKey performs constant-time comparison to prevent timing attacks.
func validateAPIKey(provided, expected string) bool {
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
