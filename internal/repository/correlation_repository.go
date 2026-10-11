package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCorrelationStateNotFound means no active window (or grace window) exists
// in Redis for the given fingerprint. For an active-window lookup, this tells
// the caller to create a brand new incident. For a grace-window lookup, this
// tells the caller there is nothing to reopen a resolve event for a
// fingerprint with no recent incident is simply a no-op.
var ErrCorrelationStateNotFound = errors.New("correlation state not found")

// correlationWindows maps a metric type to how long its correlation window
// stays active before a repeat alert is treated as a new incident instead of
// being merged into the existing one.
var correlationWindows = map[string]time.Duration{
	"disk_util":   15 * time.Minute,
	"disk_read":   15 * time.Minute,
	"cpu_usage":   1 * time.Hour,
	"memory_util": 4 * time.Hour,
}

const (
	// defaultCorrelationWindow applies to any metric_type not explicitly
	// tuned in correlationWindows above.
	defaultCorrelationWindow = 1 * time.Hour

	// flappingGraceWindow is how long a resolved incident stays reopenable
	// before a repeat alert is treated as a genuinely new incident.
	flappingGraceWindow = 15 * time.Minute
)

// CorrelationRepository stores the ephemeral "is this fingerprint currently
// active / recently resolved" state in Redis. Postgres (IncidentRepository)
// remains the source of truth for full incident history; this repository
// only answers the single question the correlation engine asks on every
// alert: "does this fingerprint already have somewhere to go?"
type CorrelationRepository struct {
	client *redis.Client
}

func NewCorrelationRepository(client *redis.Client) *CorrelationRepository {
	return &CorrelationRepository{client: client}
}

func activeKey(fingerprint string) string { return "active:" + fingerprint }
func graceKey(fingerprint string) string  { return "grace:" + fingerprint }

func (r *CorrelationRepository) WindowFor(metricType string) time.Duration {
	if d, ok := correlationWindows[metricType]; ok {
		return d
	}
	return defaultCorrelationWindow
}

// GetActiveIncident returns the incident ID currently active for this
// fingerprint, if its correlation window has not yet expired.
func (r *CorrelationRepository) GetActiveIncident(ctx context.Context, fingerprint string) (int64, error) {
	return r.getIncidentID(ctx, activeKey(fingerprint))
}

// SetActive (re)starts the active correlation window for a fingerprint,
// pointing it at incidentID. The TTL depends on metricType - once it
// expires, the next alert with this fingerprint is treated as a new
// incident rather than merged into this one.
func (r *CorrelationRepository) SetActive(ctx context.Context, fingerprint string, incidentID int64, metricType string) error {
	return r.client.Set(ctx, activeKey(fingerprint), incidentID, r.WindowFor(metricType)).Err()
}

// GetGraceIncident returns the incident ID still within its flapping grace
// period for this fingerprint, if any.
func (r *CorrelationRepository) GetGraceIncident(ctx context.Context, fingerprint string) (int64, error) {
	return r.getIncidentID(ctx, graceKey(fingerprint))
}

// StartGrace is called when an incident resolves. It clears the active
// window (the incident is no longer "ongoing") and opens a short grace
// window instead. If the same fingerprint fires again before the grace
// window expires, the correlation service treats it as flapping and
// reopens this same incident rather than creating a new one.
func (r *CorrelationRepository) StartGrace(ctx context.Context, fingerprint string, incidentID int64) error {
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, activeKey(fingerprint))
	pipe.Set(ctx, graceKey(fingerprint), incidentID, flappingGraceWindow)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *CorrelationRepository) getIncidentID(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrCorrelationStateNotFound
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}
