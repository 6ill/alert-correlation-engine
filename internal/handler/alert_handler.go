package handler

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/service"
)

type AlertHandler struct {
	svc *service.AlertService
}

func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

func (h *AlertHandler) RegisterRoutes(router fiber.Router) {
	alerts := router.Group("/alerts")
	alerts.Post("/", h.Create)
}

func (h *AlertHandler) Create(c *fiber.Ctx) error {
	var a model.Alert
	if err := json.Unmarshal(c.Body(), &a); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	if a.RawAlertID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "raw_alert_id is required"})
	}
	if a.Timestamp.IsZero() {
		a.Timestamp = time.Now()
	}

	result, err := h.svc.Ingest(c.Context(), a, c.Body())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	status := fiber.StatusCreated
	if result.Status == "duplicate_ignored" {
		status = fiber.StatusOK
	}
	return c.Status(status).JSON(fiber.Map{
		"status":      result.Status,
		"fingerprint": result.Fingerprint,
	})
}
