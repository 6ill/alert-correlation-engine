package tcop

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/6ill/alert-correlation-engine/internal/model"
)

// resourceIDKeys lists, per namespace, the dimension keys known to carry the
// instance ID, in priority order. Confirmed for CVM, MySQL (CDB), and Redis
// only - every other namespace falls back to "objId" (present in every
// documented sample, though it's a composite backend identifier rather than
// a clean instance ID). Extend this table as real alarms from other
// services are observed.
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
	if p.AlarmPolicyInfo.Conditions.MetricName == "" {
		return model.Alert{}, fmt.Errorf("tcop payload missing alarmPolicyInfo.conditions.metricName")
	}

	status := model.AlertStatusFiring
	if p.AlarmStatus == "0" {
		status = model.AlertStatusResolved
	}

	value, _ := strconv.ParseFloat(p.AlarmPolicyInfo.Conditions.CurrentValue, 64)

	account := p.AlarmObjInfo.Uin
	if account == "" {
		account = p.AlarmObjInfo.AppID
	}

	return model.Alert{
		RawAlertID:   p.SessionID,
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

func resolveResourceID(namespace string, dimensions map[string]string) string {
	for _, key := range resourceIDKeys[namespace] {
		if v, ok := dimensions[key]; ok && v != "" {
			return v
		}
	}

	if v, ok := dimensions[fallbackResourceIDKey]; ok && v != "" {
		return v
	}

	return ""
}
