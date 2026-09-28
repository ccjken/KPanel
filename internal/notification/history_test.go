package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func localHistoryService(t *testing.T) (*Service, *notificationHostSource, *notificationTestClock) {
	t.Helper()
	clock := &notificationTestClock{now: time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC)}
	source := newNotificationTestHost(clock.Now())
	source.host.ID, source.host.IsLocal = "local", true
	source.host.LastSnapshot.Telemetry.CPU.UsagePercent = 95
	s, err := NewService(Config{DataDir: t.TempDir(), Hosts: source, Now: clock.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, source, clock
}

func tickHistory(t *testing.T, s *Service, clock *notificationTestClock, count int) {
	t.Helper()
	for range count {
		if err := s.evaluate(context.Background()); err != nil {
			t.Fatal(err)
		}
		clock.Advance(30 * time.Second)
	}
}

func TestLocalHistoryWithoutChannelRecoveryAndRestart(t *testing.T) {
	s, source, clock := localHistoryService(t)
	if !s.Snapshot().LocalRecording || s.Snapshot().Enabled {
		t.Fatal("unexpected default settings")
	}
	tickHistory(t, s, clock, 3)
	page, err := s.History(HistoryQuery{HostID: "local", Rule: "cpu", Delivery: "local_only"})
	if err != nil || len(page.Items) != 1 || page.Items[0].Kind != "alert" {
		t.Fatalf("history = %#v, %v", page, err)
	}
	alertID := page.Items[0].ID
	restarted, err := NewService(Config{DataDir: filepath.Dir(s.store.directory), Hosts: source, Now: clock.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	tickHistory(t, restarted, clock, 3)
	page, _ = restarted.History(HistoryQuery{})
	if len(page.Items) != 1 {
		t.Fatal("restart duplicated the alert")
	}
	telemetry := source.host.LastSnapshot.Telemetry
	telemetry.CPU.UsagePercent = 10
	source.setTelemetry(telemetry)
	tickHistory(t, restarted, clock, 1)
	page, _ = restarted.History(HistoryQuery{})
	if len(page.Items) != 2 || page.Items[0].Kind != "recovery" || page.Items[0].RelatedEventID != alertID {
		t.Fatalf("recovery missing: %#v", page.Items)
	}
}

func TestHistoryDeliveryFailureDoesNotSuppressRecoveryOrDuplicateRecords(t *testing.T) {
	s, source, clock := localHistoryService(t)
	tg := &notificationTestTelegram{}
	s.telegram = tg
	_, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Rules: DefaultRules(), TelegramBotToken: testBotToken, ExpectedResourceVersion: s.Snapshot().ResourceVersion})
	if err != nil {
		t.Fatal(err)
	}
	tg.sendErr = errors.New("private transport failure")
	tickHistory(t, s, clock, 3)
	telemetry := source.host.LastSnapshot.Telemetry
	telemetry.CPU.UsagePercent = 10
	source.setTelemetry(telemetry)
	tickHistory(t, s, clock, 1)
	page, _ := s.History(HistoryQuery{})
	if len(page.Items) != 2 || page.Items[0].Kind != "recovery" || page.Items[1].Delivery != "failed" {
		t.Fatalf("history = %#v", page.Items)
	}
	tg.sendErr = nil
	clock.Advance(5 * time.Minute)
	tickHistory(t, s, clock, 1)
	page, _ = s.History(HistoryQuery{})
	if len(page.Items) != 2 || page.Items[0].Delivery != "sent" || page.Items[1].Delivery != "sent" || tg.messageCount() != 2 {
		t.Fatalf("retry = %#v", page.Items)
	}
	encoded, _ := json.Marshal(page)
	for _, forbidden := range []string{testBotToken, "channelFingerprint", "chatId", "private transport failure"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("API leaked %s", forbidden)
		}
	}
}

func TestLocalHistoryNeverBackfillsAfterConnecting(t *testing.T) {
	s, _, clock := localHistoryService(t)
	tickHistory(t, s, clock, 3)
	tg := &notificationTestTelegram{}
	s.telegram = tg
	if _, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Rules: DefaultRules(), TelegramBotToken: testBotToken, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	tickHistory(t, s, clock, 3)
	if tg.messageCount() != 0 {
		t.Fatal("local history was backfilled")
	}
}

func TestHistoryFailedWriteDoesNotAdvanceAndCanRecover(t *testing.T) {
	s, _, clock := localHistoryService(t)
	tickHistory(t, s, clock, 2)
	original := s.history.path
	s.history.path = filepath.Join(s.store.directory, "missing", "history.json")
	if err := s.evaluate(context.Background()); err == nil {
		t.Fatal("expected storage failure")
	}
	if s.Snapshot().LocalRecording {
		t.Fatal("write failure reported as healthy")
	}
	if s.getAlertState("local:cpu").Active {
		t.Fatal("failed persistence advanced evaluator")
	}
	s.history.path = original
	tickHistory(t, s, clock, 1)
	page, err := s.History(HistoryQuery{})
	if err != nil || len(page.Items) != 1 || !s.Snapshot().LocalRecording {
		t.Fatalf("failed to recover: %#v, %v", page, err)
	}
}

