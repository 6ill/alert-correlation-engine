"""
Generator alarm sintetis untuk testing endpoint ingestion.
Meniru pola alarm nyata (bukan data asli kantor) + skenario retry webhook
dan alarm berulang, untuk memverifikasi idempotensi minggu 1.

Pakai: API_AUTH=admin:password python scripts/generate_alerts.py
Butuh: pip install requests
"""

import time
import uuid
import os

import requests
API_AUTH = os.environ.get("API_AUTH")  # "user:password"
AUTH = tuple(API_AUTH.split(":", 1)) if API_AUTH else None
API_URL = "http://localhost:8080/api/v1/alerts"

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
        "resource_id": "redis-cache-01",
        "resource_type": "redis",
        "metric_type": "memory_util",
        "value": 98.6,
        "severity": "critical",
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
    resp = requests.post(API_URL, auth=AUTH, json=body, timeout=5)
    print(f"{resp.status_code} {resp.json()}")
    return body["raw_alert_id"]


if __name__ == "__main__":
    print("== Sending normal alerts ==")
    for scenario in SCENARIOS:
        send(scenario)
        time.sleep(0.3)

    print("\n==Retry webhook simulation (duplicated raw_alert_id , sent 2x) ==")
    print("Second alert should be'duplicate_ignored' and not sent.")
    dup_id = send(SCENARIOS[0])
    send(SCENARIOS[0], raw_id=dup_id)

    print("\n==Repeated alarm simulation (same fingerprint, but different raw_alert_id) ==")
    for _ in range(3):
        send(SCENARIOS[1])
        time.sleep(0.5)
