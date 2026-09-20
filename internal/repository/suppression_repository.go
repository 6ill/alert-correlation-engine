package repository

import (
	"context"

	"github.com/6ill/alert-correlation-engine/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SuppressionRepository struct {
	pool *pgxpool.Pool
}

func NewSuppressionRepository(pool *pgxpool.Pool) *SuppressionRepository {
	return &SuppressionRepository{pool: pool}
}

// FindCandidates returns every non-expired rule whose resource_type and
// metric_type match (a NULL match_* column matches anything). Time-window
// filtering is deliberately left to the caller - matching a wall-clock range
// that can cross midnight (e.g. "22:00-02:00") is awkward to express safely
// in plain SQL, and is easier to unit test as a small Go function.
func (r *SuppressionRepository) FindCandidates(ctx context.Context, resourceType, metricType string) ([]model.SuppressionRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, match_resource_type, match_metric_type, match_time_window, reason, expires, created_at
		FROM suppression_rules
		WHERE expires >= CURRENT_DATE
		  AND (match_resource_type IS NULL OR match_resource_type = $1)
		  AND (match_metric_type IS NULL OR match_metric_type = $2)

	`, resourceType, metricType)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.SuppressionRule, error) {
		var rule model.SuppressionRule
		err := row.Scan(
			&rule.ID, &rule.Name, &rule.MatchResourceType, &rule.MatchMetricType,
			&rule.MatchTimeWindow, &rule.Reason, &rule.Expires, &rule.CreatedAt,
		)
		return rule, err
	})
	if err != nil {
		return nil, err
	}

	return rules, err
}
