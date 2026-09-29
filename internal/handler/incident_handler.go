package handler

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/6ill/alert-correlation-engine/internal/repository"
	"github.com/6ill/alert-correlation-engine/internal/service"
	"github.com/gofiber/fiber/v2"
)

type IncidentHandler struct {
	svc *service.IncidentService
}

func NewIncidentHandler(svc *service.IncidentService) *IncidentHandler {
	return &IncidentHandler{svc: svc}
}

func (h *IncidentHandler) RegisterRoutes(router fiber.Router) {
	incidents := router.Group("/incidents")
	incidents.Get("/", h.List)
	incidents.Get("/:id", h.GetDetail)
}

// List handles GET /incidents?status=open|resolved (status is optional).
func (h *IncidentHandler) List(c *fiber.Ctx) error {
	status := c.Query("status")
	if status != "" && status != "open" && status != "resolved" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be 'open' or 'resolved'"})
	}

	incidents, err := h.svc.List(c.Context(), status)
	if err != nil {
		slog.Error("failed to list incidents", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"incidents": incidents})
}

// GetDetail handles GET /incidents/:id.
func (h *IncidentHandler) GetDetail(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id must be a number"})
	}

	detail, err := h.svc.GetDetail(c.Context(), id)
	if errors.Is(err, repository.ErrIncidentNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "incident not found"})
	}
	if err != nil {
		slog.Error("failed to get incident detail", "error", err, "incident_id", id)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(detail)
}
