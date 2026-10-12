package tcop

import (
	"bytes"
	"encoding/json"
)

// flexString decodes a JSON string, number, or null into a Go string. TCOP
// documents that field types such as durationTime and alarmStatus vary between
// metrics (string for most, number for some), and a real CVM payload arrived
// with durationTime as a number.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		*f = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(b) // number/bool literal, kept as its text form
	return nil
}

type Payload struct {
	SessionID       string          `json:"sessionId"`
	AlarmStatus     flexString      `json:"alarmStatus"` // "1" = firing, "0" = resolved
	AlarmType       string          `json:"alarmType"`   // "metric" | "event"
	DurationTime    flexString      `json:"durationTime"`
	RecoverTime     flexString      `json:"recoverTime"` // "0" while unresolved
	AlarmObjInfo    AlarmObjInfo    `json:"alarmObjInfo"`
	AlarmPolicyInfo AlarmPolicyInfo `json:"alarmPolicyInfo"`
}

type AlarmObjInfo struct {
	Region     string                `json:"region"`
	Namespace  string                `json:"namespace"` // e.g. "qce/cvm"
	AppID      flexString            `json:"appId"`
	Uin        flexString            `json:"uin"`
	Dimensions map[string]flexString `json:"dimensions"`
}

type AlarmPolicyInfo struct {
	PolicyID   string     `json:"policyId"`
	PolicyType string     `json:"policyType"`
	PolicyName string     `json:"policyName"`
	Conditions Conditions `json:"conditions"`
}

type Conditions struct {
	MetricName     string     `json:"metricName"`
	MetricShowName string     `json:"metricShowName"`
	CalcType       string     `json:"calcType"`
	CalcValue      flexString `json:"calcValue"`
	CalcUnit       string     `json:"calcUnit"`
	CurrentValue   flexString `json:"currentValue"`
	HistoryValue   flexString `json:"historyValue"`
}
