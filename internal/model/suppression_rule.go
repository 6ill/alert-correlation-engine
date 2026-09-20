package model

import "time"

// SuppressionRule represents a known, expected alert pattern that should not
// spawn an incident (e.g. a nightly cleanup cronjob that always spikes disk
// reads between 00:00-03:00). A nil Match* field means "matches any value"
// for that dimension.
type SuppressionRule struct {
	ID                int64
	Name              string
	MatchResourceType *string
	MatchMetricType   *string
	MatchTimeWindow   *string // e.g. "00:00-03:00"; nil means any time of day
	Reason            string
	Expires           time.Time
	CreatedAt         time.Time
}
