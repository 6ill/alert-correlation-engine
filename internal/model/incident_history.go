package model

import "time"

// IncidentSummary is a lightweight read model of an earlier incident that
// shares a fingerprint with the one being viewed.
type IncidentSummary struct {
	ID              int64          `json:"id"`
	Status          IncidentStatus `json:"status"`
	OccurrenceCount int            `json:"occurrence_count"`
	FlapCount       int            `json:"flap_count"`
	FirstSeen       time.Time      `json:"first_seen"`
	LastSeen        time.Time      `json:"last_seen"`
	ResolvedAt      *time.Time     `json:"resolved_at,omitempty"`
}

// Recurrence summarizes how often a fingerprint produced incidents before a
// given incident started. Flapping reopens are not counted because they are the same
// incident, not a new episode.
type Recurrence struct {
	WindowDays     int        `json:"window_days"`
	CountInWindow  int        `json:"count_in_window"`
	TotalPrevious  int        `json:"total_previous"`
	LastIncidentAt *time.Time `json:"last_incident_at,omitempty"` // first_seen of the most recent earlier incident
}
