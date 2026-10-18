package service

import (
	"context"
	"log/slog"

	"github.com/6ill/alert-correlation-engine/internal/model"
)

type PipelineResult struct {
	Status            string               `json:"status"` // "ingested" | "duplicate_ignored" | "suppressed_known_pattern"
	Fingerprint       string               `json:"fingerprint"`
	IncidentID        int64                `json:"incident_id,omitempty"`
	CorrelationEvent  CorrelationEvent     `json:"correlation_event,omitempty"`
	SuppressionRule   string               `json:"suppression_rule,omitempty"`
	SuppressionReason string               `json:"suppression_reason,omitempty"`
	Recurrence        *model.Recurrence    `json:"recurrence,omitempty"`
	RCASuggestions    []model.RCAKnowledge `json:"rca_suggestions,omitempty"`
}

// AlertPipelineService is the single place where "what happens to an incoming
// alert" is decided, regardless of which channel it arrived through (the
// generic /alerts endpoint used for testing, or a real cloud provider's
// webhook). Each ingestion channel is responsible only for translating its
// own payload format into model.Alert - everything after that goes through
// this one pipeline.
type AlertPipelineService struct {
	alertSvc       *AlertService
	suppressionSvc *SuppressionService
	correlationSvc *CorrelationService
	rcaSvc         *RCAService
	incidentSvc    *IncidentService
}

func NewAlertPipelineService(
	alertSvc *AlertService,
	suppressionSvc *SuppressionService,
	correlationSvc *CorrelationService,
	rcaSvc *RCAService,
	incidentSvc *IncidentService,
) *AlertPipelineService {
	return &AlertPipelineService{
		alertSvc:       alertSvc,
		suppressionSvc: suppressionSvc,
		correlationSvc: correlationSvc,
		rcaSvc:         rcaSvc,
		incidentSvc:    incidentSvc,
	}
}

func (p *AlertPipelineService) Process(ctx context.Context, a model.Alert, rawPayload []byte) (PipelineResult, error) {
	ingestResult, err := p.alertSvc.Ingest(ctx, a, rawPayload)
	if err != nil {
		return PipelineResult{}, err
	}
	if ingestResult.Status == "duplicate_ignored" {
		return PipelineResult{Status: ingestResult.Status, Fingerprint: ingestResult.Fingerprint}, nil
	}

	if a.Status == model.AlertStatusFiring {
		rule, err := p.suppressionSvc.Match(ctx, a.ResourceType, a.MetricType, a.Timestamp)
		if err != nil {
			return PipelineResult{}, err
		}
		if rule != nil {
			return PipelineResult{
				Status:            "suppressed_known_pattern",
				Fingerprint:       ingestResult.Fingerprint,
				SuppressionRule:   rule.Name,
				SuppressionReason: rule.Reason,
			}, nil
		}
	}

	correlationResult, err := p.correlationSvc.Correlate(
		ctx,
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
		return PipelineResult{}, err
	}

	result := PipelineResult{
		Status:           ingestResult.Status,
		Fingerprint:      ingestResult.Fingerprint,
		IncidentID:       correlationResult.IncidentID,
		CorrelationEvent: correlationResult.Event,
	}

	if correlationResult.Event == CorrelationEventNewIncident {
		suggestions, err := p.rcaSvc.Suggest(ctx, a.ResourceType, a.MetricType)
		if err != nil {
			slog.Warn("failed to load RCA suggestions", "error", err, "incident_id", correlationResult.IncidentID)
		} else {
			result.RCASuggestions = suggestions
		}

		recurrence, err := p.incidentSvc.Recurrence(ctx, correlationResult.IncidentID)
		if err != nil {
			slog.Warn("failed to load incident recurrence", "error", err, "incident_id", correlationResult.IncidentID)
		} else {
			result.Recurrence = &recurrence
		}
	}

	return result, nil
}
