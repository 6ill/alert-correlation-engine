package model

// Stats are the noise-reduction metrics.
// TicketsSaved is how many ingested alerts did NOT result in their own
// incident (merged, reopened, or suppressed) instead of a brand new ticket.
type Stats struct {
	TotalAlerts        int64   `json:"total_alerts"`
	TotalIncidents     int64   `json:"total_incidents"`
	TicketsSaved       int64   `json:"tickets_saved"`
	NoiseReductionRate float64 `json:"noise_reduction_rate"`
}
