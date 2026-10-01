package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepository struct {
	pool *pgxpool.Pool
}

func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{pool: pool}
}

func (r *StatsRepository) RawCounts(ctx context.Context) (totalAlerts, totalIncidents int64, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM alerts) AS total_alerts,
			(SELECT COUNT(*) FROM incidents) AS total_incidents
	`).Scan(&totalAlerts, &totalIncidents)
	return
}