func TestCorruptHistoryIsPreservedAndReported(t *testing.T) {
	s, source, clock := localHistoryService(t)
	if err := os.WriteFile(s.history.path, []byte("broken history"), 0o600); err != nil {
		t.Fatal(err)
	}
	restarted, err := NewService(Config{DataDir: filepath.Dir(s.store.directory), Hosts: source, Now: clock.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	if restarted.Snapshot().LocalRecording {
		t.Fatal("corruption reported as healthy")
	}
	if _, err := restarted.History(HistoryQuery{}); err == nil {
		t.Fatal("corruption became empty history")
	}
	if err := restarted.evaluate(context.Background()); err == nil {
		t.Fatal("corruption overwritten")
	}
	content, _ := os.ReadFile(s.history.path)
	if string(content) != "broken history" {
		t.Fatal("original corrupt file was lost")
	}
}

func TestHistoryFiltersAndStableCursorAcrossInsertAndExpiry(t *testing.T) {
	s, _, clock := localHistoryService(t)
	state, _ := s.history.snapshot()
	for i := 1; i <= 120; i++ {
		state.Sequence++
		state.Events = append(state.Events, storedEvent{Event: Event{ID: fmt.Sprint(i), CreatedAt: clock.Now(), HostID: "local", HostName: "本机", IsLocal: true, Rule: "cpu", Kind: "alert", Message: "CPU threshold", Delivery: "local_only"}})
	}
	if err := s.history.commit(state); err != nil {
		t.Fatal(err)
	}
	page, err := s.History(HistoryQuery{Limit: 50, Search: "THRESHOLD", HostID: "local", Rule: "cpu", Kind: "alert", Delivery: "local_only"})
	if err != nil || len(page.Items) != 50 || page.NextCursor != "71" {
		t.Fatalf("page = %#v, %v", page, err)
	}
	state.Sequence++
	insert := state.Events[0]
	insert.ID = "121"
	state.Events = append(state.Events, insert)
	if err := s.history.commit(state); err != nil {
		t.Fatal(err)
	}
	page, _ = s.History(HistoryQuery{Before: 71, Limit: 50})
	if page.Items[0].ID != "70" || page.NextCursor != "21" {
		t.Fatalf("cursor skipped or duplicated: %#v", page)
	}
	page, _ = s.History(HistoryQuery{Before: 21, Limit: 50})
	if len(page.Items) != 20 || page.NextCursor != "" {
		t.Fatalf("last page = %#v", page)
	}
	clock.Advance(31 * 24 * time.Hour)
	page, _ = s.History(HistoryQuery{})
	if len(page.Items) != 0 {
		t.Fatal("expired history visible")
	}
}

func TestHistoryCountAndByteLimits(t *testing.T) {
	s, _, clock := localHistoryService(t)
	state, _ := s.history.snapshot()
	for i := 1; i <= MaxHistoryEvents+10; i++ {
		state.Sequence++
		state.Events = append(state.Events, storedEvent{Event: Event{ID: fmt.Sprint(i), CreatedAt: clock.Now(), HostID: "local", HostName: "本机", Rule: "cpu", Kind: "alert", Message: strings.Repeat("x", 4096), Delivery: "local_only"}})
	}
	pruneHistory(&state, clock.Now())
	if len(state.Events) != MaxHistoryEvents {
		t.Fatal("count unbounded")
	}
	if err := s.history.commit(state); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(s.history.path)
	if info.Size() > MaxHistoryBytes {
		t.Fatal("bytes unbounded")
	}
	page, _ := s.History(HistoryQuery{Limit: 1})
	if page.Items[0].ID != fmt.Sprint(MaxHistoryEvents+10) {
		t.Fatal("pruned newest event")
	}
}

func TestHistoryStopsRetriesAfterDisablingPush(t *testing.T) {
	s, _, clock := localHistoryService(t)
	tg := &notificationTestTelegram{}
	s.telegram = tg
	if _, err := s.Configure(context.Background(), UpdateInput{Enabled: true, Rules: DefaultRules(), TelegramBotToken: testBotToken, ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	tg.sendErr = errors.New("unavailable")
	tickHistory(t, s, clock, 3)
	if _, err := s.Configure(context.Background(), UpdateInput{Enabled: false, Rules: DefaultRules(), ExpectedResourceVersion: s.Snapshot().ResourceVersion}); err != nil {
		t.Fatal(err)
	}
	tg.sendErr = nil
	clock.Advance(5 * time.Minute)
	tickHistory(t, s, clock, 1)
	page, _ := s.History(HistoryQuery{})
	if page.Items[0].Delivery != "cancelled" || tg.messageCount() != 0 {
		t.Fatalf("disabled retry sent: %#v", page)
	}
}

func BenchmarkHistoryFilteredPage(b *testing.B) {
	now := time.Now()
	history := historyState{SchemaVersion: 1, Rules: DefaultRules(), Alerts: map[string]alertState{}, Sequence: MaxHistoryEvents}
	for i := 1; i <= MaxHistoryEvents; i++ {
		history.Events = append(history.Events, storedEvent{Event: Event{ID: fmt.Sprint(i), CreatedAt: now, HostID: "local", HostName: "本机", Rule: "cpu", Kind: "alert", Message: strings.Repeat("CPU threshold ", 100), Delivery: "local_only"}})
	}
	s := &Service{history: &historyStore{state: history}, now: func() time.Time { return now }}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := s.History(HistoryQuery{HostID: "local", Rule: "cpu", Search: "threshold", Limit: 50}); err != nil {
			b.Fatal(err)
		}
	}
}
