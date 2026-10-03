package tcop

type Payload struct {
	SessionID       string          `json:"sessionId"`
	AlarmStatus     string          `json:"alarmStatus"` // "1" = firing, "0" = resolved
	AlarmType       string          `json:"alarmType"`   // "metric" | "event"
	AlarmObjInfo    AlarmObjInfo    `json:"alarmObjInfo"`
	AlarmPolicyInfo AlarmPolicyInfo `json:"alarmPolicyInfo"`
}

type AlarmObjInfo struct {
	Region     string            `json:"region"`
	Namespace  string            `json:"namespace"` // e.g. "qce/cvm"
	AppID      string            `json:"appId"`
	Uin        string            `json:"uin"`
	Dimensions map[string]string `json:"dimensions"`
}

type AlarmPolicyInfo struct {
	PolicyID   string     `json:"policyId"`
	PolicyType string     `json:"policyType"`
	PolicyName string     `json:"policyName"`
	Conditions Conditions `json:"conditions"`
}

type Conditions struct {
	MetricName     string `json:"metricName"`
	MetricShowName string `json:"metricShowName"`
	CalcType       string `json:"calcType"`
	CalcValue      string `json:"calcValue"`
	CalcUnit       string `json:"calcUnit"`
	CurrentValue   string `json:"currentValue"`
	HistoryValue   string `json:"historyValue"`
}
