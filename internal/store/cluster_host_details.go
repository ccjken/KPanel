package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

// ClusterHostDetails are optional, center-owned server metadata, not telemetry.
type ClusterHostDetails struct {
	ExpiresOn                        string `json:"expiresOn,omitempty"`
	ExpiryReminderEnabled            bool   `json:"expiryReminderEnabled,omitempty"` // Legacy; migrated to the global notification rule.
	Price                            string `json:"price,omitempty"`
	TrafficResetDay                  int    `json:"trafficResetDay,omitempty"`
	TrafficMonthlyQuotaGiB           int    `json:"trafficMonthlyQuotaGiB,omitempty"`
	TrafficCalculation               string `json:"trafficCalculation,omitempty"`
	TrafficTotalReceivedThresholdGiB int    `json:"trafficTotalReceivedThresholdGiB,omitempty"`
	TrafficTotalSentThresholdGiB     int    `json:"trafficTotalSentThresholdGiB,omitempty"`
}

func ValidateClusterHostDetails(value ClusterHostDetails) error {
	if value.TrafficMonthlyQuotaGiB < 0 || value.TrafficMonthlyQuotaGiB > contract.MaxTrafficThresholdGiB {
		return ErrInvalidRecord
	}
	switch value.TrafficCalculation {
	case "", "total", "received", "sent", "max":
	default:
		return ErrInvalidRecord
	}
	if value.TrafficTotalReceivedThresholdGiB < 0 || value.TrafficTotalReceivedThresholdGiB > contract.MaxTrafficThresholdGiB ||
		value.TrafficTotalSentThresholdGiB < 0 || value.TrafficTotalSentThresholdGiB > contract.MaxTrafficThresholdGiB {
		return ErrInvalidRecord
	}
	if value.ExpiryReminderEnabled && value.ExpiresOn == "" {
		return ErrInvalidRecord
	}
	if value.TrafficResetDay < 0 || value.TrafficResetDay > 31 ||
		!utf8.ValidString(value.Price) || utf8.RuneCountInString(value.Price) > 40 ||
		value.Price != strings.TrimSpace(value.Price) {
		return ErrInvalidRecord
	}
	for _, char := range value.Price {
		if unicode.IsControl(char) {
			return ErrInvalidRecord
		}
	}
	if value.ExpiresOn != "" {
		date, err := time.Parse("2006-01-02", value.ExpiresOn)
		if err != nil || date.Year() < 1 || date.Format("2006-01-02") != value.ExpiresOn {
			return ErrInvalidRecord
		}
	}
	return nil
}

func validateClusterHostDetailsMap(values map[string]ClusterHostDetails) error {
	ids := make([]string, 0, len(values))
	for id, value := range values {
		ids = append(ids, id)
		if err := ValidateClusterHostDetails(value); err != nil {
			return err
		}
	}
	return ValidateClusterHostOrder(ids)
}

func (s *Store) ClusterHostDetails() map[string]ClusterHostDetails {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.data.ClusterHostDetails)
}

func ClusterHostDetailsResourceVersion(id string, value ClusterHostDetails) string {
	data, _ := json.Marshal(value)
	digest := sha256.Sum256(append([]byte(id+"\x00"), data...))
	return fmt.Sprintf("sha256:%x", digest[:])
}

// Reconcile against the live inventory on each write to bound stale host data.
func (s *Store) ReplaceClusterHostDetails(id, expected string, value ClusterHostDetails, activeIDs []string) error {
	if ValidateClusterHostOrder(activeIDs) != nil || !slices.Contains(activeIDs, id) || ValidateClusterHostDetails(value) != nil {
		return ErrInvalidRecord
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if expected != ClusterHostDetailsResourceVersion(id, s.data.ClusterHostDetails[id]) {
		return ErrConflict
	}
	previous := cloneDiskState(s.data)
	next := make(map[string]ClusterHostDetails, len(activeIDs))
	for _, activeID := range activeIDs {
		if current := s.data.ClusterHostDetails[activeID]; current != (ClusterHostDetails{}) {
			next[activeID] = current
		}
	}
	if value == (ClusterHostDetails{}) {
		delete(next, id)
	} else {
		next[id] = value
	}
	s.data.ClusterHostDetails = next
	reconcileClusterTraffic(s.data.ClusterTraffic, next, activeIDs)
	if err := s.persistLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}
