-- The incidents table only stored the fingerprint hash, which is opaque -
-- useless for a dashboard or for matching RCA suggestions. This adds
-- human-readable identity columns, denormalized from the triggering alert
-- at incident-creation time (not re-derived on every read).
ALTER TABLE incidents
    ADD COLUMN account TEXT NOT NULL DEFAULT '',
    ADD COLUMN resource_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN resource_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN metric_type TEXT NOT NULL DEFAULT '';