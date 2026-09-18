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
	ID              int64
	Fingerprint     string
	Status          IncidentStatus
	OccurrenceCount int
	FlapCount       int
	FirstSeen       time.Time
	LastSeen        time.Time
	ResolvedAt      *time.Time
	UpdatedAt       time.Time
}
