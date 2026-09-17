package service

import (
	"crypto/sha256"
	"encoding/hex"
)

// Output stable fingerprint  from alarm identity field.
func ComputeFingerprint(account, resourceID, metricType string) string {
	raw := account + "|" + resourceID + "|" + metricType
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
