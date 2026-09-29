package model

import "time"

type IncidentStatus string

const (
	IncidentStatusOpen     IncidentStatus = "open"
	IncidentStatusResolved IncidentStatus = "resolved"
)

// Incident is the result of correlating one or more raw alerts that share
// the same fingerprint (account + resource + metric type).
type Incident struct {
	ID              int64          `json:"id"`
	Fingerprint     string         `json:"fingerprint"`
	Account         string         `json:"account"`
	ResourceID      string         `json:"resource_id"`
	ResourceType    string         `json:"resource_type"`
	MetricType      string         `json:"metric_type"`
	Status          IncidentStatus `json:"status"`
	OccurrenceCount int            `json:"occurrence_count"`
	FlapCount       int            `json:"flap_count"`
	FirstSeen       time.Time      `json:"first_seen"`
	LastSeen        time.Time      `json:"last_seen"`
	ResolvedAt      *time.Time     `json:"resolved_at,omitempty"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
