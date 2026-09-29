package notification

import (
	"context"
	"github.com/kejilion/kejilion-panel/internal/cluster"
	"path/filepath"
	"testing"
	"time"
)

func TestGlobalExpiryMigratesLegacyOnceAndPreservesExplicitChoice(t *testing.T) {
	for _, tc := range []struct {
		name   string
		legacy HostExpiry
		want   bool
	}{
		{"absent", HostExpiry{}, false}, {"date-only", HostExpiry{ExpiresOn: "2026-09-30"}, false},
		{"legacy-on", HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}, true}, {"invalid-date", HostExpiry{ExpiresOn: "2026-02-30", Enabled: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)
			hosts := newNotificationTestHost(now)
			cfg := Config{DataDir: t.TempDir(), Hosts: hosts, Now: func() time.Time { return now }, HostExpiries: func() map[string]HostExpiry { return map[string]HostExpiry{hosts.host.ID: tc.legacy} }}
			// Persist a pre-global-rule state to exercise upgrading an existing file.
			old, err := Open(cfg.DataDir)
			if err != nil {
				t.Fatal(err)
			}
			state := old.stateSnapshot()
			state.ResourceVersion = configResourceVersion(state.Settings, state.Telegram)
			if err = old.commitState(state); err != nil {
				t.Fatal(err)
			}
			s, err := NewService(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			got := s.Snapshot()
			if got.Rules.HostExpiryEnabled == nil || *got.Rules.HostExpiryEnabled != tc.want {
				t.Fatalf("migration: %+v", got.Rules)
			}
			// API callers must not mutate the saved rule through a shared pointer.
			*got.Rules.HostExpiryEnabled = !tc.want
			if *s.Snapshot().Rules.HostExpiryEnabled != tc.want {
				t.Fatal("snapshot aliases stored rule")
			}
			setGlobalExpiryRule(t, s, false)
			restarted, err := NewService(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer restarted.Close()
			if *restarted.Snapshot().Rules.HostExpiryEnabled {
				t.Fatal("legacy opt-in re-enabled an explicit global off")
			}
		})
	}
}

func TestGlobalExpiryCoversAllDatedHostsAndNewHosts(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)
	first := newNotificationTestHost(now).host
	second := first
	second.ID = "offline"
	second.LastSnapshot = nil
	missing := first
	missing.ID = "undated"
	hosts := &expiryMultiHostSource{hosts: []cluster.Host{first, second, missing}}
	details := map[string]HostExpiry{first.ID: {ExpiresOn: "2026-09-30"}, second.ID: {ExpiresOn: "2026-09-30"}}
	s, err := NewService(Config{DataDir: t.TempDir(), Hosts: hosts, Now: func() time.Time { return now }, HostExpiries: func() map[string]HostExpiry { return details }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(expiryEvents(t, s)) != 0 {
		t.Fatal("default off ignored")
	}
	setGlobalExpiryRule(t, s, true)
	if err = s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(expiryEvents(t, s)) != 2 {
		t.Fatal("global on did not cover both dated hosts")
	}
	added := first
	added.ID = "new-host"
	hosts.hosts = append(hosts.hosts, added)
	details[added.ID] = HostExpiry{ExpiresOn: "2026-09-30"}
	if err = s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(expiryEvents(t, s)) != 3 {
		t.Fatal("new host requires another opt-in")
	}
	setGlobalExpiryRule(t, s, false)
	details[missing.ID] = HostExpiry{ExpiresOn: "2026-09-30", Enabled: true}
	if err = s.evaluate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(expiryEvents(t, s)) != 3 {
		t.Fatal("legacy per-host true bypasses global off")
	}
}

func TestGlobalExpiryOldClientAndFailedSavePreserveRule(t *testing.T) {
	s, _, _, _ := expiryTestService(t, time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC), time.UTC)
	oldVersion := s.Snapshot().ResourceVersion
	if _, err := s.Configure(context.Background(), UpdateInput{Rules: DefaultRules(), ExpectedResourceVersion: oldVersion}); err != nil {
		t.Fatal(err)
	}
	if !*s.Snapshot().Rules.HostExpiryEnabled {
		t.Fatal("older client omitted rule and disabled it")
	}
	before := s.Snapshot()
	rules := before.Rules
	off := false
	rules.HostExpiryEnabled = &off
	original := s.store.statePath
	s.store.statePath = filepath.Join(s.store.directory, "missing", "state.json")
	if _, err := s.Configure(context.Background(), UpdateInput{Rules: rules, ExpectedResourceVersion: before.ResourceVersion}); err == nil {
		t.Fatal("failed write reported success")
	}
	s.store.statePath = original
	after := s.Snapshot()
	if !*after.Rules.HostExpiryEnabled || after.ResourceVersion != before.ResourceVersion {
		t.Fatal("failed save changed active configuration")
	}
	if _, err := s.Configure(context.Background(), UpdateInput{Rules: rules, ExpectedResourceVersion: "stale"}); err != ErrConflict {
		t.Fatalf("stale save: %v", err)
	}
	if !*s.Snapshot().Rules.HostExpiryEnabled {
		t.Fatal("conflict changed rule")
	}
}
