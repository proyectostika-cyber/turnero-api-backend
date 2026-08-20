package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/unknowncode44/appointments/internal/api/middleware"
	"github.com/unknowncode44/appointments/internal/services"
)

type mockEvolutionService struct {
	lastInstance string
	lastNumber   string
	lastText     string
	result       services.EvolutionSendTextResult
	err          error
}

func (m *mockEvolutionService) SendText(ctx context.Context, instance, number, text string) (services.EvolutionSendTextResult, error) {
	m.lastInstance = instance
	m.lastNumber = number
	m.lastText = text
	return m.result, m.err
}

func TestN8NWhatsAppHandlerSendTextSuccess(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.N8NTenantKey, uuid.New())
		return c.Next()
	})

	mockSvc := &mockEvolutionService{
		result: services.EvolutionSendTextResult{
			StatusCode:  202,
			Body:        []byte(`{"ok":true}`),
			ContentType: "application/json",
		},
	}
	handler := NewN8NWhatsAppHandler(mockSvc)
	app.Post("/api/v1/n8n/whatsapp/send", handler.SendText)

	payload := map[string]string{
		"instance": "test-instance",
		"number":   "+5491112345678",
		"text":     "hola",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/n8n/whatsapp/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, 202, resp.StatusCode)
	require.JSONEq(t, `{"ok":true}`, string(respBody))
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, payload["instance"], mockSvc.lastInstance)
	require.Equal(t, payload["number"], mockSvc.lastNumber)
	require.Equal(t, payload["text"], mockSvc.lastText)
}

func TestN8NWhatsAppHandlerSendTextValidation(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.N8NTenantKey, uuid.New())
		return c.Next()
	})

	handler := NewN8NWhatsAppHandler(&mockEvolutionService{})
	app.Post("/api/v1/n8n/whatsapp/send", handler.SendText)

	payload := map[string]string{
		"number": "+5491112345678",
		"text":   "hola",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/n8n/whatsapp/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
