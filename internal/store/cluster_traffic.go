package store

import (
	"maps"
	"math/bits"
	"slices"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

const maxTrafficBytes = uint64(1<<53 - 1)
const trafficSamplingInterval = 30 * time.Second
const trafficBoundaryWindow = 90 * time.Second

// Only the current cycle and one raw cursor per host are retained (at most 101).
type ClusterTrafficRecord struct {
	SourceKey   string                 `json:"sourceKey,omitempty"`
	Period      contract.TrafficPeriod `json:"period"`
	ResetDay    int                    `json:"resetDay"`
	Timezone    string                 `json:"timezone"`
	SampleAt    time.Time              `json:"sampleAt"`
	CollectedAt time.Time              `json:"collectedAt"`
	RecordedAt  time.Time              `json:"recordedAt"`
	Received    uint64                 `json:"received"`
	Sent        uint64                 `json:"sent"`
	Uptime      uint64                 `json:"uptime"`
}

type ClusterTrafficSample struct {
	SourceKey   string
	ReceivedAt  time.Time
	CollectedAt time.Time
	Received    uint64
	Sent        uint64
	Uptime      uint64
}

func trafficCycle(now time.Time, day int, location *time.Location) (time.Time, time.Time) {
	now = now.In(location)
	boundary := func(month int) time.Time {
		first := time.Date(now.Year(), now.Month()+time.Month(month), 1, 0, 0, 0, 0, location)
		lastDay := first.AddDate(0, 1, -1).Day()
		return time.Date(first.Year(), first.Month(), min(day, lastDay), 0, 0, 0, 0, location).UTC()
	}
	start := boundary(0)
	if now.Before(start) {
		return boundary(-1), start
	}
	return start, boundary(1)
}

func validateClusterTraffic(values map[string]ClusterTrafficRecord, details map[string]ClusterHostDetails) error {
	ids := make([]string, 0, len(values))
	for id, value := range values {
		ids = append(ids, id)
		p := value.Period
		if value.ResetDay < 1 || value.ResetDay > 31 || details[id].TrafficResetDay != value.ResetDay ||
			len(value.SourceKey) > 512 || len(value.Timezone) > 128 || value.Timezone == "" || p.StartedAt.IsZero() || !p.EndsAt.After(p.StartedAt) ||
			p.EndsAt.Sub(p.StartedAt) > 32*24*time.Hour || value.Received > maxTrafficBytes || value.Sent > maxTrafficBytes ||
			p.ReceivedBytes > maxTrafficBytes || p.SentBytes > maxTrafficBytes || value.Uptime > maxTrafficBytes ||
			(p.Available && (value.SampleAt.IsZero() || value.CollectedAt.IsZero())) {
			return ErrInvalidRecord
		}
		location, err := time.LoadLocation(value.Timezone)
		if err != nil {
			return ErrInvalidRecord
		}
		start, end := trafficCycle(p.StartedAt, value.ResetDay, location)
		if !start.Equal(p.StartedAt) || !end.Equal(p.EndsAt) {
			return ErrInvalidRecord
		}
	}
	return ValidateClusterHostOrder(ids)
}

// UpdateClusterTraffic atomically persists derived counters and raw cursors.
// A failed write rolls both back, allowing the next sample to retry the delta.
func (s *Store) UpdateClusterTraffic(samples map[string]ClusterTrafficSample, activeIDs []string, now time.Time, location *time.Location) (map[string]contract.TrafficPeriod, error) {
	if ValidateClusterHostOrder(activeIDs) != nil || location == nil {
		return nil, ErrInvalidRecord
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[string]ClusterTrafficRecord)
	result := make(map[string]contract.TrafficPeriod)
	for _, id := range activeIDs {
		day := s.data.ClusterHostDetails[id].TrafficResetDay
		if day == 0 {
			continue
		}
		record := s.data.ClusterTraffic[id]
		sample, ok := samples[id]
		fresh := ok && len(sample.SourceKey) <= 512 && !sample.ReceivedAt.IsZero() && !sample.CollectedAt.IsZero() &&
			!sample.ReceivedAt.After(now.Add(5*time.Second)) && now.Sub(sample.ReceivedAt) <= trafficBoundaryWindow &&
			sample.Received <= maxTrafficBytes && sample.Sent <= maxTrafficBytes && sample.Uptime <= maxTrafficBytes
		start, end := trafficCycle(now, day, location)
		if record.ResetDay != day || record.Timezone != location.String() || (fresh && record.SourceKey != sample.SourceKey) {
			record = ClusterTrafficRecord{ResetDay: day, Timezone: location.String()}
		}
		// Do not reopen an older cycle if the center clock moves backwards.
		if start.Before(record.Period.StartedAt) {
			next[id] = record
			result[id] = contract.TrafficPeriod{StartedAt: start, EndsAt: end, Partial: true}
			continue
		}
		rolled := !start.Equal(record.Period.StartedAt)
		if rolled {
			record.Period = contract.TrafficPeriod{StartedAt: start, EndsAt: end}
		}
		if fresh && sample.CollectedAt.After(record.CollectedAt) && sample.ReceivedAt.After(record.SampleAt) &&
			(rolled || record.RecordedAt.IsZero() || now.Sub(record.RecordedAt) >= trafficSamplingInterval) {
			advanceClusterTraffic(&record, sample)
			record.RecordedAt = now.UTC()
		}
		next[id] = record
		result[id] = record.Period
	}
	if maps.Equal(next, s.data.ClusterTraffic) {
		return result, nil
	}
	previous := s.data.ClusterTraffic
	s.data.ClusterTraffic = next
	if err := s.persistLocked(); err != nil {
		s.data.ClusterTraffic = previous
		for id, period := range result {
			period.Available = false
			result[id] = period
		}
		return result, err
	}
	return result, nil
}

func advanceClusterTraffic(record *ClusterTrafficRecord, sample ClusterTrafficSample) {
	p := &record.Period
	defer func() {
		record.SourceKey = sample.SourceKey
		record.SampleAt, record.CollectedAt = sample.ReceivedAt.UTC(), sample.CollectedAt.UTC()
		record.Received, record.Sent, record.Uptime = sample.Received, sample.Sent, sample.Uptime
	}()
	// An old cached sample must not initialize the next period.
	if sample.ReceivedAt.Before(p.StartedAt) {
		return
	}
	p.Available = true
	if record.SampleAt.IsZero() {
		p.Partial = true // The first sample is a baseline, never historical usage.
		return
	}
	elapsed := sample.CollectedAt.Sub(record.CollectedAt).Seconds()
	rebooted := sample.Uptime < record.Uptime || (elapsed > 0 && float64(sample.Uptime)+5 < float64(record.Uptime)+elapsed)
	delta := func(current, previous uint64) uint64 {
		if current < previous {
			p.Partial = true // Interface counters rolled back; their history is unknown.
			return 0
		}
		return current - previous
	}
	received, sent := uint64(0), uint64(0)
	if rebooted {
		p.Partial = true
		if sample.Uptime <= uint64((32*24*time.Hour)/time.Second) && !sample.ReceivedAt.Add(-time.Duration(sample.Uptime)*time.Second).Before(p.StartedAt) {
			received, sent = sample.Received, sample.Sent
		}
	} else if record.SampleAt.Before(p.StartedAt) {
		gap := sample.ReceivedAt.Sub(record.SampleAt)
		if gap <= trafficBoundaryWindow {
			// The single interval straddling midnight is prorated, not charged in full.
			proportion := func(value uint64) uint64 {
				hi, lo := bits.Mul64(value, uint64(sample.ReceivedAt.Sub(p.StartedAt)))
				quotient, _ := bits.Div64(hi, lo, uint64(gap))
				return quotient
			}
			received, sent = proportion(delta(sample.Received, record.Received)), proportion(delta(sample.Sent, record.Sent))
			p.Estimated = true
		} else {
			p.Partial = true
		}
	} else {
		received, sent = delta(sample.Received, record.Received), delta(sample.Sent, record.Sent)
	}
	add := func(total, increment uint64) uint64 {
		if increment > maxTrafficBytes-total {
			p.Partial = true
			return maxTrafficBytes
		}
		return total + increment
	}
	p.ReceivedBytes, p.SentBytes = add(p.ReceivedBytes, received), add(p.SentBytes, sent)
}

func reconcileClusterTraffic(values map[string]ClusterTrafficRecord, details map[string]ClusterHostDetails, activeIDs []string) {
	for id, record := range values {
		if !slices.Contains(activeIDs, id) || details[id].TrafficResetDay != record.ResetDay {
			delete(values, id)
		}
	}
}
