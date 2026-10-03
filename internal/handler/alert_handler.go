package handler

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/service"
)

type AlertHandler struct {
	alertPipelineSvc *service.AlertPipelineService
}

func NewAlertHandler(
	alertPipelineSvc *service.AlertPipelineService,
) *AlertHandler {
	return &AlertHandler{alertPipelineSvc: alertPipelineSvc}
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
	if a.Status == "" {
		a.Status = model.AlertStatusFiring
	}

	result, err := h.alertPipelineSvc.Process(c.Context(), a, c.Body())
	if err != nil {
		slog.Error("failed to process alert", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	status := fiber.StatusCreated
	if result.Status == "duplicate_ignored" || result.Status == "suppressed_known_pattern" {
		status = fiber.StatusOK
	}

	return c.Status(status).JSON(result)
}
