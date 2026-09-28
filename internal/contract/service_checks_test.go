package contract

import (
	"strings"
	"testing"
	"time"
)

func TestServiceCheckSummaryRejectsUnsafeAndInconsistentObservations(t *testing.T) {
	now := time.Now().UTC()
	valid := ServiceCheckSummary{Epoch: strings.Repeat("a", 32), Sequence: 3, IntervalSeconds: 300, Available: true, Items: []ServiceCheckStatus{{ID: "health", Revision: strings.Repeat("b", 32), Name: "Health", Kind: "http", Target: "example.com", State: "down", CheckedAt: now, Failures: 3, FailureSince: &now, ErrorCode: "timeout"}}}
	if !ValidServiceCheckSummary(&valid, now) {
		t.Fatal("valid observation rejected")
	}
	cases := map[string]func(*ServiceCheckSummary){
		"secret query": func(s *ServiceCheckSummary) { s.Items[0].Target = "host?token=secret" },
		"duplicate":    func(s *ServiceCheckSummary) { s.Items = append(s.Items, s.Items[0]) },
		"future":       func(s *ServiceCheckSummary) { s.Items[0].CheckedAt = now.Add(10 * time.Minute) },
		"mixed counts": func(s *ServiceCheckSummary) { s.Items[0].Successes = 1 },
		"too many":     func(s *ServiceCheckSummary) { s.Items = make([]ServiceCheckStatus, 17) },
		"raw error":    func(s *ServiceCheckSummary) { s.Items[0].ErrorCode = "https://host/secret" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := CloneServiceCheckSummary(&valid)
			mutate(value)
			if ValidServiceCheckSummary(value, now) {
				t.Fatal("accepted malformed summary")
			}
		})
	}
}
