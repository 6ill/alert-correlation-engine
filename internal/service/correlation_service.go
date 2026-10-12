package service

import (
	"context"
	"errors"
	"time"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/6ill/alert-correlation-engine/internal/repository"
)

type CorrelationEvent string

const (
	CorrelationEventNewIncident      CorrelationEvent = "new_incident"
	CorrelationEventMerged           CorrelationEvent = "merged_into_active_incident"
	CorrelationEventReopenedFlapping CorrelationEvent = "reopened_flapping_incident"
	CorrelationEventResolved         CorrelationEvent = "incident_resolved"
	CorrelationEventResolveIgnored   CorrelationEvent = "resolve_ignored_no_active_incident"
)

type CorrelationResult struct {
	IncidentID int64
	Event      CorrelationEvent
}

// CorrelationService decides what happens to an incident given a newly
// ingested alert: merge into an active incident, reopen a recently-resolved
// one (flapping), create a brand new one, or close an existing one.
//
// It is called once per alert AFTER AlertService.Ingest has already stored
// the raw alert row. It never touches the alerts table.
// Only incidents and the active/grace window state.
type CorrelationService struct {
	incidents   *repository.IncidentRepository
	correlation *repository.CorrelationRepository
}

func NewCorrelationService(incidents *repository.IncidentRepository, correlation *repository.CorrelationRepository) *CorrelationService {
	return &CorrelationService{incidents: incidents, correlation: correlation}
}

func (s *CorrelationService) Correlate(ctx context.Context, fingerprint string, identity model.AlertIdentity, resolved bool) (CorrelationResult, error) {
	if resolved {
		return s.handleResolve(ctx, fingerprint)
	}
	return s.handleFiring(ctx, fingerprint, identity)
}

func (s *CorrelationService) handleFiring(ctx context.Context, fingerprint string, identity model.AlertIdentity) (CorrelationResult, error) {
	// If already active, merge into it and slide the correlation window
	// forward - each new occurrence within the window keeps it alive longer,
	// rather than expiring on a fixed clock from the first occurrence.
	if incidentID, err := s.correlation.GetActiveIncident(ctx, fingerprint); err == nil {
		if err := s.incidents.IncrementOccurrence(ctx, incidentID); err != nil {
			return CorrelationResult{}, err
		}
		if err := s.correlation.SetActive(ctx, fingerprint, incidentID, identity.MetricType); err != nil {
			return CorrelationResult{}, err
		}
		return CorrelationResult{IncidentID: incidentID, Event: CorrelationEventMerged}, nil
	} else if !errors.Is(err, repository.ErrCorrelationStateNotFound) {
		return CorrelationResult{}, err
	}

	// Redis has no active window, but that might mean it lost its
	// state (restart, eviction), not that the window genuinely expired.
	// Check Postgres, the durable source of truth, before concluding this is a new incident.
	if incident, err := s.incidents.FindOpenByFingerprint(ctx, fingerprint); err == nil {
		if time.Since(incident.LastSeen) <= s.correlation.WindowFor(identity.MetricType) {
			if err := s.incidents.IncrementOccurrence(ctx, incident.ID); err != nil {
				return CorrelationResult{}, err
			}
			if err := s.correlation.SetActive(ctx, fingerprint, incident.ID, identity.MetricType); err != nil {
				return CorrelationResult{}, err
			}
			return CorrelationResult{IncidentID: incident.ID, Event: CorrelationEventMerged}, nil
		}
	} else if !errors.Is(err, repository.ErrIncidentNotFound) {
		return CorrelationResult{}, err
	}

	// Reopen the recently resolved and still within the flapping grace period
	if incidentID, err := s.correlation.GetGraceIncident(ctx, fingerprint); err == nil {
		if err := s.incidents.Reopen(ctx, incidentID); err != nil {
			return CorrelationResult{}, err
		}
		if err := s.correlation.SetActive(ctx, fingerprint, incidentID, identity.MetricType); err != nil {
			return CorrelationResult{}, err
		}
		return CorrelationResult{IncidentID: incidentID, Event: CorrelationEventReopenedFlapping}, nil
	} else if !errors.Is(err, repository.ErrCorrelationStateNotFound) {
		return CorrelationResult{}, err
	}

	// New incident
	incident, err := s.incidents.CreateOpen(ctx, fingerprint, identity)
	if err != nil {
		return CorrelationResult{}, err
	}
	if err := s.correlation.SetActive(ctx, fingerprint, incident.ID, identity.MetricType); err != nil {
		return CorrelationResult{}, err
	}
	return CorrelationResult{IncidentID: incident.ID, Event: CorrelationEventNewIncident}, nil
}

func (s *CorrelationService) handleResolve(ctx context.Context, fingerprint string) (CorrelationResult, error) {
	incidentID, err := s.correlation.GetActiveIncident(ctx, fingerprint)
	if errors.Is(err, repository.ErrCorrelationStateNotFound) {
		// No active window in Redis, but real alarms routinely outlast the
		// correlation window (and TCOP may push only at trigger and recovery).
		// An explicit resolve is authoritative, so fall back to Postgres
		// without any staleness check.
		incident, findErr := s.incidents.FindOpenByFingerprint(ctx, fingerprint)
		if errors.Is(findErr, repository.ErrIncidentNotFound) {
			return CorrelationResult{Event: CorrelationEventResolveIgnored}, nil
		}
		if findErr != nil {
			return CorrelationResult{}, findErr
		}
		incidentID = incident.ID
	} else if err != nil {
		return CorrelationResult{}, err
	}

	if err := s.incidents.MarkResolved(ctx, incidentID); err != nil {
		return CorrelationResult{}, err
	}
	if err := s.correlation.StartGrace(ctx, fingerprint, incidentID); err != nil {
		return CorrelationResult{}, err
	}
	return CorrelationResult{IncidentID: incidentID, Event: CorrelationEventResolved}, nil
}
