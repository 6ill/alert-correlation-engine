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
	alertSvc       *service.AlertService
	suppressionSvc *service.SuppressionService
	correlationSvc *service.CorrelationService
}

func NewAlertHandler(
	alertSvc *service.AlertService,
	suppressionSvc *service.SuppressionService,
	correlationSvc *service.CorrelationService,
) *AlertHandler {
	return &AlertHandler{alertSvc: alertSvc, suppressionSvc: suppressionSvc, correlationSvc: correlationSvc}
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

	ingestResult, err := h.alertSvc.Ingest(c.Context(), a, c.Body())
	if err != nil {
		slog.Error("failed to ingest alert", "error", err, "raw_alert_id", a.RawAlertID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	if ingestResult.Status == "duplicate_ignored" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":      ingestResult.Status,
			"fingerprint": ingestResult.Fingerprint,
		})
	}

	// Suppression only applies to firing alerts - a resolve event always
	// goes straight to the correlation engine so an open incident can
	// actually be closed.
	if a.Status == model.AlertStatusFiring {
		rule, err := h.suppressionSvc.Match(c.Context(), a.ResourceType, a.MetricType, a.Timestamp)
		if err != nil {
			slog.Error("failed to check suppression rules", "error", err, "fingerprint", ingestResult.Fingerprint)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		if rule != nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":             "suppressed_known_pattern",
				"fingerprint":        ingestResult.Fingerprint,
				"suppression_rule":   rule.Name,
				"suppression_reason": rule.Reason,
			})
		}
	}

	correlationResult, err := h.correlationSvc.Correlate(
		c.Context(),
		ingestResult.Fingerprint,
		model.AlertIdentity{
			Account:      a.Account,
			ResourceID:   a.ResourceID,
			ResourceType: a.ResourceType,
			MetricType:   a.MetricType,
		},
		a.Status == model.AlertStatusResolved,
	)
	if err != nil {
		slog.Error("failed to correlate alert", "error", err, "fingerprint", ingestResult.Fingerprint)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":            ingestResult.Status,
		"fingerprint":       ingestResult.Fingerprint,
		"incident_id":       correlationResult.IncidentID,
		"correlation_event": correlationResult.Event,
	})
}
