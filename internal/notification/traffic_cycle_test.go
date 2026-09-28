package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestCumulativeTrafficUsesPeriodAndRearmsAcrossCycles(t *testing.T) {
	clock := &notificationTestClock{now: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)}
	source := newNotificationTestHost(clock.Now())
	telegram := &notificationTestTelegram{}
	s := configureNotificationTestService(t, t.TempDir(), source, telegram, clock)
	defer s.Close()
	rules := DefaultRules()
	rules.CPUEnabled, rules.MemoryEnabled, rules.DiskEnabled, rules.HostOfflineEnabled, rules.SSHLoginEnabled = false, false, false, false, false
	rules.TrafficTotalReceivedEnabled, rules.TrafficTotalSentEnabled = true, false
	rules.TrafficTotalReceivedThresholdGiB = 1
	if _, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Locale: "en-US", Rules: rules, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	source.host.LastSnapshot.Telemetry.Network.ReceivedBytes = 100 << 30
	source.host.TrafficPeriod = &contract.TrafficPeriod{Available: true, ReceivedBytes: 100, StartedAt: clock.Now().AddDate(0, 0, -19)}
	for range 2 {
		if err := s.evaluate(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if telegram.messageCount() != 0 {
		t.Fatal("alert used raw counter")
	}
	source.host.TrafficPeriod.Available = false
	if err := s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if telegram.messageCount() != 0 {
		t.Fatal("unavailable period fell back to raw")
	}
	source.host.TrafficPeriod.Available = true
	source.host.TrafficPeriod.ReceivedBytes = 2 << 30
	for range 2 {
		if err := s.evaluate(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if telegram.messageCount() != 1 {
		t.Fatal("threshold not deduplicated")
	}
	clock.Advance(31 * 24 * time.Hour)
	source.host.TrafficPeriod.StartedAt = source.host.TrafficPeriod.StartedAt.AddDate(0, 1, 0)
	source.host.TrafficPeriod.ReceivedBytes = 3 << 30 // Higher than last cycle; value rollback alone cannot detect this.
	telegram.sendErr = errors.New("offline")
	if err := s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
	history, _ := s.history.snapshot()
	if len(history.Events) != 1 || history.Events[0].Delivery != "failed" {
		t.Fatalf("new cycle: %+v", history.Events)
	}
	clock.Advance(2 * time.Minute)
	source.host.TrafficPeriod = nil
	telegram.sendErr = nil
	if err := s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
	history, _ = s.history.snapshot()
	if history.Events[0].Delivery != "cancelled" || history.Events[0].LastErrorCode != "traffic_cycle_changed" {
		t.Fatal("retried old cycle after disabling accounting")
	}
	if telegram.messageCount() != 2 {
		t.Fatal("legacy counter was not restored")
	}
}

func TestCumulativeTrafficReconfigurationCancelsPendingOldUsage(t *testing.T) {
	for _, kind := range []string{"month-end date change", "disable and re-enable"} {
		t.Run(kind, func(t *testing.T) {
			clock := &notificationTestClock{now: time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)}
			source := newNotificationTestHost(clock.Now())
			telegram := &notificationTestTelegram{sendErr: errors.New("offline")}
			s := configureNotificationTestService(t, t.TempDir(), source, telegram, clock)
			defer s.Close()
			rules := DefaultRules()
			rules.CPUEnabled, rules.MemoryEnabled, rules.DiskEnabled, rules.HostOfflineEnabled, rules.SSHLoginEnabled = false, false, false, false, false
			rules.TrafficTotalReceivedEnabled, rules.TrafficTotalSentEnabled = true, false
			rules.TrafficTotalReceivedThresholdGiB = 1
			if _, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Locale: "en-US", Rules: rules, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
				t.Fatal(err)
			}
			source.host.TrafficPeriod = &contract.TrafficPeriod{ID: "first", Available: true, ReceivedBytes: 2 << 30, StartedAt: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)}
			if err := s.evaluate(context.Background()); err != nil {
				t.Fatal(err)
			}
			clock.Advance(6 * time.Minute)
			source.host.TrafficPeriod.ReceivedBytes = 0
			if kind == "month-end date change" {
				source.host.TrafficPeriod.EndsAt = source.host.TrafficPeriod.EndsAt.AddDate(0, 0, 1)
			} else {
				source.host.TrafficPeriod.ID = "second"
			}
			telegram.sendErr = nil
			if err := s.evaluate(context.Background()); err != nil {
				t.Fatal(err)
			}
			history, _ := s.history.snapshot()
			if len(history.Events) != 1 || history.Events[0].Delivery != "cancelled" || telegram.messageCount() != 0 {
				t.Fatalf("stale usage alert: %+v", history.Events)
			}
		})
	}
}
