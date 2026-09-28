package contract

import "time"

// MaxTrafficThresholdGiB bounds cumulative traffic notification settings.
const MaxTrafficThresholdGiB = 1_048_576

// TrafficPeriod is center-owned accounting, separate from the agent's raw counters.
type TrafficPeriod struct {
	ID            string    `json:"id"`
	ReceivedBytes uint64    `json:"receivedBytes"`
	SentBytes     uint64    `json:"sentBytes"`
	Available     bool      `json:"available"`
	StartedAt     time.Time `json:"startedAt"`
	EndsAt        time.Time `json:"endsAt"`
	Partial       bool      `json:"partial"`
	Estimated     bool      `json:"estimated"`
}
