package service

import (
	"context"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/repository"
)

type StatsService struct {
	repo *repository.StatsRepository
}

func NewStatsService(repo *repository.StatsRepository) *StatsService {
	return &StatsService{repo: repo}
}

func (s *StatsService) Get(ctx context.Context) (model.Stats, error) {
	totalAlerts, totalIncidents, err := s.repo.RawCounts(ctx)
	if err != nil {
		return model.Stats{}, err
	}

	ticketsSaved := totalAlerts - totalIncidents
	var noiseReductionRate float64
	if totalAlerts > 0 {
		noiseReductionRate = float64(ticketsSaved) / float64(totalAlerts)
	}

	return model.Stats{
		TotalAlerts:        totalAlerts,
		TotalIncidents:     totalIncidents,
		TicketsSaved:       ticketsSaved,
		NoiseReductionRate: noiseReductionRate,
	}, nil
}
