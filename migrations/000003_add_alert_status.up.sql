-- Stats must count only firing alerts: a recovery event is not noise, and
-- counting it halves the apparent noise reduction of every normal episode.
ALTER TABLE alerts ADD COLUMN status TEXT NOT NULL DEFAULT 'firing';

-- Backfill from stored payloads: TCOP uses alarmStatus "0", the generic channel uses status "resolved".
UPDATE alerts SET status = 'resolved'
WHERE payload->>'alarmStatus' = '0' OR payload->>'status' = 'resolved';