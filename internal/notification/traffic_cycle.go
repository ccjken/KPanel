package notification

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/kejilion/kejilion-panel/internal/cluster"
	"time"
)

func trafficCycleKey(host cluster.Host) string {
	if host.TrafficPeriod == nil {
		return ""
	}
	p := host.TrafficPeriod
	digest := sha256.Sum256([]byte(p.ID + "\x00" + p.StartedAt.Format(time.RFC3339Nano) + "\x00" + p.EndsAt.Format(time.RFC3339Nano)))
	return hex.EncodeToString(digest[:])
}

func validTrafficCycle(value string) bool {
	if value == "" {
		return true
	}
	return validHexString(value, 64)
}
