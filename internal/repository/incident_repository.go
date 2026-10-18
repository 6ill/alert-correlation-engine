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

type rowScanner interface {
	Scan(dest ...any) error
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

func scanIncident(row rowScanner) (*model.Incident, error) {
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

// List returns incidents ordered by most recently updated first, optionally
// filtered by status ("" means no filter). Powers the dashboard's incident list view.
func (r *IncidentRepository) List(ctx context.Context, status string, limit int) ([]model.Incident, error) {
	const columns = `id, fingerprint, account, resource_id, resource_type, metric_type, status, occurrence_count, flap_count, first_seen, last_seen, resolved_at, updated_at`

	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.pool.Query(ctx, `SELECT `+columns+` FROM incidents ORDER BY updated_at DESC LIMIT $1`, limit)
	} else {
		rows, err = r.pool.Query(ctx, `SELECT `+columns+` FROM incidents WHERE status = $1 ORDER BY updated_at DESC LIMIT $2`, status, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Incident, error) {
		inc, err := scanIncident(row)
		if err != nil {
			return model.Incident{}, err
		}
		return *inc, nil
	})
}

// FindOpenByFingerprint looks up a still-open incident by fingerprint. Used
// as the Redis fallback path when the active-window key is missing - either
// because the window genuinely expired, or because Redis lost its state
// (restart, eviction). The caller is responsible for distinguishing the two
// by comparing LastSeen against the configured correlation window.
func (r IncidentRepository) FindOpenByFingerprint(ctx context.Context, fingerprint string) (*model.Incident, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, fingerprint, account, resource_id, resource_type, metric_type, status, occurrence_count, flap_count, first_seen, last_seen, resolved_at, updated_at
		FROM incidents
		WHERE status = 'open' AND fingerprint = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, fingerprint)
	return scanIncident(row)
}

// Recurrence counts incidents that share this incident's fingerprint and
// started before it. Those within windowDays before it began, and all of
// them. A single query anchored on the incident ID, so callers never need the
// fingerprint or start time. An aggregate over zero joined rows still returns
// one row (0, 0, NULL), which is exactly the "first time ever" answer.
func (r *IncidentRepository) Recurrence(ctx context.Context, incidentID int64, windowDays int) (model.Recurrence, error) {
	rec := model.Recurrence{WindowDays: windowDays}
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE prev.first_seen >= cur.first_seen - make_interval(days => $2::int)),
			COUNT(*),
			MAX(prev.first_seen)
		FROM incidents cur
		JOIN incidents prev
			ON prev.fingerprint = cur.fingerprint AND prev.first_seen < cur.first_seen
		WHERE cur.id = $1
	`, incidentID, windowDays).Scan(&rec.CountInWindow, &rec.TotalPrevious, &rec.LastIncidentAt)
	return rec, err
}

// ListPrevious returns earlier incidents with the same fingerprint, newest first.
func (r *IncidentRepository) ListPrevious(ctx context.Context, incidentID int64, limit int) ([]model.IncidentSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, status, occurrence_count, flap_count, first_seen, last_seen, resolved_at
		FROM incidents
		WHERE fingerprint = (SELECT fingerprint FROM incidents WHERE id = $1)
		  AND first_seen < (SELECT first_seen FROM incidents WHERE id = $1)
		ORDER BY first_seen DESC
		LIMIT $2
	`, incidentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.IncidentSummary, error) {
		var s model.IncidentSummary
		err := row.Scan(&s.ID, &s.Status, &s.OccurrenceCount, &s.FlapCount, &s.FirstSeen, &s.LastSeen, &s.ResolvedAt)
		return s, err
	})
}
