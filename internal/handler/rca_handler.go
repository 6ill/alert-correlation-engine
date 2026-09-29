package handler

import (
	"encoding/json"
	"log/slog"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/service"
	"github.com/gofiber/fiber/v2"
)

type RCAHandler struct {
	svc *service.RCAService
}

func NewRCAHandler(svc *service.RCAService) *RCAHandler {
	return &RCAHandler{svc: svc}
}

func (h *RCAHandler) RegisterRoutes(router fiber.Router) {
	rca := router.Group("/rca-knowledge")
	rca.Post("/", h.Create)
}

func (h *RCAHandler) Create(c *fiber.Ctx) error {
	var rca model.RCAKnowledge
	if err := json.Unmarshal(c.Body(), &rca); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	if rca.ResourceType == "" || rca.MetricType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "resource_type and metric_type are required"})
	}

	created, err := h.svc.Create(c.Context(), rca)
	if err != nil {
		slog.Error("failed to create rca knowledge", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}
