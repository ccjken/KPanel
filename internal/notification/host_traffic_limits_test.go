package notification

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/cluster"
	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestHostTrafficLimitsPriorityDirectionClearAndRestart(t *testing.T) {
	s, source, clock := localHistoryService(t)
	source.host.LastSnapshot.Telemetry.CPU.UsagePercent = 0
	source.host.LastSnapshot.Telemetry.Network = contract.NetworkSummary{ReceivedBytes: 20 << 30, SentBytes: 3 << 30}
	limits := map[string]HostTrafficLimits{"local": {ReceivedGiB: 100}}
	s.hostTrafficLimits = func() map[string]HostTrafficLimits { return limits }
	rules := DefaultRules()
	rules.TrafficTotalReceivedEnabled, rules.TrafficTotalSentEnabled = true, true
	rules.TrafficTotalReceivedThresholdGiB, rules.TrafficTotalSentThresholdGiB = 1, 2
	configure := func() {
		t.Helper()
		if _, err := s.Configure(context.Background(), UpdateInput{Rules: rules, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
			t.Fatal(err)
		}
	}
	configure()
	tickHistory(t, s, clock, 1)
	page, _ := s.History(HistoryQuery{})
	if len(page.Items) != 1 || page.Items[0].Rule != cumulativeTrafficSentRuleKey {
		t.Fatalf("override did not replace lower global / independent sent: %+v", page.Items)
	}
	limits["local"] = HostTrafficLimits{ReceivedGiB: 10}
	tickHistory(t, s, clock, 1)
	page, _ = s.History(HistoryQuery{})
	if len(page.Items) != 2 || !strings.Contains(page.Items[0].Message, "10.0 GB") {
		t.Fatalf("host limit: %+v", page.Items)
	}
	// Global changes must not re-arm a host override or the untouched direction.
	rules.TrafficTotalReceivedEnabled = false
	rules.TrafficTotalReceivedThresholdGiB = 8
	configure()
	tickHistory(t, s, clock, 1)
	page, _ = s.History(HistoryQuery{})
	if len(page.Items) != 2 {
		t.Fatal("global change duplicated host alert")
	}
	restarted, err := NewService(Config{DataDir: filepath.Dir(s.store.directory), Hosts: source, Now: clock.Now, HostTrafficLimits: s.hostTrafficLimits})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	tickHistory(t, restarted, clock, 1)
	page, _ = restarted.History(HistoryQuery{})
	if len(page.Items) != 2 {
		t.Fatal("restart duplicated override")
	}
	delete(limits, "local")
	tickHistory(t, restarted, clock, 1)
	if _, exists := restarted.alertStateSnapshot()["local:"+cumulativeTrafficReceivedRuleKey]; exists {
		t.Fatal("clear did not inherit global disabled")
	}
	// A positive override independently enables a globally disabled direction.
	limits["local"] = HostTrafficLimits{ReceivedGiB: 5}
	tickHistory(t, restarted, clock, 1)
	page, _ = restarted.History(HistoryQuery{})
	if len(page.Items) != 3 || page.Items[0].Delivery != "local_only" {
		t.Fatalf("override enable/local recording: %+v", page.Items)
	}
}

func TestHostTrafficLimitsPeriodAndPendingInvalidation(t *testing.T) {
	clock := &notificationTestClock{now: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)}
	source := newNotificationTestHost(clock.Now())
	tg := &notificationTestTelegram{sendErr: errors.New("offline")}
	s := configureNotificationTestService(t, t.TempDir(), source, tg, clock)
	defer s.Close()
	limits := map[string]HostTrafficLimits{source.host.ID: {ReceivedGiB: 2, SentGiB: 5}}
	s.hostTrafficLimits = func() map[string]HostTrafficLimits { return limits }
	source.host.LastSnapshot.Telemetry.Network = contract.NetworkSummary{ReceivedBytes: 100 << 30, SentBytes: 100 << 30}
	source.host.TrafficPeriod = &contract.TrafficPeriod{Available: true, ReceivedBytes: 1 << 30, SentBytes: 3 << 30, ID: "test"}
	tickHistory(t, s, clock, 1)
	history, _ := s.history.snapshot()
	if len(history.Events) != 0 {
		t.Fatal("used raw totals instead of period")
	}
	source.host.TrafficPeriod.ReceivedBytes = 3 << 30
	tickHistory(t, s, clock, 1)
	history, _ = s.history.snapshot()
	if len(history.Events) != 1 || history.Events[0].Delivery != "failed" || history.Events[0].TrafficThresholdGiB != 2 {
		t.Fatalf("override push with global rule disabled: %+v", history.Events)
	}
	limits[source.host.ID] = HostTrafficLimits{ReceivedGiB: 10, SentGiB: 5}
	clock.Advance(6 * time.Minute)
	tg.sendErr = nil
	tickHistory(t, s, clock, 1)
	history, _ = s.history.snapshot()
	if history.Events[0].Delivery != "cancelled" || history.Events[0].LastErrorCode != "traffic_threshold_changed" || tg.messageCount() != 0 {
		t.Fatalf("stale retry: %+v", history.Events)
	}
	// Clearing only received falls back to its enabled global threshold.
	rules := s.Snapshot().Rules
	rules.TrafficTotalReceivedEnabled = true
	rules.TrafficTotalReceivedThresholdGiB = 1
	if _, err := s.Configure(context.Background(), UpdateInput{Rules: rules, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	limits[source.host.ID] = HostTrafficLimits{SentGiB: 5}
	tickHistory(t, s, clock, 1)
	history, _ = s.history.snapshot()
	if len(history.Events) != 2 || history.Events[1].TrafficThresholdGiB != 1 || history.Events[1].Delivery != "local_only" {
		t.Fatal("clear did not restore global threshold")
	}
}

func TestHostTrafficLimitsLegacyThresholdMigration(t *testing.T) {
	rules := DefaultRules()
	rules.TrafficTotalReceivedEnabled = true
	rules.TrafficTotalReceivedThresholdGiB = 7
	h := historyState{Rules: rules, Alerts: map[string]alertState{"host:traffic-total-received": {Active: true}}, Events: []storedEvent{{Event: Event{Rule: cumulativeTrafficReceivedRuleKey}}}}
	reconcileTrafficThresholds(&h, rules, nil)
	if !h.Alerts["host:traffic-total-received"].Active || h.Alerts["host:traffic-total-received"].TrafficThresholdGiB != 7 || h.Events[0].TrafficThresholdGiB != 7 {
		t.Fatal("legacy state lost known threshold/dedup")
	}
	rules.TrafficTotalReceivedThresholdGiB = 10
	reconcileTrafficThresholds(&h, rules, nil)
	if len(h.Alerts) != 0 || h.Events[0].TrafficThresholdGiB != 7 {
		t.Fatal("legacy retry changed its original threshold")
	}
}

func TestHostTrafficLimitsIsolationAndCancellationDuringDelivery(t *testing.T) {
	clock := &notificationTestClock{now: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)}
	first := newNotificationTestHost(clock.Now()).host
	first.LastSnapshot.Telemetry.Network.ReceivedBytes = 3 << 30
	second, third := first, first
	second.ID, third.ID = "second", "third"
	limits := map[string]HostTrafficLimits{first.ID: {ReceivedGiB: 2}, second.ID: {ReceivedGiB: 1}}
	tg := &expiryAfterFirstTelegram{}
	s, err := NewService(Config{DataDir: t.TempDir(), Hosts: expiryMultiHostSource{[]cluster.Host{first, second, third}}, Telegram: tg, Now: clock.Now,
		HostTrafficLimits: func() map[string]HostTrafficLimits { return limits }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Rules: DefaultRules(), TelegramBotToken: testBotToken, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	tg.afterFirst = func() { limits[second.ID] = HostTrafficLimits{ReceivedGiB: 10} }
	tickHistory(t, s, clock, 1)
	history, _ := s.history.snapshot()
	if len(history.Events) != 2 || tg.messageCount() != 1 || history.Events[1].Delivery != "cancelled" || history.Events[1].LastErrorCode != "traffic_threshold_changed" {
		t.Fatalf("host isolation/batch check: %+v sends=%d", history.Events, tg.messageCount())
	}
}
