-- Supports the recurrence queries (earlier incidents of the same fingerprint,
-- by first_seen). The existing (fingerprint, status, updated_at) index serves
-- status/recency lookups, but its third column doesn't help with ranges or
-- ordering on first_seen.
CREATE INDEX idx_incidents_fingerprint_first_seen
    ON incidents (fingerprint, first_seen DESC);