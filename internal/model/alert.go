package model

import "time"

type Alert struct {
	RawAlertID   string    `json:"raw_alert_id"`
	Account      string    `json:"account"`
	ResourceID   string    `json:"resource_id"`
	ResourceType string    `json:"resource_type"`
	MetricType   string    `json:"metric_type"`
	Value        float64   `json:"value"`
	Severity     string    `json:"severity"`
	Timestamp    time.Time `json:"timestamp"`
}
