package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/unknowncode44/appointments/internal/api/dto"
	"github.com/unknowncode44/appointments/internal/api/response"
	"github.com/unknowncode44/appointments/internal/services"
)

type N8NWhatsAppHandler struct {
	service services.EvolutionService
}

func NewN8NWhatsAppHandler(service services.EvolutionService) *N8NWhatsAppHandler {
	return &N8NWhatsAppHandler{service: service}
}

func (h *N8NWhatsAppHandler) SendText(c *fiber.Ctx) error {
	if _, err := ExtractTenantFromN8NContext(c); err != nil {
		return response.BadRequest(c, err)
	}
	var req dto.N8NWhatsAppSendRequest
	if err := bindAndValidate(c, &req); err != nil {
		return response.BadRequest(c, err)
	}
	result, err := h.service.SendText(c.Context(), req.Instance, req.Number, req.Text)
	if err != nil {
		return response.Error(c, err)
	}
	if result.ContentType != "" {
		c.Set(fiber.HeaderContentType, result.ContentType)
	}
	return c.Status(result.StatusCode).Send(result.Body)
}
