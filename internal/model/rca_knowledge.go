package model

import "time"

type RCAKnowledge struct {
	ID             int64     `json:"id"`
	ResourceType   string    `json:"resource_type"`
	MetricType     string    `json:"metric_type"`
	SymptomSummary string    `json:"symptom_summary"`
	RootCause      string    `json:"root_cause"`
	Resolution     string    `json:"resolution"`
	CreatedAt      time.Time `json:"created_at"`
}
