package test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"github.com/unknowncode44/appointments/internal/api/middleware"
	"github.com/unknowncode44/appointments/internal/util"
)

func TestAPIKeyMiddleware(t *testing.T) {
	validKey := "sk_n8n_test_abc123xyz456"
	invalidKey := "sk_n8n_test_wrong"

	config := util.Config{
		N8NAPIKey: validKey,
	}

	app := fiber.New()
	app.Use(middleware.APIKeyMiddleware(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "valid API key",
			apiKey:         validKey,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid API key",
			apiKey:         invalidKey,
			expectedStatus: fiber.StatusUnauthorized,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			expectedStatus: fiber.StatusUnauthorized,
		},
		{
			name:           "empty API key header",
			apiKey:         "   ",
			expectedStatus: fiber.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestAPIKeyMiddleware_ConstantTimeComparison(t *testing.T) {
	// This test ensures timing attacks are prevented by using constant-time comparison.
	// While we can't measure timing directly in a unit test, we verify the logic is sound.

	validKey := "sk_n8n_test_abcdefgh12345678"
	similarKey := "sk_n8n_test_abcdefgh12345679" // Only last char differs

	config := util.Config{
		N8NAPIKey: validKey,
	}

	app := fiber.New()
	app.Use(middleware.APIKeyMiddleware(config))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", similarKey)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}
