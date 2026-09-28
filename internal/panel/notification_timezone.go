package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const notificationTimezoneRefreshInterval = 5 * time.Minute

// notificationTimezoneSource reads the host-configured IANA timezone from
// Agent and caches it briefly. The Panel container itself may intentionally run
// in UTC, so time.Local is not a reliable representation of the host timezone.
type notificationTimezoneSource struct {
	agent agentAPI

	mu        sync.Mutex
	location  *time.Location
	known     bool
	checkedAt time.Time
}

func newNotificationTimezoneSource(agent agentAPI) *notificationTimezoneSource {
	return &notificationTimezoneSource{agent: agent}
}

func (s *notificationTimezoneSource) Location(ctx context.Context) *time.Location {
	fallback := time.Local
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	refreshInterval := notificationTimezoneRefreshInterval
	if !s.known {
		refreshInterval = 30 * time.Second
	}
	if s.location != nil && now.Sub(s.checkedAt) < refreshInterval {
		return s.location
	}
	s.checkedAt = now

	location := fallback
	if s.known {
		location = s.location
	}
	if s.agent != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		response, err := s.agent.Get(lookupCtx, "/v1/system/summary", "", newRequestID())
		cancel()
		if err == nil && response.StatusCode == http.StatusOK {
			var summary contract.SystemSummary
			if json.Unmarshal(response.Body, &summary) == nil {
				if value := strings.TrimSpace(summary.Management.Timezone); value != "" {
					if loaded, loadErr := time.LoadLocation(value); loadErr == nil {
						location = loaded
						s.known = true
					}
				}
			}
		}
	}

	s.location = location
	return location
}

// Accounting cannot guess a cycle timezone on startup. Once known, transient
// Agent failures retain it; an explicit successful timezone change replaces it.
func (s *notificationTimezoneSource) AccountingLocation(ctx context.Context) *time.Location {
	s.Location(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.known {
		return nil
	}
	return s.location
}
