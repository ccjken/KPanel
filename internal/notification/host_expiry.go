package notification

import (
	"fmt"
	"time"

	"github.com/kejilion/kejilion-panel/internal/cluster"
)

const serverExpiryRuleKey = "server-expiry"

// HostExpiry comes from the center's saved host metadata, never peer telemetry.
type HostExpiry struct {
	ExpiresOn string
	Enabled   bool // Legacy opt-in, used only to initialize the global rule.
}

func validExpiryDate(value string) bool {
	date, err := time.Parse("2006-01-02", value)
	return err == nil && date.Year() > 0 && date.Format("2006-01-02") == value
}

func (s *Service) expirySnapshot() map[string]HostExpiry {
	if s.hostExpiries == nil {
		return nil
	}
	return s.hostExpiries()
}

func (s *Service) handleHostExpiry(host cluster.Host, details HostExpiry, now time.Time, locale string, record func(cluster.Host, string, string, string) (bool, bool)) bool {
	if !validExpiryDate(details.ExpiresOn) {
		return false
	}
	expiry, _ := time.Parse("2006-01-02", details.ExpiresOn)
	// Compare calendar dates in the center timezone; local days may be 23/25h.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	days := int((expiry.Unix() - today.Unix()) / 86400)
	var bit uint8
	for index, milestone := range []int{7, 3, 1, 0} {
		if days == milestone {
			bit = 1 << index
			break
		}
	}
	key := host.ID + ":" + serverExpiryRuleKey
	state, tracked := s.reserveAlertState(key)
	if !tracked {
		return false
	}
	changed := state.ExpiryDate != details.ExpiresOn
	if changed {
		state = alertState{ExpiryDate: details.ExpiresOn}
	}
	// Missed calendar days are not backfilled. Keep every delivered milestone
	// for this date so restarts, toggles and backward clock changes do not repeat it.
	if bit != 0 && state.ExpiryNotifiedMask&bit == 0 {
		if recorded, _ := record(host, serverExpiryRuleKey, "info", hostExpiryMessage(host, details.ExpiresOn, days, now, locale)); recorded {
			state.ExpiryNotifiedMask |= bit
			changed = true
		}
	}
	if changed {
		s.setAlertState(key, state)
	}
	return changed
}

func hostExpiryMessage(host cluster.Host, date string, days int, now time.Time, locale string) string {
	name := safeMessageText(host.Name)
	switch locale {
	case "en-US":
		return fmt.Sprintf("⏰ [KPanel Cluster Notice]\n\nHost: %s\nExpiry date: %s\nDays remaining: %d\n\nTime: %s", name, date, days, formatNotificationTime(now))
	case "zh-TW":
		return fmt.Sprintf("⏰ [KPanel 叢集通知]\n\n主機：%s\n到期日期：%s\n剩餘天數：%d\n\n時間：%s", name, date, days, formatNotificationTime(now))
	default:
		return fmt.Sprintf("⏰ [KPanel 集群通知]\n\n主机：%s\n到期日期：%s\n剩余天数：%d\n\n时间：%s", name, date, days, formatNotificationTime(now))
	}
}
