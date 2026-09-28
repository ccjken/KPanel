package monitoring

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func checkEpoch() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic("monitoring random source unavailable")
	}
	return hex.EncodeToString(value[:])
}

func checkRevision(item Check) string {
	digest := sha256.Sum256([]byte(item.Kind + "\x00" + item.Target))
	return hex.EncodeToString(digest[:16])
}

func checkStatusTarget(item Check) string {
	target := item.Target
	if item.Kind == "http" {
		if parsed, err := url.Parse(target); err == nil {
			target = parsed.Host
		}
	}
	if len(target) > 253 || strings.ContainsAny(target, "/?#@\\\"<>&") {
		return "configured-target"
	}
	return target
}

// CheckStatus reads memory only. History persistence is deliberately independent.
func (s *Service) CheckStatus() contract.ServiceCheckSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value := contract.ServiceCheckSummary{Epoch: s.checkEpoch, Sequence: s.checkSequence,
		IntervalSeconds: max(1, int(s.operatorLatencyInterval.Seconds())),
		Available:       s.checks.available && s.operatorLatency != nil, Items: []contract.ServiceCheckStatus{}}
	for _, item := range s.checks.state.Items {
		status := contract.ServiceCheckStatus{ID: item.ID, Revision: checkRevision(item), Name: item.Name,
			Kind: item.Kind, Target: checkStatusTarget(item), State: "unknown"}
		if current, ok := s.checkStatuses[item.ID]; ok && current.Revision == status.Revision {
			status = current
			status.Name = item.Name
		}
		value.Items = append(value.Items, status)
	}
	return *contract.CloneServiceCheckSummary(&value)
}

func (s *Service) recordCheckStatus(items []operatorLatencyResult, at time.Time, generation uint64) {
	s.mu.Lock()
	if generation != s.checkGeneration {
		s.mu.Unlock()
		return
	}
	s.checkSequence++
	for _, result := range items {
		item := result.target
		// A completed sample for a replaced/deleted target must not resurrect it.
		current := false
		for _, configured := range s.checks.state.Items {
			if configured.ID == item.ID && checkRevision(configured) == checkRevision(item) {
				current = true
				break
			}
		}
		if !current {
			continue
		}
		previous := s.checkStatuses[item.ID]
		if !previous.CheckedAt.IsZero() && !at.After(previous.CheckedAt) {
			continue
		}
		if previous.Revision != checkRevision(item) || at.Sub(previous.CheckedAt) > 2*s.operatorLatencyInterval+time.Minute {
			previous = contract.ServiceCheckStatus{}
		}
		status := contract.ServiceCheckStatus{ID: item.ID, Revision: checkRevision(item), Name: item.Name,
			Kind: item.Kind, Target: checkStatusTarget(item), CheckedAt: at, State: "down", ErrorCode: result.errorCode}
		if result.unknown {
			status.State = "unknown"
		} else if result.reachable {
			status.State = "up"
			status.Successes = min(2, previous.Successes+1)
			status.ErrorCode = ""
		} else {
			status.Failures = min(3, previous.Failures+1)
			status.FailureSince = previous.FailureSince
			if status.FailureSince == nil {
				started := at
				status.FailureSince = &started
			}
		}
		s.checkStatuses[item.ID] = status
	}
	for id := range s.checkStatuses {
		found := false
		for _, item := range s.checks.state.Items {
			if item.ID == id {
				found = true
				break
			}
		}
		if !found {
			delete(s.checkStatuses, id)
		}
	}
	s.mu.Unlock()
	if s.onCheckStatus != nil {
		s.onCheckStatus(s.CheckStatus())
	}
}
