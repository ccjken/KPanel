package contract

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxServiceChecks            = 16
	MaxServiceCheckSummaryBytes = 16 << 10
)

var serviceCheckIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)
var serviceCheckVersionPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// ServiceCheckSummary is a bounded observation, never a probe instruction.
// Epoch changes on sampler restart; Sequence advances only on real samples.
type ServiceCheckSummary struct {
	Epoch           string               `json:"epoch"`
	Sequence        uint64               `json:"sequence"`
	IntervalSeconds int                  `json:"intervalSeconds"`
	Available       bool                 `json:"available"`
	Items           []ServiceCheckStatus `json:"items"`
}

type ServiceCheckStatus struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	// Target contains only the host/port, never URL paths, queries or credentials.
	Target       string     `json:"target"`
	State        string     `json:"state"`
	CheckedAt    time.Time  `json:"checkedAt"`
	Failures     int        `json:"failures"`
	Successes    int        `json:"successes"`
	FailureSince *time.Time `json:"failureSince,omitempty"`
	ErrorCode    string     `json:"errorCode,omitempty"`
}

func ValidServiceCheckSummary(value *ServiceCheckSummary, now time.Time) bool {
	if value == nil || !serviceCheckVersionPattern.MatchString(value.Epoch) || value.Sequence > 1<<53-1 ||
		value.IntervalSeconds < 1 || value.IntervalSeconds > 3600 || len(value.Items) > MaxServiceChecks {
		return false
	}
	seen := make(map[string]bool, len(value.Items))
	for _, item := range value.Items {
		if !serviceCheckIDPattern.MatchString(item.ID) || seen[item.ID] || !serviceCheckVersionPattern.MatchString(item.Revision) ||
			!serviceCheckText(item.Name, 192) || !serviceCheckText(item.Target, 253) ||
			strings.ContainsAny(item.Target, "/?#@\\\"<>&") ||
			(item.Kind != "ping" && item.Kind != "tcp" && item.Kind != "http") ||
			(item.State != "up" && item.State != "down" && item.State != "unknown") ||
			item.Failures < 0 || item.Failures > 3 || item.Successes < 0 || item.Successes > 2 {
			return false
		}
		seen[item.ID] = true
		if item.CheckedAt.After(now.Add(5*time.Minute)) || (item.State != "unknown" && item.CheckedAt.IsZero()) ||
			(item.State == "up" && (item.Failures != 0 || item.Successes == 0 || item.FailureSince != nil || item.ErrorCode != "")) ||
			(item.State == "down" && (item.Successes != 0 || item.Failures == 0 || item.FailureSince == nil)) ||
			(item.State == "unknown" && (item.Successes != 0 || item.Failures != 0 || item.FailureSince != nil)) {
			return false
		}
		if item.FailureSince != nil && (item.FailureSince.IsZero() || item.FailureSince.After(item.CheckedAt)) {
			return false
		}
		switch item.ErrorCode {
		case "", "timeout", "dns", "tls", "connection", "http_status", "probe_failed", "cancelled":
		default:
			return false
		}
	}
	encoded, err := json.Marshal(value)
	return err == nil && len(encoded) <= MaxServiceCheckSummaryBytes
}

func CloneServiceCheckSummary(value *ServiceCheckSummary) *ServiceCheckSummary {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Items = append([]ServiceCheckStatus{}, value.Items...)
	for i := range copy.Items {
		if at := copy.Items[i].FailureSince; at != nil {
			t := *at
			copy.Items[i].FailureSince = &t
		}
	}
	return &copy
}

func serviceCheckText(value string, max int) bool {
	if value == "" || len(value) > max || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
