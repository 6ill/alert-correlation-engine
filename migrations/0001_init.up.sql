CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    raw_alert_id TEXT NOT NULL UNIQUE,
    account TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    metric_type TEXT NOT NULL,
    value NUMERIC,
    severity TEXT,
    fingerprint TEXT NOT NULL,
    payload JSONB,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_alerts_fingerprint ON alerts (fingerprint);

CREATE TABLE incidents (
    id BIGSERIAL PRIMARY KEY,
    fingerprint TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open', -- 'open' | 'resolved'
    occurrence_count INT NOT NULL DEFAULT 1,
    flap_count INT NOT NULL DEFAULT 0,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_incidents_fingerprint_status_updated
    ON incidents (fingerprint, status, updated_at);

CREATE TABLE suppression_rules (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    match_resource_type TEXT,
    match_metric_type TEXT,
    match_time_window TEXT,
    reason TEXT,
    expires DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rca_knowledge (
    id BIGSERIAL PRIMARY KEY,
    resource_type TEXT NOT NULL,
    metric_type TEXT NOT NULL,
    symptom_summary TEXT,
    root_cause TEXT,
    resolution TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
