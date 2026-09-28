package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/cluster"
	"github.com/kejilion/kejilion-panel/internal/contract"
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
	setGlobalCheckRule(t, parent, true)
	return alerts, host, telegram, clock
}
func setGlobalCheckRule(t *testing.T, parent *Service, enabled bool) {
	t.Helper()
	snapshot := parent.Snapshot()
	rules := snapshot.Rules
	rules.ServiceChecksEnabled = enabled
	if _, err := parent.Configure(context.Background(), UpdateInput{Enabled: snapshot.Enabled, Locale: snapshot.Locale, Rules: rules, ExpectedResourceVersion: snapshot.ResourceVersion}); err != nil {
		t.Fatal(err)
	}
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
	next := cloneCheckDisk(s.state)
	for len(next.Incidents) <= maxCheckIncidents {
		next.Incidents[fmt.Sprintf("limit-%d", len(next.Incidents))] = checkIncident{}
	}
	if s.commit(next) == nil {
		t.Fatal("unbounded incidents")
	}

}
func TestCheckAlertsChangedTargetDoesNotInheritIncident(t *testing.T) {
	s, host, _, clock := checkAlertFixture(t)
	before, err := os.ReadFile(s.parent.store.statePath)
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(10 * time.Minute)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	evaluateChecks(t, s)
	host.mu.Lock()
	host.host.LastSnapshot.Telemetry.ServiceChecks.Items[0].Revision = strings.Repeat("c", 32)
	host.host.LastSnapshot.Telemetry.ServiceChecks.Items[0].Failures = 1
	host.mu.Unlock()
	evaluateChecks(t, s)
	item := s.state.Incidents[host.host.ID+":health"]
	if item.Revision != strings.Repeat("c", 32) || item.Active || item.Pending != nil {
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
	evaluateChecks(t, s)
	before := telegram.messageCount()
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 {
		t.Fatal("same-host alerts were not grouped")
	}
	for _, item := range s.state.Incidents {
		if item.Pending != nil {
			t.Fatal("batch acknowledgment incomplete")
		}
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
	setGlobalCheckRule(t, s.parent, false)
	evaluateChecks(t, s)
}
func BenchmarkServiceCheckEvaluationFullCluster(b *testing.B) {
	now := time.Now().UTC()
	source := benchmarkCheckHosts{}
	for h := 0; h <= cluster.MaxHosts; h++ {
		id := fmt.Sprintf("host-%d", h)
		summary := &contract.ServiceCheckSummary{Epoch: strings.Repeat("a", 32), Sequence: 3, IntervalSeconds: 300, Available: true}
		for c := 0; c < contract.MaxServiceChecks; c++ {
			checkID := fmt.Sprintf("check-%d", c)
			revision := strings.Repeat("b", 32)
			summary.Items = append(summary.Items, contract.ServiceCheckStatus{ID: checkID, Name: "Health", Kind: "http", Target: "example.com", Revision: revision, CheckedAt: now, State: "down", Failures: 3, FailureSince: &now})
		}
		source.value.Items = append(source.value.Items, cluster.Host{ID: id, Name: id, State: cluster.HostOnline, LastSnapshot: &cluster.HostSnapshot{Telemetry: contract.HostTelemetry{CollectedAt: now, ServiceChecks: summary}}})
	}
	parent, err := NewService(Config{DataDir: b.TempDir(), Hosts: source, Now: func() time.Time { return now }})
	if err != nil {
		b.Fatal(err)
	}
	state := parent.store.stateSnapshot()
	state.Settings.Enabled = true
	state.Settings.Rules.ServiceChecksEnabled = true
	if err = parent.store.commitState(state); err != nil {
		b.Fatal(err)
	}
	s, err := NewCheckAlerts(parent)
	if err != nil {
		b.Fatal(err)
	}
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

func TestCheckAlertsGlobalRuleDefaultOffFutureHostsAndDisable(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	if DefaultRules().ServiceChecksEnabled {
		t.Fatal("rule defaults on")
	}
	setGlobalCheckRule(t, s.parent, false)
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	before := telegram.messageCount()
	evaluateChecks(t, s)
	deliverChecks(t, s)
	if len(s.state.Incidents) != 0 || telegram.messageCount() != before {
		t.Fatal("disabled rule sent or tracked alerts")
	}
	setGlobalCheckRule(t, s.parent, true)
	source := &benchmarkCheckHosts{value: host.Hosts(context.Background())}
	s.parent.hosts = source
	evaluateChecks(t, s)
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 {
		t.Fatal("existing host not covered")
	}
	added := newNotificationTestHost(clock.Now())
	added.host.ID = "new-host"
	added.host.Name = "New host"
	setCheckObservation(added, clock.Now(), 3, "down", 3)
	source.value.Items = append(source.value.Items, added.host)
	evaluateChecks(t, s)
	deliverChecks(t, s)
	if telegram.messageCount() != before+2 {
		t.Fatal("new host required another settings save")
	}
	second := source.value.Items[1].LastSnapshot.Telemetry.ServiceChecks.Items[0]
	second.ID = "new-check"
	source.value.Items[1].LastSnapshot.Telemetry.ServiceChecks.Items = append(source.value.Items[1].LastSnapshot.Telemetry.ServiceChecks.Items, second)
	evaluateChecks(t, s)
	if s.state.Incidents["new-host:new-check"].Pending == nil {
		t.Fatal("future check not covered")
	}
	setGlobalCheckRule(t, s.parent, false)
	deliverChecks(t, s)
	if telegram.messageCount() != before+2 {
		t.Fatal("disabled rule delivered pending event")
	}
	evaluateChecks(t, s)
	if len(s.state.Incidents) != 0 {
		t.Fatal("disabled rule retained outbox")
	}
	data, _ := os.ReadFile(s.parent.store.statePath)
	if strings.Contains(string(data), "serviceChecksEnabled") {
		t.Fatal("disabled extension must be omitted for rollback")
	}
	var restored persistedState
	if err := decodeState(data, &restored); err != nil || restored.Settings.Rules.ServiceChecksEnabled {
		t.Fatal(err)
	}
}
func TestCheckAlertsGlobalRulePersistsAndConflicts(t *testing.T) {
	s, _, _, _ := checkAlertFixture(t)
	snapshot := s.parent.Snapshot()
	data, err := os.ReadFile(s.parent.store.statePath)
	if err != nil {
		t.Fatal(err)
	}
	var restored persistedState
	if err = decodeState(data, &restored); err != nil || !restored.Settings.Rules.ServiceChecksEnabled {
		t.Fatal("global rule not persisted", err)
	}
	rules := snapshot.Rules
	rules.ServiceChecksEnabled = false
	if _, err = s.parent.Configure(context.Background(), UpdateInput{Enabled: snapshot.Enabled, Rules: rules, ExpectedResourceVersion: "stale"}); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if !s.parent.Snapshot().Rules.ServiceChecksEnabled {
		t.Fatal("conflict changed rule")
	}
	legacy, _ := json.Marshal(DefaultRules())
	if strings.Contains(string(legacy), "serviceChecksEnabled") {
		t.Fatal("default JSON is not legacy compatible")
	}
}
func TestCheckAlertsInitiallyUnavailableSamplerRetained(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	host.host.LastSnapshot.Telemetry.ServiceChecks.Available = false
	host.host.LastSnapshot.Telemetry.ServiceChecks.Items = nil
	evaluateChecks(t, s)
	host.host.LastSnapshot.Telemetry.ServiceChecks = nil
	clock.Advance(12 * time.Minute)
	evaluateChecks(t, s)
	before := telegram.messageCount()
	deliverChecks(t, s)
	if telegram.messageCount() != before+1 {
		t.Fatal("missing initial sampler was forgotten")
	}
}

func TestCheckAlertsLegacyCandidateDoesNotEnableGlobalRule(t *testing.T) {
	s, host, _, _ := checkAlertFixture(t)
	setGlobalCheckRule(t, s.parent, false)
	legacy := []byte(`{"schemaVersion":1,"settings":{"enabled":true,"repeat":true,"subscriptions":[]},"generation":0,"incidents":{}}`)
	if err := os.WriteFile(s.path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, err := NewCheckAlerts(s.parent)
	if err != nil || restored.parent.Snapshot().Rules.ServiceChecksEnabled {
		t.Fatal("legacy candidate must not enable global monitoring", err)
	}
	// An older node with no capability must not generate a sampling alarm.
	setGlobalCheckRule(t, s.parent, true)
	host.host.LastSnapshot.Telemetry.ServiceChecks = nil
	evaluateChecks(t, restored)
	if len(restored.state.Incidents) != 0 {
		t.Fatal("unsupported node generated service incidents")
	}
}

func TestCheckAlertsFailuresLoggedOnceUntilRecovery(t *testing.T) {
	s, host, telegram, clock := checkAlertFixture(t)
	var output bytes.Buffer
	s.logger = slog.New(slog.NewJSONHandler(&output, nil))
	path := s.path
	s.path = filepath.Join(t.TempDir(), "missing", "state.json")
	setCheckObservation(host, clock.Now(), 3, "down", 3)
	for range 2 {
		if s.evaluate(context.Background()) == nil {
			t.Fatal("expected storage failure")
		}
	}
	if strings.Count(output.String(), `"code":"storage_unavailable"`) != 1 {
		t.Fatal("storage failure must be visible and deduplicated", output.String())
	}
	s.path = path
	evaluateChecks(t, s)
	telegram.sendErr = errors.New("sensitive upstream detail")
	deliverChecks(t, s)
	clock.Advance(time.Minute)
	deliverChecks(t, s)
	if strings.Count(output.String(), `"code":"delivery_failed"`) != 1 || strings.Contains(output.String(), "sensitive upstream") {
		t.Fatal("delivery failure must be visible, deduplicated and redacted", output.String())
	}
	clock.Advance(5 * time.Minute)
	telegram.sendErr = nil
	deliverChecks(t, s)
	if strings.Count(output.String(), "Service notification worker recovered") != 2 {
		t.Fatal("missing storage/delivery recovery log", output.String())
	}
}
