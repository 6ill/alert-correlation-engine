package handler

import (
	"encoding/json"
	"log/slog"

	"github.com/6ill/alert-correlation-engine/internal/adapter/tcop"
	"github.com/6ill/alert-correlation-engine/internal/service"
	"github.com/gofiber/fiber/v2"
)

type TCOPWebhookHandler struct {
	alertPipelineSvc *service.AlertPipelineService
	token            string
	basicAuthUsers   map[string]string
}

func NewTCOPWebhookHandler(
	alertPipelineSvc *service.AlertPipelineService,
	token string,
	basicAuthUsers map[string]string,
) *TCOPWebhookHandler {
	return &TCOPWebhookHandler{
		alertPipelineSvc: alertPipelineSvc,
		token:            token,
		basicAuthUsers:   basicAuthUsers,
	}
}

func (h *TCOPWebhookHandler) RegisterRoutes(router fiber.Router) {
	webhooks := router.Group("/webhooks/tcop")
	webhooks.Post("/:token", h.Receive)
}


func (h *TCOPWebhookHandler) Receive(c *fiber.Ctx) error {
	if c.Params("token") != h.token {
		return c.SendStatus(fiber.StatusNotFound)
	}

	var payload tcop.Payload
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	alert, err := tcop.Translate(payload)
	if err != nil {
		slog.Error("failed to translate TCOP payload", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unrecognized payload"})
	}

	result, err := h.alertPipelineSvc.Process(c.Context(), alert, c.Body())
	if err != nil {
		slog.Error("failed to process TCOP alert", "error", err, "session_id", payload.SessionID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	
	return c.Status(fiber.StatusCreated).JSON(result)
}