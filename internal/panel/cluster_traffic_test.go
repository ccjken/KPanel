package panel

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/cluster"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/store"
)

type trafficTestHosts struct {
	host   cluster.Host
	called chan struct{}
}

func (s trafficTestHosts) Hosts(context.Context) cluster.HostList {
	if s.called != nil {
		select {
		case s.called <- struct{}{}:
		default:
		}
	}
	return cluster.HostList{Items: []cluster.Host{s.host}}
}

func TestClusterTrafficSourceBackgroundAndRawIsolation(t *testing.T) {
	storage, err := store.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	now := time.Now().UTC()
	raw := trafficTestHosts{host: cluster.Host{ID: "local", LastSnapshot: &cluster.HostSnapshot{
		ReceivedAt: now, Telemetry: contract.HostTelemetry{CollectedAt: now, UptimeSeconds: 1000, Network: contract.NetworkSummary{ReceivedBytes: 1000, SentBytes: 2000}},
		ReceiveBytesPerSecond: 42, TransmitBytesPerSecond: 21,
	}}, called: make(chan struct{}, 1)}
	source := &clusterTrafficSource{raw: raw, store: storage, now: func() time.Time { return now }, location: func(context.Context) *time.Location { return time.UTC }}
	legacy := source.Hosts(context.Background()).Items[0]
	if legacy.TrafficPeriod != nil || legacy.LastSnapshot.Telemetry.Network.ReceivedBytes != 1000 {
		t.Fatal("legacy changed")
	}
	if err := storage.ReplaceClusterHostDetails("local", store.ClusterHostDetailsResourceVersion("local", store.ClusterHostDetails{}), store.ClusterHostDetails{TrafficResetDay: 1}, []string{"local"}); err != nil {
		t.Fatal(err)
	}
	<-raw.called
	source.Start(context.Background())
	select {
	case <-raw.called:
	case <-time.After(5 * time.Second):
		t.Fatal("background sampler did not start")
	}
	source.Close()
	first := source.Hosts(context.Background()).Items[0]
	if first.TrafficPeriod == nil || !first.TrafficPeriod.Available || first.TrafficPeriod.ReceivedBytes != 0 {
		t.Fatalf("baseline: %+v", first.TrafficPeriod)
	}
	source.location = func(context.Context) *time.Location { return nil }
	unavailable := source.Hosts(context.Background()).Items[0]
	if unavailable.TrafficPeriod == nil || unavailable.TrafficPeriod.Available || unavailable.LastSnapshot.Telemetry.Network.ReceivedBytes != 1000 {
		t.Fatal("unknown timezone fell back to raw totals")
	}
	source.location = func(context.Context) *time.Location { return time.UTC }
	now = now.Add(30 * time.Second)
	raw.host.LastSnapshot.Telemetry.CollectedAt = now
	raw.host.LastSnapshot.ReceivedAt = now
	raw.host.LastSnapshot.Telemetry.UptimeSeconds += 30
	raw.host.LastSnapshot.Telemetry.Network.ReceivedBytes += 500
	source.raw = raw
	next := source.Hosts(context.Background()).Items[0]
	if next.TrafficPeriod.ReceivedBytes != 500 || next.LastSnapshot.Telemetry.Network.ReceivedBytes != 1500 || next.LastSnapshot.ReceiveBytesPerSecond != 42 || raw.host.TrafficPeriod != nil {
		t.Fatalf("accounting leaked into telemetry: %+v", next)
	}
}
