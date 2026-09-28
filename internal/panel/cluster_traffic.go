package panel

import (
	"context"
	"sync"
	"time"

	"github.com/kejilion/kejilion-panel/internal/cluster"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/notification"
	"github.com/kejilion/kejilion-panel/internal/store"
)

type clusterTrafficSource struct {
	raw      notification.HostSource
	store    *store.Store
	location func(context.Context) *time.Location
	now      func() time.Time
	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
}

func (s *clusterTrafficSource) Hosts(ctx context.Context) cluster.HostList {
	return s.account(ctx, s.raw.Hosts(ctx))
}

func (s *clusterTrafficSource) account(ctx context.Context, hosts cluster.HostList) cluster.HostList {
	configured := false
	for _, details := range s.store.ClusterHostDetails() {
		configured = configured || details.TrafficResetDay > 0
	}
	if !configured {
		return hosts
	}
	ids := make([]string, 0, len(hosts.Items))
	samples := make(map[string]store.ClusterTrafficSample)
	for _, host := range hosts.Items {
		ids = append(ids, host.ID)
		if snapshot := host.LastSnapshot; snapshot != nil {
			samples[host.ID] = store.ClusterTrafficSample{ReceivedAt: snapshot.ReceivedAt, SourceKey: host.RemoteNodeID + "\x00" + host.PeerFingerprint,
				CollectedAt: snapshot.Telemetry.CollectedAt, Received: snapshot.Telemetry.Network.ReceivedBytes,
				Sent: snapshot.Telemetry.Network.SentBytes, Uptime: snapshot.Telemetry.UptimeSeconds}
		}
	}
	location := s.location(ctx)
	periods := make(map[string]contract.TrafficPeriod)
	if location == nil {
		for id, details := range s.store.ClusterHostDetails() {
			if details.TrafficResetDay > 0 {
				periods[id] = contract.TrafficPeriod{}
			}
		}
	} else {
		// Write failures return unavailable periods without consuming the cursors.
		periods, _ = s.store.UpdateClusterTraffic(samples, ids, s.now(), location)
	}
	// Copy the inventory, leaving raw snapshots and real-time rates untouched.
	hosts.Items = append([]cluster.Host(nil), hosts.Items...)
	for i := range hosts.Items {
		id := hosts.Items[i].ID
		period, configured := periods[id]
		if !configured {
			continue
		}
		hosts.Items[i].TrafficPeriod = &period
	}
	return hosts
}

// Sampling does not depend on an open browser or a working notification channel.
func (s *clusterTrafficSource) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done != nil {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			if len(s.store.ClusterHostDetails()) > 0 {
				sampleCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
				s.Hosts(sampleCtx)
				cancel()
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *clusterTrafficSource) Close() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func (s *Server) accountedClusterHosts(ctx context.Context) cluster.HostList {
	if s.clusterTraffic != nil {
		return s.clusterTraffic.account(ctx, s.cluster.Hosts(ctx))
	}
	return s.cluster.Hosts(ctx)
}
