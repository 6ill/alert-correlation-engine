package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/6ill/alert-correlation-engine/internal/model"
)

type AlertRepository struct {
	pool *pgxpool.Pool
}

func NewAlertRepository(pool *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{pool: pool}
}

// Return inserted=false if raw_alert_id exists
func (r *AlertRepository) Insert(ctx context.Context, a model.Alert, fingerprint string, rawPayload []byte) (inserted bool, err error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO alerts (raw_alert_id, account, resource_id, resource_type, metric_type, value, severity, fingerprint, payload, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (raw_alert_id) DO NOTHING
	`, a.RawAlertID, a.Account, a.ResourceID, a.ResourceType, a.MetricType, a.Value, a.Severity, fingerprint, rawPayload, a.Timestamp)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FindRecentByResource returns alerts recently received on the same
// account+resource, excluding the fingerprint already shown as the
// incident's own occurrences - used to surface "other alarms nearby" on an
// incident's detail view without doing full cross-metric correlation.
func (r *AlertRepository) FindRecentByResource(ctx context.Context, account, resourceID, excludeFingerprint string, since time.Time, limit int) ([]model.AlertSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT raw_alert_id, metric_type, value, severity, received_at
		FROM alerts
		WHERE account = $1 AND resource_id = $2 AND fingerprint != $3 AND received_at >= $4
		ORDER BY received_at DESC
		LIMIT $5
	`, account, resourceID, excludeFingerprint, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.AlertSummary, error) {
		var a model.AlertSummary
		err := row.Scan(&a.RawAlertID, &a.MetricType, &a.Value, &a.Severity, &a.ReceivedAt)
		return a, err
	})
}
