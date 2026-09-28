package notification

import (
	"github.com/kejilion/kejilion-panel/internal/cluster"
	"time"
)

func trafficCycleKey(host cluster.Host) string {
	if host.TrafficPeriod == nil {
		return ""
	}
	return host.TrafficPeriod.StartedAt.Format(time.RFC3339Nano)
}

func validTrafficCycle(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 35 {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}
