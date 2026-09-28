package notification

import (
	"context"
	"errors"
	"fmt"
	"github.com/kejilion/kejilion-panel/internal/cluster"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func checkAlertFixture(t *testing.T) (*CheckAlerts, *notificationHostSource, *notificationTestTelegram, *notificationTestClock) {
	t.Helper()
	clock := &notificationTestClock{now: time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC)}
	host := newNotificationTestHost(clock.Now())
	telegram := &notificationTestTelegram{}
	parent := configureNotificationTestService(t, t.TempDir(), host, telegram, clock)
	alerts, err := NewCheckAlerts(parent)
	if err != nil {
		t.Fatal(err)
	}
	setCheckObservation(host, clock.Now(), 1, "down", 1)
	snapshot := alerts.Snapshot(context.Background())
	_, err = alerts.Configure(context.Background(), CheckAlertInput{CheckAlertSettings: CheckAlertSettings{Enabled: true, Subscriptions: []CheckSubscription{{HostID: host.host.ID, CheckID: "health", Revision: strings.Repeat("b", 32)}}}, ExpectedResourceVersion: snapshot.ResourceVersion})
	if err != nil {
		t.Fatal(err)
	}
	return alerts, host, telegram, clock
}
func setCheckObservation(host *notificationHostSource, now time.Time, sequence uint64, state string, count int) {
	item := contract.ServiceCheckStatus{ID: "health", Revision: strings.Repeat("b", 32), Name: "HTTP Health", Target: "example.com", Kind: "http", State: state, CheckedAt: now}
	if state == "down" {
		item.Failures = count
		item.FailureSince = &now
		item.ErrorCode = "http_status"
	} else if state == "up" {
		item.Successes = count
	}
	host.mu.Lock()
	defer host.mu.Unlock()
	host.host.LastSnapshot.Telemetry.ServiceChecks = &contract.ServiceCheckSummary{Epoch: strings.Repeat("a", 32), Sequence: sequence, IntervalSeconds: 300, Available: true, Items: []contract.ServiceCheckStatus{item}}
}
func evaluateChecks(t *testing.T, s *CheckAlerts) {
	t.Helper()
	if err := s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func deliverChecks(t *testing.T, s *CheckAlerts) {
	t.Helper()
	if err := s.deliver(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestCheckAlertsCountRealSamplesPersistRetryAndCoalesceRecovery(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	baseline := telegram.messageCount()
	for i := 0; i < 5; i++ {
		evaluateChecks(t, s)
		deliverChecks(t, s)
	}
	if telegram.messageCount() != baseline {
		t.Fatal("duplicate evaluations confirmed a failure")
	}
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	evaluateChecks(t, s)
	telegram.sendErr = errors.New("network")
	deliverChecks(t, s)
	key := host.host.ID + ":health"
	pending := s.state.Incidents[key].Pending
	if pending == nil || pending.Attempts != 1 || !pending.NextAttempt.After(clock.Now()) {
		t.Fatalf("retry not scheduled: %+v", pending)
	}
	deliverChecks(t, s)
	if s.state.Incidents[key].Pending.Attempts != 1 {
		t.Fatal("retried before backoff")
	}
	restored, err := NewCheckAlerts(s.parent)
	if err != nil {
		t.Fatal(err)
	}
	s = restored
	clock.Advance(5 * time.Minute)
	setCheckObservation(host, clock.Now(), 4, "up", 1)
	evaluateChecks(t, s)
	if !s.state.Incidents[key].Active {
		t.Fatal("recovered after only one success")
	}
	clock.Advance(5 * time.Minute)
	setCheckObservation(host, clock.Now(), 5, "up", 2)
	evaluateChecks(t, s)
	if s.state.Incidents[key].Pending.Kind != "resolved" {
		t.Fatal("unsent failure and recovery did not coalesce")
	}
	telegram.sendErr = nil
	deliverChecks(t, s)
	messages := telegram.messagesSnapshot()
	if len(messages) != baseline+1 || !strings.Contains(messages[len(messages)-1], "服务曾异常") {
		t.Fatal(messages)
	}
	evaluateChecks(t, s)
	deliverChecks(t, s)
	if telegram.messageCount() != baseline+1 {
		t.Fatal("duplicate recovery")
	}
}
func TestCheckAlertsOfflinePausesRecoveryAndMissingSummaryAlerts(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	key := host.host.ID + ":health"
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	evaluateChecks(t, s)
	deliverChecks(t, s)
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 5, "up", 2)
	evaluateChecks(t, s)
	host.setState(cluster.HostOffline)
	evaluateChecks(t, s)
	deliverChecks(t, s)
	if s.state.Incidents[key].Pending == nil || s.state.Incidents[key].Pending.Kind != "recovery" {
		t.Fatal("offline discarded recovery")
	}
	before := telegram.messageCount()
	host.setState(cluster.HostOnline)
	evaluateChecks(t, s)
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 {
		t.Fatal("recovery did not resume")
	}
	host.mu.Lock()
	host.host.LastSnapshot.Telemetry.ServiceChecks = nil
	host.mu.Unlock()
	clock.Advance(12 * time.Minute)
	evaluateChecks(t, s)
	deliverChecks(t, s)
	if !strings.Contains(telegram.messagesSnapshot()[telegram.messageCount()-1], "检测数据中断") {
		t.Fatal("missing summary was not reported")
	}
}
func TestCheckAlertsUnknownAndDeletedChecksAreDistinct(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	evaluateChecks(t, s)
	for seq := uint64(2); seq <= 4; seq++ {
		clock.Advance(5 * time.Minute)
		setCheckObservation(host, clock.Now(), seq, "unknown", 0)
		evaluateChecks(t, s)
	}
	before := telegram.messageCount()
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 {
		t.Fatal("repeated unknown samples concealed outage")
	}
	host.mu.Lock()
	host.host.LastSnapshot.Telemetry.ServiceChecks.Items = nil
	host.mu.Unlock()
	evaluateChecks(t, s)
	if len(s.state.Incidents) != 0 {
		t.Fatalf("deleted checks retained incidents: %+v", s.state.Incidents)
	}
}
func TestCheckAlertsStorageFailureDoesNotAdvanceMemory(t *testing.T) {
	s, host, _, clock := checkAlertFixture(t)
	old := s.state.Generation
	s.path = filepath.Join(t.TempDir(), "missing", "state.json")
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	if err := s.evaluate(context.Background()); err == nil {
		t.Fatal("expected persistence failure")
	}
	if s.state.Generation != old || s.lastError != "storage_unavailable" {
		t.Fatal("failed commit advanced state")
	}
}
func TestCheckAlertsUnavailableEmptyCatalogIsNotDeletion(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	evaluateChecks(t, s)
	host.mu.Lock()
	host.host.LastSnapshot.Telemetry.ServiceChecks.Available = false
	host.host.LastSnapshot.Telemetry.ServiceChecks.Items = nil
	host.mu.Unlock()
	clock.Advance(12 * time.Minute)
	evaluateChecks(t, s)
	before := telegram.messageCount()
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 || !s.state.Incidents[host.host.ID+":@sampler"].Active {
		t.Fatal("unreadable catalog treated as deletion")
	}
	host.mu.Lock()
	host.host.LastSnapshot.Telemetry.ServiceChecks.Available = true
	host.mu.Unlock()
	evaluateChecks(t, s)
	if len(s.state.Incidents) != 0 {
		t.Fatal("available empty catalog was not treated as deletion")
	}
}
func TestCheckAlertsBoundedFullClusterAndPausedQueueFairness(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	evaluateChecks(t, s)
	for i := 0; i < 8; i++ {
		s.state.Generation++
		s.state.Incidents[fmt.Sprintf("000-%d:health", i)] = checkIncident{HostID: fmt.Sprintf("000-%d", i), Name: "paused", Pending: &checkDelivery{ID: s.state.Generation, Kind: "down", CreatedAt: clock.Now(), NextAttempt: clock.Now().Add(-time.Hour)}}
	}
	before := telegram.messageCount()
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 {
		t.Fatal("paused outbox starved a deliverable host")
	}
	settings := CheckAlertSettings{Enabled: true}
	for h := 0; h <= cluster.MaxHosts; h++ {
		for c := 0; c < contract.MaxServiceChecks; c++ {
			settings.Subscriptions = append(settings.Subscriptions, CheckSubscription{HostID: fmt.Sprintf("host-%d", h), CheckID: fmt.Sprintf("check-%d", c), Revision: strings.Repeat("a", 32)})
		}
	}
	if err := validateCheckSettings(settings); err != nil {
		t.Fatalf("full cluster rejected: %v", err)
	}
	settings.Subscriptions = append(settings.Subscriptions, CheckSubscription{})
	if validateCheckSettings(settings) == nil {
		t.Fatal("unbounded subscriptions")
	}
}
func TestCheckAlertsRevisionConflictAndLegacyStateCompatibility(t *testing.T) {
	s, host, _, clock := checkAlertFixture(t)
	before, err := os.ReadFile(s.parent.store.statePath)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := s.Snapshot(context.Background())
	input := CheckAlertInput{CheckAlertSettings: snapshot.CheckAlertSettings, ExpectedResourceVersion: "stale"}
	if _, err = s.Configure(context.Background(), input); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	evaluateChecks(t, s)
	host.mu.Lock()
	host.host.LastSnapshot.Telemetry.ServiceChecks.Items[0].Revision = strings.Repeat("c", 32)
	host.mu.Unlock()
	evaluateChecks(t, s)
	if len(s.state.Incidents) != 0 {
		t.Fatal("target change inherited prior incident")
	}
	after, _ := os.ReadFile(s.parent.store.statePath)
	if string(before) != string(after) {
		t.Fatal("service checks changed legacy notification file")
	}
}

func TestCheckAlertsBatchSameHostAndTraditionalLocale(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	host.mu.Lock()
	second := host.host.LastSnapshot.Telemetry.ServiceChecks.Items[0]
	second.ID = "database"
	host.host.LastSnapshot.Telemetry.ServiceChecks.Items = append(host.host.LastSnapshot.Telemetry.ServiceChecks.Items, second)
	host.mu.Unlock()
	snap := s.Snapshot(context.Background())
	snap.Subscriptions = append(snap.Subscriptions, CheckSubscription{HostID: host.host.ID, CheckID: "database", Revision: second.Revision})
	if _, err := s.Configure(context.Background(), CheckAlertInput{CheckAlertSettings: snap.CheckAlertSettings, ExpectedResourceVersion: snap.ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	evaluateChecks(t, s)
	before := telegram.messageCount()
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 {
		t.Fatal("same-host alerts were not grouped")
	}
	if s.Snapshot(context.Background()).Pending != 0 {
		t.Fatal("batch acknowledgment incomplete")
	}
	if message := serviceCheckMessage(checkIncident{Name: "test"}, checkDelivery{Kind: "down"}, clock.Now(), "zh-TW"); !strings.Contains(message, "服務異常") || !strings.Contains(message, "觀測主機") {
		t.Fatal(message)
	}
}

type benchmarkCheckHosts struct{ value cluster.HostList }

func (s benchmarkCheckHosts) Hosts(context.Context) cluster.HostList { return s.value }

type unexpectedCheckHosts struct{ t *testing.T }

func (s unexpectedCheckHosts) Hosts(context.Context) cluster.HostList {
	s.t.Fatal("idle notification worker requested telemetry")
	return cluster.HostList{}
}
func TestCheckAlertsIdleWorkersDoNotCollectTelemetry(t *testing.T) {
	s, _, _, _ := checkAlertFixture(t)
	s.parent.hosts = unexpectedCheckHosts{t}
	// Enabled with no pending delivery still must avoid a duplicate host poll.
	deliverChecks(t, s)
	s.state.Settings.Enabled = false
	evaluateChecks(t, s)
	s.state.Settings.Enabled = true
	s.state.Settings.Subscriptions = nil
	evaluateChecks(t, s)
}
func BenchmarkServiceCheckEvaluationFullCluster(b *testing.B) {
	now := time.Now().UTC()
	source := benchmarkCheckHosts{}
	settings := CheckAlertSettings{Enabled: true}
	for h := 0; h <= cluster.MaxHosts; h++ {
		id := fmt.Sprintf("host-%d", h)
		summary := &contract.ServiceCheckSummary{Epoch: strings.Repeat("a", 32), Sequence: 3, IntervalSeconds: 300, Available: true}
		for c := 0; c < contract.MaxServiceChecks; c++ {
			checkID := fmt.Sprintf("check-%d", c)
			revision := strings.Repeat("b", 32)
			summary.Items = append(summary.Items, contract.ServiceCheckStatus{ID: checkID, Name: "Health", Kind: "http", Target: "example.com", Revision: revision, CheckedAt: now, State: "down", Failures: 3, FailureSince: &now})
			settings.Subscriptions = append(settings.Subscriptions, CheckSubscription{HostID: id, CheckID: checkID, Revision: revision})
		}
		source.value.Items = append(source.value.Items, cluster.Host{ID: id, Name: id, State: cluster.HostOnline, LastSnapshot: &cluster.HostSnapshot{Telemetry: contract.HostTelemetry{CollectedAt: now, ServiceChecks: summary}}})
	}
	parent, err := NewService(Config{DataDir: b.TempDir(), Hosts: source, Now: func() time.Time { return now }})
	if err != nil {
		b.Fatal(err)
	}
	state := parent.store.stateSnapshot()
	state.Settings.Enabled = true
	if err = parent.store.commitState(state); err != nil {
		b.Fatal(err)
	}
	s, err := NewCheckAlerts(parent)
	if err != nil {
		b.Fatal(err)
	}
	s.state.Settings = settings
	if err = s.evaluate(context.Background()); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err = s.evaluate(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}
