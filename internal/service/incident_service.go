package service

import (
	"context"
	"time"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/repository"
)

// relatedEventsWindow is how far back to look for other alerts on the same
// resource when showing "related events" alongside an incident.
// Informational only; it never merges into the incident itself.
const relatedEventsWindow = 15 * time.Minute

// defaultListLimit caps how many incidents a listing call returns.
const defaultListLimit = 50

type IncidentDetail struct {
	Incident       model.Incident       `json:"incident"`
	RelatedAlerts  []model.AlertSummary `json:"related_alerts"`
	RCASuggestions []model.RCAKnowledge `json:"rca_suggestions"`
}

type IncidentService struct {
	incidentRepo *repository.IncidentRepository
	alertRepo    *repository.AlertRepository
	rcaSvc       *RCAService
}

func NewIncidentService(incidentRepo *repository.IncidentRepository, alertRepo *repository.AlertRepository, rcaSvc *RCAService) *IncidentService {
	return &IncidentService{incidentRepo: incidentRepo, alertRepo: alertRepo, rcaSvc: rcaSvc}
}

func (s *IncidentService) List(ctx context.Context, status string) ([]model.Incident, error) {
	return s.incidentRepo.List(ctx, status, defaultListLimit)
}

// GetDetail composes a single incident with the context an ops
// needs to triage it: other alerts recently seen on the same resource, and
// past RCA writeups tagged with the same resource/metric type.
func (s *IncidentService) GetDetail(ctx context.Context, id int64) (*IncidentDetail, error) {
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	since := incident.LastSeen.Add(-relatedEventsWindow)
	relatedAlerts, err := s.alertRepo.FindRecentByResource(ctx, incident.Account, incident.ResourceID, incident.Fingerprint, since, 10)
	if err != nil {
		return nil, err
	}

	rcaSuggestions, err := s.rcaSvc.Suggest(ctx, incident.ResourceType, incident.MetricType)
	if err != nil {
		return nil, err
	}

	return &IncidentDetail{
		Incident:       *incident,
		RelatedAlerts:  relatedAlerts,
		RCASuggestions: rcaSuggestions,
	}, nil
}
