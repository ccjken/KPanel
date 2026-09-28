package notification

import "strings"

// HostTrafficLimits are center-owned notification settings. Zero inherits the
// global rule; a positive value enables and overrides that direction only.
type HostTrafficLimits struct {
	ReceivedGiB int
	SentGiB     int
}

func (s *Service) trafficLimitsSnapshot() map[string]HostTrafficLimits {
	if s.hostTrafficLimits == nil {
		return nil
	}
	return s.hostTrafficLimits()
}

func effectiveTrafficRules(global Rules, limits HostTrafficLimits) Rules {
	if limits.ReceivedGiB > 0 && limits.ReceivedGiB <= MaxTrafficTotalThresholdGiB {
		global.TrafficTotalReceivedEnabled = true
		global.TrafficTotalReceivedThresholdGiB = limits.ReceivedGiB
	}
	if limits.SentGiB > 0 && limits.SentGiB <= MaxTrafficTotalThresholdGiB {
		global.TrafficTotalSentEnabled = true
		global.TrafficTotalSentThresholdGiB = limits.SentGiB
	}
	return global
}

func isCumulativeTrafficRule(rule string) bool {
	return rule == cumulativeTrafficReceivedRuleKey || rule == cumulativeTrafficSentRuleKey
}

func trafficThreshold(rules Rules, rule string) int {
	if rule == cumulativeTrafficReceivedRuleKey {
		return rules.TrafficTotalReceivedThresholdGiB
	}
	if rule == cumulativeTrafficSentRuleKey {
		return rules.TrafficTotalSentThresholdGiB
	}
	return 0
}

// Preserve deduplication when only an unrelated global/host rule changes. Old
// alert states belong to the last evaluated global rules, unlike older events.
func reconcileTrafficThresholds(history *historyState, global Rules, limits map[string]HostTrafficLimits) {
	for key, alert := range history.Alerts {
		for _, rule := range []string{cumulativeTrafficReceivedRuleKey, cumulativeTrafficSentRuleKey} {
			if !strings.HasSuffix(key, ":"+rule) {
				continue
			}
			if alert.TrafficThresholdGiB == 0 {
				alert.TrafficThresholdGiB = trafficThreshold(history.Rules, rule)
			}
			current := effectiveTrafficRules(global, limits[strings.TrimSuffix(key, ":"+rule)])
			if !ruleEnabled(current, rule) || alert.TrafficThresholdGiB != trafficThreshold(current, rule) {
				delete(history.Alerts, key)
			} else {
				history.Alerts[key] = alert
			}
		}
	}
	for i := range history.Events {
		event := &history.Events[i]
		if isCumulativeTrafficRule(event.Rule) && event.TrafficThresholdGiB == 0 &&
			(event.Delivery == "pending" || event.Delivery == "failed") {
			// A legacy event may predate history.Rules. Its original threshold is
			// unknown; do not relabel and retry it under the latest global value.
			event.Delivery = "cancelled"
			event.LastErrorCode = "traffic_threshold_unknown"
		}
	}
}
