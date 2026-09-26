"""
Synthetic alert generator for testing the ingestion, suppression, and
correlation pipeline end to end.

Usage: python scripts/generate_alerts.py
Requires: pip install requests

Before running the suppression tests, insert the seed rule:

    INSERT INTO suppression_rules (name, match_resource_type, match_metric_type, match_time_window, reason, expires)
    VALUES ('nightly-cleanup-disk-read-spike', 'cvm', 'disk_read', '00:00-03:00',
            'Known scheduled cleanup cronjob, not an incident', CURRENT_DATE + INTERVAL '90 days');
"""

import time
import uuid

import requests

API_URL = "http://localhost:8080/api/v1/alerts/"

SCENARIOS = [
    {
        "account": "demo_prod",
        "resource_id": "cvm-web-01",
        "resource_type": "cvm",
        "metric_type": "disk_util",
        "value": 92.4,
        "severity": "warning",
    },
    {
        "account": "demo_prod",
        "resource_id": "cvm-etl-01",
        "resource_type": "cvm",
        "metric_type": "disk_read",
        "value": 87.0,
        "severity": "warning",
    },
]


def send(payload, raw_id=None):
    body = dict(payload)
    body["raw_alert_id"] = raw_id or str(uuid.uuid4())
    resp = requests.post(API_URL, json=body, timeout=5)
    print(f"{resp.status_code} {resp.json()}")
    return body["raw_alert_id"]


def section(title):
    print(f"\n== {title} ==")


if __name__ == "__main__":
    section("Normal alerts (each should be a new_incident)")
    for scenario in SCENARIOS:
        send(scenario)
        time.sleep(0.3)

    section("Retry webhook simulation (same raw_alert_id sent twice)")
    print("Second send must be 'duplicate_ignored', not a new row.")
    dup_id = send(SCENARIOS[0])
    send(SCENARIOS[0], raw_id=dup_id)

    section("Repeated fingerprint, different raw_alert_id each time")
    print("First send here is new_incident; the next two must be")
    print("'merged_into_active_incident' with the same incident_id.")
    for _ in range(3):
        send(SCENARIOS[0])
        time.sleep(0.5)

    section("Suppression match (timestamp inside the 00:00-03:00 window)")
    print("Must return 'suppressed_known_pattern' - no incident created.")
    suppressed_alert = {
        "account": "demo_prod",
        "resource_id": "cvm-cleanup-job-01",
        "resource_type": "cvm",
        "metric_type": "disk_read",
        "value": 88.5,
        "severity": "warning",
        "timestamp": "2025-01-01T01:30:00+07:00",  # 01:30 WIB, inside the window
    }
    send(suppressed_alert)

    section("Same resource/metric, but outside the suppression window")
    print("Must go through normal correlation (new_incident), not suppressed.")
    not_suppressed_alert = dict(suppressed_alert, timestamp="2025-01-01T10:00:00+07:00")
    send(not_suppressed_alert)

    section("Resolve, then flapping reopen within the grace period")
    flap_scenario = {
        "account": "demo_prod",
        "resource_id": "redis-flap-test-01",
        "resource_type": "redis",
        "metric_type": "memory_util",
        "value": 98.1,
        "severity": "critical",
    }
    print("1) firing -> new_incident")
    send(flap_scenario)
    time.sleep(0.5)

    print("2) resolved -> incident_resolved")
    send(dict(flap_scenario, status="resolved"))
    time.sleep(0.5)

    print("3) firing again quickly -> reopened_flapping_incident (same incident_id as step 1)")
    send(flap_scenario)

    print(
        "\nNote: the flapping grace period is 15 minutes. This script cannot "
        "wait that long automatically - to verify the 'grace period expired, "
        "genuinely new incident' path, resend this same scenario by hand "
        "after 15+ minutes and confirm it comes back as 'new_incident'."
    )