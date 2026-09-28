package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/kejilion/kejilion-panel/internal/cluster"
)

func expiryTestService(t *testing.T, now time.Time, zone *time.Location) (*Service, *notificationHostSource, *notificationTestClock, map[string]HostExpiry) {
	t.Helper()
	clock := &notificationTestClock{now: now}
	hosts := newNotificationTestHost(now)
	details := map[string]HostExpiry{}
	s, err := NewService(Config{DataDir: t.TempDir(), Hosts: hosts, Now: clock.Now,
		Timezone:     func(context.Context) *time.Location { return zone },
		HostExpiries: func() map[string]HostExpiry { return details },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, hosts, clock, details
}

func expiryEvents(t *testing.T, s *Service) []Event {
	t.Helper()
	page, err := s.History(HistoryQuery{Rule: serverExpiryRuleKey})
	if err != nil {
		t.Fatal(err)
	}
	return page.Items
}

func setExpiryClock(clock *notificationTestClock, now time.Time) {
	clock.mu.Lock()
	clock.now = now
	clock.mu.Unlock()
}

func TestHostExpiryMilestonesRestartAndClockRollback(t *testing.T) {
	start := time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)
	s, hosts, clock, details := expiryTestService(t, start, time.UTC)
	details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}
	for i, day := range []int{23, 27, 29, 30} {
		setExpiryClock(clock, time.Date(2026, 9, day, 8, 0, 0, 0, time.UTC))
		tickHistory(t, s, clock, 2)
		if got := len(expiryEvents(t, s)); got != i+1 {
			t.Fatalf("day %d: got %d reminders", day, got)
		}
	}
	restarted, err := NewService(Config{DataDir: filepath.Dir(s.store.directory), Hosts: hosts, Now: clock.Now, HostExpiries: s.hostExpiries})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	for _, day := range []int{30, 23, 29, 27} {
		setExpiryClock(clock, time.Date(2026, 9, day, 8, 0, 0, 0, time.UTC))
		tickHistory(t, restarted, clock, 1)
	}
	events := expiryEvents(t, restarted)
	if len(events) != 4 {
		t.Fatal("restart or clock rollback repeated a milestone")
	}
	for i, days := range []string{"0", "1", "3", "7"} {
		if events[i].Delivery != "local_only" || events[i].Kind != "info" || !strings.Contains(events[i].Message, "剩余天数："+days) {
			t.Fatalf("incorrect event: %+v", events[i])
		}
	}
}

func TestHostExpiryOptInRenewalAndOfflineHost(t *testing.T) {
	s, hosts, clock, details := expiryTestService(t, time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC), time.UTC)
	hosts.host.LastSnapshot = nil
	details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30"}
	tickHistory(t, s, clock, 1)
	if len(expiryEvents(t, s)) != 0 {
		t.Fatal("unchecked host notified")
	}
	details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}
	tickHistory(t, s, clock, 1)
	if len(expiryEvents(t, s)) != 1 {
		t.Fatal("offline host did not notify")
	}
	details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30"}
	tickHistory(t, s, clock, 1)
	details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}
	tickHistory(t, s, clock, 1)
	if len(expiryEvents(t, s)) != 1 {
		t.Fatal("checkbox toggle duplicated a recorded milestone")
	}
	details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-10-30", Enabled: true}
	tickHistory(t, s, clock, 1)
	setExpiryClock(clock, time.Date(2026, 10, 23, 8, 0, 0, 0, time.UTC))
	tickHistory(t, s, clock, 1)
	events := expiryEvents(t, s)
	if len(events) != 2 || !strings.Contains(events[0].Message, "2026-10-30") {
		t.Fatal("renewed date did not get its own reminder")
	}
}

func TestHostExpiryCalendarTimezoneAndDST(t *testing.T) {
	for _, tc := range []struct{ zone, now, expiry string }{
		{"Asia/Shanghai", "2026-09-22T16:30:00Z", "2026-09-30"},
		{"America/New_York", "2026-03-07T17:00:00Z", "2026-03-14"},
		{"America/New_York", "2026-10-31T16:00:00Z", "2026-11-07"},
	} {
		t.Run(tc.now, func(t *testing.T) {
			zone, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			now, _ := time.Parse(time.RFC3339, tc.now)
			s, hosts, clock, details := expiryTestService(t, now, zone)
			details[hosts.host.ID] = HostExpiry{ExpiresOn: tc.expiry, Enabled: true}
			tickHistory(t, s, clock, 1)
			events := expiryEvents(t, s)
			if len(events) != 1 || !strings.Contains(events[0].Message, "剩余天数：7") {
				t.Fatalf("calendar mismatch: %+v", events)
			}
		})
	}
}

