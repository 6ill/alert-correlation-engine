package model

import "time"

type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

type Alert struct {
	RawAlertID   string      `json:"raw_alert_id"`
	Account      string      `json:"account"`
	ResourceID   string      `json:"resource_id"`
	ResourceType string      `json:"resource_type"`
	MetricType   string      `json:"metric_type"`
	Value        float64     `json:"value"`
	Severity     string      `json:"severity"`
	Timestamp    time.Time   `json:"timestamp"`
	Status       AlertStatus `json:"status"`
}

// AlertSummary is a lightweight read model for displaying an alert's past
// occurrences (e.g. "related events" on an incident's detail view)..
type AlertSummary struct {
	RawAlertID string    `json:"raw_alert_id"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Severity   string    `json:"severity"`
	ReceivedAt time.Time `json:"received_at"`
}
