package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func trafficStore(t *testing.T, day int) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	setTrafficDay(t, s, day)
	return s
}

func setTrafficDay(t *testing.T, s *Store, day int) {
	t.Helper()
	old := s.ClusterHostDetails()["local"]
	next := old
	next.TrafficResetDay = day
	if err := s.ReplaceClusterHostDetails("local", ClusterHostDetailsResourceVersion("local", old), next, []string{"local"}); err != nil {
		t.Fatal(err)
	}
}

func sampleTraffic(t *testing.T, s *Store, at time.Time, received, sent, uptime uint64) contract.TrafficPeriod {
	t.Helper()
	values, err := s.UpdateClusterTraffic(map[string]ClusterTrafficSample{"local": {
		ReceivedAt: at, CollectedAt: at, Received: received, Sent: sent, Uptime: uptime,
	}}, []string{"local"}, at, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	return values["local"]
}

func TestClusterTrafficCycleBoundaries(t *testing.T) {
	for _, tc := range []struct {
		at, zone   string
		day        int
		start, end string
	}{
		{"2026-02-28T00:00:00Z", "UTC", 31, "2026-02-28T00:00:00Z", "2026-03-31T00:00:00Z"},
		{"2028-02-28T23:59:59Z", "UTC", 31, "2028-01-31T00:00:00Z", "2028-02-29T00:00:00Z"},
		{"2026-12-31T16:00:00Z", "Asia/Shanghai", 1, "2026-12-31T16:00:00Z", "2027-01-31T16:00:00Z"},
		{"2026-03-08T05:00:00Z", "America/New_York", 8, "2026-03-08T05:00:00Z", "2026-04-08T04:00:00Z"},
	} {
		t.Run(tc.at+tc.zone, func(t *testing.T) {
			at, _ := time.Parse(time.RFC3339, tc.at)
			zone, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			start, end := trafficCycle(at, tc.day, zone)
			if start.Format(time.RFC3339) != tc.start || end.Format(time.RFC3339) != tc.end {
				t.Fatalf("cycle: %s %s", start, end)
			}
		})
	}
}

func TestClusterTrafficPersistentDeltasAndConfiguration(t *testing.T) {
	s := trafficStore(t, 15)
	at := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	if got := sampleTraffic(t, s, at, 1000, 500, 10000); !got.Available || !got.Partial || got.ReceivedBytes != 0 {
		t.Fatalf("baseline: %+v", got)
	}
	if got := sampleTraffic(t, s, at.Add(30*time.Second), 1600, 700, 10030); got.ReceivedBytes != 600 || got.SentBytes != 200 {
		t.Fatalf("delta: %+v", got)
	}
	// Duplicate, stale and too-frequent samples never add traffic or advance the cursor.
	sampleTraffic(t, s, at.Add(30*time.Second), 1900, 900, 10030)
	sampleTraffic(t, s, at, 9999, 9999, 10000)
	sampleTraffic(t, s, at.Add(31*time.Second), 1700, 750, 10031)
	path := s.path
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got := sampleTraffic(t, reopened, at.Add(60*time.Second), 2000, 800, 10060); got.ReceivedBytes != 1000 || got.SentBytes != 300 {
		t.Fatalf("restart: %+v", got)
	}
	old := reopened.ClusterHostDetails()["local"]
	renamed := old
	renamed.Price = "$8/month"
	if err := reopened.ReplaceClusterHostDetails("local", ClusterHostDetailsResourceVersion("local", old), renamed, []string{"local"}); err != nil {
		t.Fatal(err)
	}
	if len(reopened.data.ClusterTraffic) != 1 {
		t.Fatal("unrelated metadata reset traffic")
	}
	setTrafficDay(t, reopened, 1)
	if len(reopened.data.ClusterTraffic) != 0 {
		t.Fatal("changed day retained baseline")
	}
	if got := sampleTraffic(t, reopened, at.Add(90*time.Second), 3000, 1000, 10090); got.ReceivedBytes != 0 {
		t.Fatal(got)
	}
	setTrafficDay(t, reopened, 0)
	values, err := reopened.UpdateClusterTraffic(nil, []string{"local"}, at, time.UTC)
	if err != nil || len(values) != 0 || len(reopened.data.ClusterTraffic) != 0 {
		t.Fatalf("unset: %+v %v", values, err)
	}
}

func TestClusterTrafficRolloverOfflineRebootAndCounterRollback(t *testing.T) {
	at := time.Date(2026, 9, 30, 23, 59, 45, 0, time.UTC)
	t.Run("short boundary interval", func(t *testing.T) {
		s := trafficStore(t, 1)
		sampleTraffic(t, s, at, 1000, 2000, 10000)
		got := sampleTraffic(t, s, at.Add(30*time.Second), 1600, 2800, 10030)
		if got.ReceivedBytes != 300 || got.SentBytes != 400 || got.Partial || !got.Estimated {
			t.Fatalf("prorated: %+v", got)
		}
	})
	t.Run("offline across boundary", func(t *testing.T) {
		s := trafficStore(t, 1)
		sampleTraffic(t, s, at, 1000, 2000, 10000)
		values, err := s.UpdateClusterTraffic(nil, []string{"local"}, at.Add(5*time.Minute), time.UTC)
		if err != nil || values["local"].Available || values["local"].ReceivedBytes != 0 {
			t.Fatalf("rollover without data: %+v %v", values, err)
		}
		got := sampleTraffic(t, s, at.Add(time.Hour), 20000, 25000, 13600)
		if !got.Available || !got.Partial || got.ReceivedBytes != 0 {
			t.Fatalf("must not borrow prior cycle: %+v", got)
		}
		got = sampleTraffic(t, s, at.Add(time.Hour+30*time.Second), 20100, 25200, 13630)
		if got.ReceivedBytes != 100 || got.SentBytes != 200 {
			t.Fatal(got)
		}
	})
	t.Run("reboot and interface rollback", func(t *testing.T) {
		s := trafficStore(t, 1)
		sampleTraffic(t, s, at.Add(-time.Hour), 1000, 2000, 10000)
		got := sampleTraffic(t, s, at.Add(-time.Hour+30*time.Second), 100, 200, 10)
		if got.ReceivedBytes != 100 || got.SentBytes != 200 || !got.Partial {
			t.Fatal(got)
		}
		got = sampleTraffic(t, s, at.Add(-time.Hour+60*time.Second), 50, 300, 40)
		if got.ReceivedBytes != 100 || got.SentBytes != 300 {
			t.Fatal(got)
		}
	})
}

func TestClusterTrafficWriteFailureBackupValidationAndBounds(t *testing.T) {
	s := trafficStore(t, 1)
	at := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	sampleTraffic(t, s, at, 1000, 2000, 10000)
	previous := s.data.ClusterTraffic["local"]
	path := s.path
	s.path = filepath.Join(t.TempDir(), "missing", "state.json")
	nextAt := at.Add(30 * time.Second)
	values, err := s.UpdateClusterTraffic(map[string]ClusterTrafficSample{"local": {ReceivedAt: nextAt, CollectedAt: nextAt, Received: 1200, Sent: 2300, Uptime: 10030}}, []string{"local"}, nextAt, time.UTC)
	if err == nil || values["local"].Available || s.data.ClusterTraffic["local"] != previous {
		t.Fatal("failed persist consumed delta")
	}
	s.path = path
	if got := sampleTraffic(t, s, nextAt, 1200, 2300, 10030); got.ReceivedBytes != 200 {
		t.Fatal(got)
	}
	if err := s.CreateInitialAdmin(User{ID: "admin", Username: "admin", PasswordHash: strings.Repeat("h", 32), Role: "admin", CreatedAt: at, UpdatedAt: at}); err != nil {
		t.Fatal(err)
	}
	data, err := s.ExportIdentity()
	if err != nil || ValidateIdentityBackup(data) != nil {
		t.Fatalf("backup: %v", err)
	}
	destination := trafficStore(t, 0)
	if err := destination.RestoreIdentity(data); err != nil {
		t.Fatal(err)
	}
	if got := sampleTraffic(t, destination, at.Add(time.Minute), 1400, 2600, 10060); got.ReceivedBytes != 400 {
		t.Fatal(got)
	}
	var invalid diskState
	if err := json.Unmarshal(data, &invalid); err != nil {
		t.Fatal(err)
	}
	record := invalid.ClusterTraffic["local"]
	record.Period.ReceivedBytes = maxTrafficBytes + 1
	invalid.ClusterTraffic["local"] = record
	bad, _ := json.Marshal(invalid)
	if ValidateIdentityBackup(bad) == nil {
		t.Fatal("accepted corrupt counter")
	}
	corruptPath := filepath.Join(t.TempDir(), "corrupt.json")
	if err := os.WriteFile(corruptPath, bad, 0600); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open(corruptPath); err == nil {
		opened.Close()
		t.Fatal("accepted corrupt state")
	}
	after, _ := os.ReadFile(corruptPath)
	if string(after) != string(bad) {
		t.Fatal("rewrote corrupt state")
	}
	tooMany := make([]string, MaxClusterHostOrder+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprint(i)
	}
	if _, err := s.UpdateClusterTraffic(nil, tooMany, at, time.UTC); !errors.Is(err, ErrInvalidRecord) {
		t.Fatal(err)
	}
	if _, err := s.UpdateClusterTraffic(nil, []string{}, at, time.UTC); err != nil || len(s.data.ClusterTraffic) != 0 {
		t.Fatal("removed host was not pruned")
	}
}

func TestClusterTrafficStaleSampleClockTimezoneAndSaturation(t *testing.T) {
	s := trafficStore(t, 1)
	at := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	values, err := s.UpdateClusterTraffic(map[string]ClusterTrafficSample{"local": {ReceivedAt: at, CollectedAt: at}}, []string{"local"}, at.Add(10*time.Minute), time.UTC)
	if err != nil || values["local"].Available {
		t.Fatalf("stale baseline: %+v %v", values, err)
	}
	sampleTraffic(t, s, at.Add(11*time.Minute), 0, 0, 10000)
	got := sampleTraffic(t, s, at.Add(12*time.Minute), maxTrafficBytes, maxTrafficBytes, 10060)
	if got.ReceivedBytes != maxTrafficBytes {
		t.Fatal(got)
	}
	got = sampleTraffic(t, s, at.Add(13*time.Minute), 100, 100, 20)
	if got.ReceivedBytes != maxTrafficBytes || !got.Partial {
		t.Fatal("counter overflow")
	}
	values, err = s.UpdateClusterTraffic(nil, []string{"local"}, at.AddDate(0, -1, 0), time.UTC)
	if err != nil || values["local"].Available {
		t.Fatal("clock rollback reopened period")
	}
	zone, _ := time.LoadLocation("Asia/Shanghai")
	values, err = s.UpdateClusterTraffic(nil, []string{"local"}, at, zone)
	if err != nil || values["local"].Available || values["local"].ReceivedBytes != 0 {
		t.Fatal("timezone changed without rebaselining")
	}
}

func BenchmarkClusterTraffic101Hosts(b *testing.B) {
	s, err := Open(filepath.Join(b.TempDir(), "state.json"))
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	s.data.ClusterHostDetails = make(map[string]ClusterHostDetails)
	ids := make([]string, MaxClusterHostOrder)
	samples := make(map[string]ClusterTrafficSample)
	for i := range ids {
		ids[i] = fmt.Sprint(i)
		s.data.ClusterHostDetails[ids[i]] = ClusterHostDetails{TrafficResetDay: 1}
	}
	at := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		now := at.Add(time.Duration(n) * 30 * time.Second)
		for _, id := range ids {
			samples[id] = ClusterTrafficSample{ReceivedAt: now, CollectedAt: now, Uptime: uint64(n * 30), Received: uint64(n * 1000)}
		}
		if _, err := s.UpdateClusterTraffic(samples, ids, now, time.UTC); err != nil {
			b.Fatal(err)
		}
	}
	data, _ := json.Marshal(s.data)
	b.ReportMetric(float64(len(data)), "state-bytes")
}
