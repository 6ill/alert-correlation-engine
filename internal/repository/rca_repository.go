package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/6ill/alert-correlation-engine/internal/model"
)

type RCARepository struct {
	pool *pgxpool.Pool
}

func NewRCARepository(pool *pgxpool.Pool) *RCARepository {
	return &RCARepository{pool: pool}
}

func (r *RCARepository) Create(ctx context.Context, rca model.RCAKnowledge) (*model.RCAKnowledge, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO rca_knowledge (resource_type, metric_type, symptom_summary, root_cause, resolution)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, resource_type, metric_type, symptom_summary, root_cause, resolution, created_at
	`, rca.ResourceType, rca.MetricType, rca.SymptomSummary, rca.RootCause, rca.Resolution)
	return scanRCA(row)
}

// FindByTags returns past RCA entries tagged with the same resource_type and
// metric_type, most recent first. This is the entire "matching" logic for
// the MVP
func (r *RCARepository) FindByTags(ctx context.Context, resourceType, metricType string, limit int) ([]model.RCAKnowledge, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, resource_type, metric_type, symptom_summary, root_cause, resolution, created_at
		FROM rca_knowledge
		WHERE resource_type = $1 AND metric_type = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, resourceType, metricType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.RCAKnowledge, error) {
		var rca model.RCAKnowledge
		err := row.Scan(&rca.ID, &rca.ResourceType, &rca.MetricType, &rca.SymptomSummary, &rca.RootCause, &rca.Resolution, &rca.CreatedAt)
		return rca, err
	})
}

func scanRCA(row pgx.Row) (*model.RCAKnowledge, error) {
	var rca model.RCAKnowledge
	err := row.Scan(&rca.ID, &rca.ResourceType, &rca.MetricType, &rca.SymptomSummary, &rca.RootCause, &rca.Resolution, &rca.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rca, nil
}
