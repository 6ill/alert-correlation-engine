package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/6ill/alert-correlation-engine/internal/model"
)

var ErrIncidentNotFound = errors.New("incident not found")

type IncidentRepository struct {
	pool *pgxpool.Pool
}

func NewIncidentRepository(pool *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{pool: pool}
}

// CreateOpen inserts a brand new open incident for a fingerprint that has
// no active correlation window in Redis. occurrence_count and flap_count
// start at their table defaults (1 and 0).
func (r *IncidentRepository) CreateOpen(ctx context.Context, fingerprint string, identity model.AlertIdentity) (*model.Incident, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO incidents (fingerprint, account, resource_id, resource_type, metric_type, status)
		VALUES ($1, $2, $3, $4, $5, 'open')
		RETURNING id, fingerprint, account, resource_id, resource_type, metric_type, status, occurrence_count, flap_count, first_seen, last_seen, resolved_at, updated_at
	`, fingerprint, identity.Account, identity.ResourceID, identity.ResourceType, identity.MetricType)
	return scanIncident(row)
}

func (r *IncidentRepository) GetByID(ctx context.Context, id int64) (*model.Incident, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, fingerprint, account, resource_id, resource_type, metric_type, status, occurrence_count, flap_count, first_seen, last_seen, resolved_at, updated_at
		FROM incidents WHERE id = $1
	`, id)
	return scanIncident(row)
}

// IncrementOccurrence bumps occurrence_count and last_seen for an incident
// that is still within its active correlation window (same fingerprint,
// still open, but not a flap).
func (r *IncidentRepository) IncrementOccurrence(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE incidents
		SET occurrence_count = occurrence_count + 1,
		    last_seen = now(),
		    updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

// Reopen brings a resolved incident back to open status because a new alert
// with the same fingerprint arrived within the flapping grace period.
// flap_count is incremented to make flapping visible on the dashboard.
func (r *IncidentRepository) Reopen(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE incidents
		SET status = 'open',
		    resolved_at = NULL,
		    flap_count = flap_count + 1,
		    occurrence_count = occurrence_count + 1,
		    last_seen = now(),
		    updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

// MarkResolved closes an incident. Called when an explicit "resolved" alert
// arrives for its fingerprint.
func (r *IncidentRepository) MarkResolved(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE incidents
		SET status = 'resolved',
		    resolved_at = now(),
		    updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

func scanIncident(row pgx.Row) (*model.Incident, error) {
	var inc model.Incident
	err := row.Scan(
		&inc.ID, &inc.Fingerprint, &inc.Account, &inc.ResourceID, &inc.ResourceType, &inc.MetricType,
		&inc.Status, &inc.OccurrenceCount, &inc.FlapCount,
		&inc.FirstSeen, &inc.LastSeen, &inc.ResolvedAt, &inc.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrIncidentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inc, nil
}
