package service

import (
	"context"
	"time"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/repository"
)

const (
	// relatedEventsWindow is how far back to look for other alerts on the same
	// resource when showing "related events" alongside an incident.
	relatedEventsWindow = 15 * time.Minute

	// defaultListLimit caps how many incidents a listing call returns.
	defaultListLimit = 50

	// recurrenceWindowDays is the starting default for
	// "how many times did this fingerprint produce an incident recently".
	recurrenceWindowDays = 7

	// maxPreviousIncidents caps the history list on the detail view.
	maxPreviousIncidents = 10
)

type IncidentDetail struct {
	Incident          model.Incident          `json:"incident"`
	Recurrence        model.Recurrence        `json:"recurrence"`
	PreviousIncidents []model.IncidentSummary `json:"previous_incidents"`
	RelatedAlerts     []model.AlertSummary    `json:"related_alerts"`
	RCASuggestions    []model.RCAKnowledge    `json:"rca_suggestions"`
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

// Recurrence reports how often this incident's fingerprint produced
// incidents before it. Shared by the detail view and the alert pipeline.
func (s *IncidentService) Recurrence(ctx context.Context, id int64) (model.Recurrence, error) {
	return s.incidentRepo.Recurrence(ctx, id, recurrenceWindowDays)
}

// GetDetail composes a single incident with the context an itops needs
// to triage it: how often it has happened before, other alerts recently seen
// on the same resource, and past RCA writeups for the same resource/metric.
func (s *IncidentService) GetDetail(ctx context.Context, id int64) (*IncidentDetail, error) {
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	recurrence, err := s.Recurrence(ctx, id)
	if err != nil {
		return nil, err
	}

	previous, err := s.incidentRepo.ListPrevious(ctx, id, maxPreviousIncidents)
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
		Incident:          *incident,
		Recurrence:        recurrence,
		PreviousIncidents: previous,
		RelatedAlerts:     relatedAlerts,
		RCASuggestions:    rcaSuggestions,
	}, nil
}
