package model

// AlertIdentity carries the human-readable identity fields of an alert
// (as opposed to its opaque fingerprint hash). Passed through the
// correlation engine so a newly created Incident can be denormalized with
// enough context for the dashboard and RCA matching to use directly,
// without joining back to the alerts table on every read.
type AlertIdentity struct {
	Account      string
	ResourceID   string
	ResourceType string
	MetricType   string
}
