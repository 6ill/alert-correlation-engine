#!/usr/bin/env bash
# Manual test script for the TCOP webhook endpoint.
# Usage: TCOP_WEBHOOK_TOKEN=xxx ./scripts/test_tcop_webhook.sh
# Requires: curl (jq optional, for pretty-printing)

set -euo pipefail

TOKEN="${TCOP_WEBHOOK_TOKEN:?Set TCOP_WEBHOOK_TOKEN first}"
BASE_URL="${BASE_URL:-http://localhost:8080}"
URL="$BASE_URL/api/v1/webhooks/tcop/$TOKEN"

# Unique per run, so re-running this script never collides with a previous
# run's raw_alert_id values (unlike real TCOP sessionIds, which are already
# unique per genuine alarm event - this suffix only exists for this script).
RUN_ID="$(date +%s)"

post() {
  echo "--- $1 ---"
  curl -s -X POST "$URL" -H "Content-Type: application/json" -d "$2" | (jq . 2>/dev/null || cat)
  echo
}

echo "== 1. Firing alert (CVM CPU) - expect new_incident =="
post "firing" '{
  "sessionId": "test-session-firing-001-'"$RUN_ID"'",
  "alarmStatus": "1",
  "alarmType": "metric",
  "alarmObjInfo": {
    "region": "ap-jakarta",
    "namespace": "qce/cvm",
    "appId": "100000000001",
    "uin": "100000000001",
    "dimensions": {
      "unInstanceId": "ins-test0001",
      "objId": "ins-test0001",
      "objName": "demo-cvm-01"
    }
  },
  "alarmPolicyInfo": {
    "policyId": "policy-001",
    "policyType": "metric",
    "policyName": "CPU usage too high",
    "conditions": {
      "metricName": "CpuUsage",
      "metricShowName": "CPU Utilization",
      "calcType": "static_gt",
      "calcValue": "90",
      "calcUnit": "%",
      "currentValue": "95.2",
      "historyValue": "60.1"
    }
  }
}'

echo "== 2. Same alert fires again shortly after - expect merged_into_active_incident =="
post "firing again" '{
  "sessionId": "test-session-firing-002-'"$RUN_ID"'",
  "alarmStatus": "1",
  "alarmType": "metric",
  "alarmObjInfo": {
    "region": "ap-jakarta",
    "namespace": "qce/cvm",
    "appId": "100000000001",
    "uin": "100000000001",
    "dimensions": {
      "unInstanceId": "ins-test0001",
      "objId": "ins-test0001",
      "objName": "demo-cvm-01"
    }
  },
  "alarmPolicyInfo": {
    "policyId": "policy-001",
    "policyType": "metric",
    "policyName": "CPU usage too high",
    "conditions": {
      "metricName": "CpuUsage",
      "metricShowName": "CPU Utilization",
      "calcType": "static_gt",
      "calcValue": "90",
      "calcUnit": "%",
      "currentValue": "97.8",
      "historyValue": "95.2"
    }
  }
}'

echo "== 3. Resolve event (same instance+metric, different sessionId) - expect incident_resolved =="
post "resolved" '{
  "sessionId": "test-session-resolved-001-'"$RUN_ID"'",
  "alarmStatus": "0",
  "alarmType": "metric",
  "alarmObjInfo": {
    "region": "ap-jakarta",
    "namespace": "qce/cvm",
    "appId": "100000000001",
    "uin": "100000000001",
    "dimensions": {
      "unInstanceId": "ins-test0001",
      "objId": "ins-test0001",
      "objName": "demo-cvm-01"
    }
  },
  "alarmPolicyInfo": {
    "policyId": "policy-001",
    "policyType": "metric",
    "policyName": "CPU usage too high",
    "conditions": {
      "metricName": "CpuUsage",
      "metricShowName": "CPU Utilization",
      "calcType": "static_gt",
      "calcValue": "90",
      "calcUnit": "%",
      "currentValue": "40.0",
      "historyValue": "97.8"
    }
  }
}'

echo "== 4. Wrong token - expect HTTP 404 =="
curl -s -o /dev/null -w "HTTP %{http_code}\n" -X POST "$BASE_URL/api/v1/webhooks/tcop/wrong-token" \
  -H "Content-Type: application/json" -d '{}'

echo "== 5. Malformed payload (missing metricName) - expect HTTP 400 =="
curl -s -w "\nHTTP %{http_code}\n" -X POST "$URL" \
  -H "Content-Type: application/json" \
  -d '{"sessionId":"test-malformed-'"$RUN_ID"'","alarmStatus":"1","alarmObjInfo":{"namespace":"qce/cvm","dimensions":{}},"alarmPolicyInfo":{"conditions":{}}}'