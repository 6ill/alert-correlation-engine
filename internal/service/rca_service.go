package service

import (
	"context"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/repository"
)

// maxSuggestions caps how many past RCA entries are suggested per new
// incident, so the response stays readable instead of dumping every match.
const maxSuggestions = 3

type RCAService struct {
	repo *repository.RCARepository
}

func NewRCAService(repo *repository.RCARepository) *RCAService {
	return &RCAService{repo: repo}
}

func (s *RCAService) Create(ctx context.Context, rca model.RCAKnowledge) (*model.RCAKnowledge, error) {
	return s.repo.Create(ctx, rca)
}

// Suggest returns past RCA entries tagged with the same resource_type and
// metric_type as a newly created incident
func (s *RCAService) Suggest(ctx context.Context, resourceType, metricType string) ([]model.RCAKnowledge, error) {
	return s.repo.FindByTags(ctx, resourceType, metricType, maxSuggestions)
}
