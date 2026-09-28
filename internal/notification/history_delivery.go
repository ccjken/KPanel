package notification

import (
	"context"
	"reflect"
	"time"

	"github.com/kejilion/kejilion-panel/internal/cluster"
)

func ruleEnabled(r Rules, rule string) bool {
	switch rule {
	case "cpu":
		return r.CPUEnabled
	case "memory":
		return r.MemoryEnabled
	case "disk":
		return r.DiskEnabled
	case "traffic":
		return r.TrafficEnabled
	case cumulativeTrafficReceivedRuleKey:
		return r.TrafficTotalReceivedEnabled
	case cumulativeTrafficSentRuleKey:
		return r.TrafficTotalSentEnabled
	case "availability":
		return r.HostOfflineEnabled
	case "ssh":
		return r.SSHLoginEnabled
	case serverExpiryRuleKey:
		return true // The per-host setting is checked against the live expiry below.
	}
	return false
}

func (s *Service) deliverHistory(ctx context.Context, state persistedState, hosts []cluster.Host, credential string, ready bool, now time.Time) error {
	history, err := s.history.snapshot()
	if err != nil {
		return historyError(err)
	}
	before := cloneHistory(history)
	expiries := s.expirySnapshot()
	present := map[string]bool{}
	for _, host := range hosts {
		present[host.ID] = true
	}
	indices := []int{}
	for i := range history.Events {
		event := &history.Events[i]
		if event.Delivery != "pending" && event.Delivery != "failed" {
			continue
		}
		reason := ""
		switch {
		case !state.Settings.Enabled:
			reason = "push_disabled"
		case event.Rule == serverExpiryRuleKey && (!expiries[event.HostID].Enabled || expiries[event.HostID].ExpiresOn != event.ExpiryDate):
			reason = "expiry_reminder_changed"
		case !ruleEnabled(state.Settings.Rules, event.Rule):
			reason = "rule_disabled"
		case !present[event.HostID]:
			reason = "host_removed"
		case credential != "" && event.ChannelFingerprint != tokenFingerprint(credential):
			reason = "channel_changed"
		case now.Sub(event.CreatedAt) >= 24*time.Hour:
			reason = "retry_expired"
		}
		if reason != "" {
			event.Delivery = "cancelled"
			event.LastErrorCode = reason
			continue
		}
		if !ready || len(indices) >= maxMessagesPerEvaluation ||
			(event.LastAttemptAt != nil && now.Sub(*event.LastAttemptAt) < alertRetryInterval) {
			continue
		}
		// Persist the attempt before the network call. A crash cannot cause a tight
		// retry loop; as with any webhook, acknowledgement loss may cause redelivery.
		event.Attempts++
		event.LastAttemptAt = timePtr(now)
		indices = append(indices, i)
	}
	if !reflect.DeepEqual(before, history) {
		if err := s.history.commit(history); err != nil {
			return historyError(err)
		}
	}
	if len(indices) == 0 {
		return nil
	}
	// Capacity pruning can remove old pending events. Never send an event that
	// no longer has a persisted local record.
	selected := map[string]bool{}
	for _, i := range indices {
		selected[history.Events[i].ID] = true
	}
	history, err = s.history.snapshot()
	if err != nil {
		return historyError(err)
	}
	indices = indices[:0]
	for i := range history.Events {
		if selected[history.Events[i].ID] {
			indices = append(indices, i)
		}
	}
	channel := state.Telegram
	for _, i := range indices {
		event := &history.Events[i]
		sendCtx, cancel := context.WithTimeout(ctx, channelSendTimeout)
		err := s.sendChannel(sendCtx, event.Provider, credential, channel, event.Message)
		cancel()
		channel.LastCheckedAt = timePtr(now)
		if err != nil {
			event.Delivery = "failed"
			event.LastErrorCode = channelCode(event.Provider, err)
			channel.Status = TelegramError
			channel.LastErrorCode = event.LastErrorCode
		} else {
			event.Delivery = "sent"
			event.LastErrorCode = ""
			channel.Status = TelegramReady
			channel.LastSuccessAt = timePtr(now)
			channel.LastErrorCode = ""
		}
	}
	if err := s.history.commit(history); err != nil {
		return historyError(err)
	}
	state.Telegram = channel
	return s.store.commitState(state)
}
