package tcop

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/6ill/alert-correlation-engine/internal/model"
)

// resourceIDKeys lists, per namespace, the dimension keys known to carry the
// instance ID, in priority order. "unInstanceId" for CVM is confirmed from a
// real payload; the others come from Tencent's docs. Every other namespace
// falls back to "objId".
var resourceIDKeys = map[string][]string{
	"qce/cvm":       {"unInstanceId"},
	"qce/cdb":       {"uInstanceId"},
	"qce/redis":     {"instanceid", "InstanceId"},
	"qce/redis_mem": {"instanceid", "InstanceId"},
}

const fallbackResourceIDKey = "objId"

// Translate converts a TCOP alarm callback payload into internal Alert
// model. It only errors on a payload missing the one field we truly cannot
// proceed without (the metric name) or everything else degrades gracefully
// with a documented assumption, since a unexpected field from a real cloud
// provider shouldn't drop the whole alert.
func Translate(p Payload) (model.Alert, error) {
	if p.SessionID == "" {
		return model.Alert{}, fmt.Errorf("tcop payload missing sessionId")
	}
	if p.AlarmPolicyInfo.Conditions.MetricName == "" {
		return model.Alert{}, fmt.Errorf("tcop payload missing alarmPolicyInfo.conditions.metricName")
	}

	status := model.AlertStatusFiring
	if p.AlarmStatus == "0" {
		status = model.AlertStatusResolved
	}

	value, _ := strconv.ParseFloat(string(p.AlarmPolicyInfo.Conditions.CurrentValue), 64)

	account := string(p.AlarmObjInfo.Uin)
	if account == "" {
		account = string(p.AlarmObjInfo.AppID)
	}

	return model.Alert{
		RawAlertID:   rawAlertID(p),
		Account:      account,
		ResourceID:   resolveResourceID(p.AlarmObjInfo.Namespace, p.AlarmObjInfo.Dimensions),
		ResourceType: strings.TrimPrefix(p.AlarmObjInfo.Namespace, "qce/"),
		MetricType:   p.AlarmPolicyInfo.Conditions.MetricName,
		Value:        value,
		Severity:     "warning",
		Status:       status,
		Timestamp:    time.Now(),
	}, nil
}

// rawAlertID builds the idempotency key. sessionId alone identifies the whole
// alarm record, so a recovery notification likely shares it with the firing
// one and would be dropped as a duplicate. Adding status, durationTime and
// recoverTime makes the key unique per notification, while a webhook retry of
// the same notification still produces an identical key.
func rawAlertID(p Payload) string {
	return fmt.Sprintf("%s:%s:%s:%s", p.SessionID, p.AlarmStatus, p.DurationTime, p.RecoverTime)
}

func resolveResourceID(namespace string, dimensions map[string]flexString) string {
	for _, key := range resourceIDKeys[namespace] {
		if v, ok := dimensions[key]; ok && v != "" {
			return string(v)
		}
	}
	if v, ok := dimensions[fallbackResourceIDKey]; ok {
		return string(v)
	}

	return ""
}
