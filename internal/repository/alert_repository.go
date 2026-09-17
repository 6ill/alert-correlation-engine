package repository

import (
	"context"

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
