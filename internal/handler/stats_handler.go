package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/6ill/alert-correlation-engine/internal/service"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

func (h *StatsHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/stats", h.Get)
}

func (h *StatsHandler) Get(c *fiber.Ctx) error {
	stats, err := h.svc.Get(c.Context())
	if err != nil {
		slog.Error("failed to compute stats", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(stats)
}
