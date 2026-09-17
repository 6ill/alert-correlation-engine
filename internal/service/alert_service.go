package service

import (
	"context"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/repository"
)

type IngestResult struct {
	Status      string // "ingested" | "duplicate_ignored"
	Fingerprint string
}

type AlertService struct {
	repo *repository.AlertRepository
}

func NewAlertService(repo *repository.AlertRepository) *AlertService {
	return &AlertService{repo: repo}
}

// Process an alert: count fingerprint, save idempotently.
func (s *AlertService) Ingest(ctx context.Context, a model.Alert, rawPayload []byte) (IngestResult, error) {
	fp := ComputeFingerprint(a.Account, a.ResourceID, a.MetricType)

	inserted, err := s.repo.Insert(ctx, a, fp, rawPayload)
	if err != nil {
		return IngestResult{}, err
	}
	if !inserted {
		return IngestResult{Status: "duplicate_ignored", Fingerprint: fp}, nil
	}
	return IngestResult{Status: "ingested", Fingerprint: fp}, nil
}