func TestHostExpirySkipsInvalidPastAndMissedDates(t *testing.T) {
	s, hosts, clock, details := expiryTestService(t, time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC), time.UTC)
	for _, date := range []string{"", "2026-02-30", "0000-01-01", "2026-09-22", "2026-09-29", "2027-01-01"} {
		details[hosts.host.ID] = HostExpiry{ExpiresOn: date, Enabled: true}
		tickHistory(t, s, clock, 1)
	}
	if len(expiryEvents(t, s)) != 0 {
		t.Fatal("unexpected catch-up reminder")
	}
}

func TestHostExpiryDoesNotRecordWithoutDedupCapacity(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)
	s, hosts, _, _ := expiryTestService(t, now, time.UTC)
	for i := range MaxAlertStates {
		s.alerts[fmt.Sprintf("host-%d:cpu", i)] = alertState{}
	}
	details := HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}
	recorded := 0
	record := func(cluster.Host, string, string, string) (bool, bool) { recorded++; return true, true }
	if s.handleHostExpiry(hosts.host, details, now, "zh-CN", record) || recorded != 0 {
		t.Fatal("recorded a reminder without capacity to remember it")
	}
	delete(s.alerts, "host-0:cpu")
	if !s.handleHostExpiry(hosts.host, details, now, "zh-CN", record) || recorded != 1 {
		t.Fatal("did not recover when capacity became available")
	}
	s.handleHostExpiry(hosts.host, details, now, "zh-CN", record)
	if recorded != 1 {
		t.Fatal("repeated a milestone at capacity")
	}
}

func TestHostExpiryPersistsBeforeDeliveryAndRetriesWithoutNewEvent(t *testing.T) {
	s, hosts, clock, details := expiryTestService(t, time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC), time.UTC)
	details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}
	tg := &notificationTestTelegram{}
	s.telegram = tg
	if _, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Rules: DefaultRules(), TelegramBotToken: testBotToken, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	original := s.history.path
	s.history.path = filepath.Join(s.store.directory, "missing", "history.json")
	if err := s.evaluate(context.Background()); err == nil {
		t.Fatal("expected persistence failure")
	}
	if tg.messageCount() != 0 || s.getAlertState(hosts.host.ID+":"+serverExpiryRuleKey).ExpiryNotifiedMask != 0 {
		t.Fatal("failed persistence advanced state or sent a message")
	}
	s.history.path = original
	tg.sendErr = errors.New("unavailable")
	tickHistory(t, s, clock, 2)
	if events := expiryEvents(t, s); len(events) != 1 || events[0].Delivery != "failed" {
		t.Fatalf("failed delivery: %+v", events)
	}
	tg.sendErr = nil
	clock.Advance(5 * time.Minute)
	tickHistory(t, s, clock, 1)
	if events := expiryEvents(t, s); len(events) != 1 || events[0].Delivery != "sent" || tg.messageCount() != 1 {
		t.Fatalf("retry: %+v", events)
	}
	encoded, _ := json.Marshal(expiryEvents(t, s))
	if strings.Contains(string(encoded), "expiryDate") || strings.Contains(string(encoded), "channelFingerprint") {
		t.Fatal("private retry metadata leaked")
	}
}

func TestHostExpiryCancelsPendingAfterUncheckRenewalOrRemoval(t *testing.T) {
	for _, change := range []string{"uncheck", "renew", "remove"} {
		t.Run(change, func(t *testing.T) {
			s, hosts, clock, details := expiryTestService(t, time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC), time.UTC)
			details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}
			tg := &notificationTestTelegram{sendErr: errors.New("unavailable")}
			s.telegram = tg
			if _, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Rules: DefaultRules(), TelegramBotToken: testBotToken, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
				t.Fatal(err)
			}
			tickHistory(t, s, clock, 1)
			switch change {
			case "uncheck":
				details[hosts.host.ID] = HostExpiry{ExpiresOn: "2026-09-30"}
			case "renew":
				details[hosts.host.ID] = HostExpiry{ExpiresOn: "2027-09-30", Enabled: true}
			case "remove":
				delete(details, hosts.host.ID)
			}
			tg.sendErr = nil
			clock.Advance(5 * time.Minute)
			tickHistory(t, s, clock, 1)
			if events := expiryEvents(t, s); len(events) != 1 || events[0].Delivery != "cancelled" || tg.messageCount() != 0 {
				t.Fatalf("stale reminder sent: %+v", events)
			}
		})
	}
}
