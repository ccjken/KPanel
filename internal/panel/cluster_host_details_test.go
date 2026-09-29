package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/cluster"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/notification"
	"github.com/kejilion/kejilion-panel/internal/store"
)

func TestClusterHostDetailsAuthorizationLifecycleAndPublicWhitelist(t *testing.T) {
	s, tokenPath := newTestServer(t)
	path := "/api/v1/cluster/hosts/local/details"
	if result := performRequest(s, http.MethodPut, path, nil, map[string]string{"Origin": "http://panel.test"}); result.Code != http.StatusUnauthorized {
		t.Fatal(result.Code)
	}
	session, csrf := bootstrapCookies(t, s, tokenPath)
	view := s.clusterHostsView(context.Background())
	initial := view.HostDetails["local"].ResourceVersion
	input := clusterHostDetailsInput{ClusterHostDetails: store.ClusterHostDetails{ExpiresOn: "2027-09-28", ExpiryReminderEnabled: true, Price: "$5/month", TrafficResetDay: 31, TrafficMonthlyQuotaGiB: 1500, TrafficCalculation: "sent", TrafficTotalReceivedThresholdGiB: 512, TrafficTotalSentThresholdGiB: 1024}, ExpectedResourceVersion: initial}
	body, _ := json.Marshal(input)
	headers := map[string]string{"Content-Type": "application/json", "Origin": "http://panel.test", "X-CSRF-Token": csrf.Value}
	for _, missing := range []string{"Origin", "X-CSRF-Token"} {
		partial := map[string]string{}
		for k, v := range headers {
			if k != missing {
				partial[k] = v
			}
		}
		result := authenticatedRequest(s, http.MethodPut, path, body, session, csrf, partial)
		if result.Code != http.StatusForbidden {
			t.Fatalf("missing %s: %d", missing, result.Code)
		}
	}
	// Prime the share cache, then prove a successful update invalidates it.
	settings := store.ClusterShare{Enabled: true, Token: strings.Repeat("a", 64)}
	before := s.clusterShareSnapshot(context.Background(), settings, "share-version")
	if before.Items[0].Price != "" {
		t.Fatal("unexpected initial price")
	}
	result := authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers)
	if result.Code != http.StatusOK {
		t.Fatalf("save: %d %s", result.Code, result.Body.String())
	}
	var updated clusterHostDetailsResponse
	if err := json.Unmarshal(result.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.ClusterHostDetails != input.ClusterHostDetails || updated.ResourceVersion == initial {
		t.Fatal("incorrect response")
	}
	if s.clusterHostsView(context.Background()).HostDetails["local"] != updated {
		t.Fatal("inventory omitted details")
	}
	public := s.clusterShareSnapshot(context.Background(), settings, "share-version")
	encoded, _ := json.Marshal(public.Items[0])
	if public.Items[0].TrafficPeriod == nil || strings.Contains(string(encoded), "sampleAt") || strings.Contains(string(encoded), "recordedAt") {
		t.Fatalf("public accounting whitelist: %s", encoded)
	}
	if public.Items[0].Price != "$5/month" || public.Items[0].ExpiresOn != "2027-09-28" || public.Items[0].TrafficResetDay != 31 {
		t.Fatalf("public metadata missing: %s", encoded)
	}
	if public.Items[0].TrafficMonthlyQuotaGiB != 1500 || public.Items[0].TrafficCalculation != "sent" {
		t.Fatalf("public quota missing: %s", encoded)
	}
	for _, forbidden := range []string{"trafficTotalReceivedThresholdGiB", "trafficTotalSentThresholdGiB", "expiryReminderEnabled", "resourceVersion", "expectedResourceVersion", "origin", "peerFingerprint", "remoteNodeId"} {
		if strings.Contains(string(encoded), `"`+forbidden+`"`) {
			t.Fatalf("leaked %s", forbidden)
		}
	}
	if result = authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers); result.Code != http.StatusConflict {
		t.Fatalf("stale save: %d", result.Code)
	}
	input.ExpectedResourceVersion = updated.ResourceVersion
	for _, invalid := range []store.ClusterHostDetails{
		{TrafficMonthlyQuotaGiB: -1}, {TrafficMonthlyQuotaGiB: 1_048_577}, {TrafficCalculation: "unknown"},
	} {
		bad := clusterHostDetailsInput{ClusterHostDetails: invalid, ExpectedResourceVersion: updated.ResourceVersion}
		payload, _ := json.Marshal(bad)
		if result = authenticatedRequest(s, http.MethodPut, path, payload, session, csrf, headers); result.Code != http.StatusUnprocessableEntity {
			t.Fatalf("invalid quota configuration: %d", result.Code)
		}
		if s.clusterHostsView(context.Background()).HostDetails["local"] != updated {
			t.Fatal("invalid request changed saved details")
		}
	}
	input.TrafficResetDay = 32
	body, _ = json.Marshal(input)
	if result = authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers); result.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid day: %d", result.Code)
	}
	input.ClusterHostDetails = store.ClusterHostDetails{}
	input.ExpiryReminderEnabled = true
	body, _ = json.Marshal(input)
	if result = authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers); result.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reminder without date: %d", result.Code)
	}
	input.ExpiryReminderEnabled = false
	body, _ = json.Marshal(input)
	if result = authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers); result.Code != http.StatusOK {
		t.Fatalf("clear: %d", result.Code)
	}
	public = s.clusterShareSnapshot(context.Background(), settings, "share-version")
	encoded, _ = json.Marshal(public.Items[0])
	if strings.Contains(string(encoded), "expiresOn") || strings.Contains(string(encoded), "price") || strings.Contains(string(encoded), "trafficResetDay") || strings.Contains(string(encoded), "trafficMonthlyQuotaGiB") || strings.Contains(string(encoded), "trafficCalculation") {
		t.Fatalf("empty metadata exposed: %s", encoded)
	}
	if result = authenticatedRequest(s, http.MethodPut, "/api/v1/cluster/hosts/missing/details", body, session, csrf, headers); result.Code != http.StatusNotFound {
		t.Fatalf("unknown host: %d", result.Code)
	}
}

func TestClusterHostDetailsExpiryReminderReachesNotificationHistory(t *testing.T) {
	s, tokenPath := newTestServer(t)
	session, csrf := bootstrapCookies(t, s, tokenPath)
	input := clusterHostDetailsInput{
		ClusterHostDetails:      store.ClusterHostDetails{ExpiresOn: time.Now().AddDate(0, 0, 7).Format("2006-01-02")},
		ExpectedResourceVersion: s.clusterHostsView(context.Background()).HostDetails["local"].ResourceVersion,
	}
	body, _ := json.Marshal(input)
	result := authenticatedRequest(s, http.MethodPut, "/api/v1/cluster/hosts/local/details", body, session, csrf,
		map[string]string{"Content-Type": "application/json", "Origin": "http://panel.test", "X-CSRF-Token": csrf.Value})
	if result.Code != http.StatusOK {
		t.Fatalf("save: %d %s", result.Code, result.Body.String())
	}
	enabled := true
	rules := notification.DefaultRules()
	rules.HostExpiryEnabled = &enabled
	notificationBody, _ := json.Marshal(notification.UpdateInput{Rules: rules, ExpectedResourceVersion: s.notifications.Snapshot().ResourceVersion})
	updated := authenticatedRequest(s, http.MethodPut, clusterNotificationsPath, notificationBody, session, csrf,
		map[string]string{"Content-Type": "application/json", "Origin": "http://panel.test", "X-CSRF-Token": csrf.Value})
	if updated.Code != http.StatusOK {
		t.Fatalf("global reminder save: %d %s", updated.Code, updated.Body.String())
	}
	s.notifications.Start(context.Background())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		page, err := s.notifications.History(notification.HistoryQuery{Rule: "server-expiry"})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) == 1 {
			if page.Items[0].HostID != "local" || page.Items[0].Delivery != "local_only" || !strings.Contains(page.Items[0].Message, input.ExpiresOn) {
				t.Fatalf("unexpected expiry event: %+v", page.Items[0])
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("saved host details did not reach the notification evaluator")
}

func TestClusterHostDetailsTrafficLimitsReachNotifications(t *testing.T) {
	s, tokenPath := newTestServer(t)
	session, csrf := bootstrapCookies(t, s, tokenPath)
	headers := map[string]string{"Content-Type": "application/json", "Origin": "http://panel.test", "X-CSRF-Token": csrf.Value}
	version := s.clusterHostsView(context.Background()).HostDetails["local"].ResourceVersion
	for _, invalid := range []string{"-1", "1048577", "1.5", `"10"`} {
		body := []byte(`{"trafficTotalSentThresholdGiB":` + invalid + `,"expectedResourceVersion":"` + version + `"}`)
		result := authenticatedRequest(s, http.MethodPut, "/api/v1/cluster/hosts/local/details", body, session, csrf, headers)
		if result.Code != http.StatusBadRequest && result.Code != http.StatusUnprocessableEntity {
			t.Fatalf("invalid %s: %d", invalid, result.Code)
		}
	}
	input := clusterHostDetailsInput{ClusterHostDetails: store.ClusterHostDetails{TrafficTotalReceivedThresholdGiB: 2, TrafficTotalSentThresholdGiB: 10}, ExpectedResourceVersion: version}
	body, _ := json.Marshal(input)
	result := authenticatedRequest(s, http.MethodPut, "/api/v1/cluster/hosts/local/details", body, session, csrf, headers)
	if result.Code != http.StatusOK {
		t.Fatalf("save: %d %s", result.Code, result.Body.String())
	}
	now := time.Now()
	s.clusterTraffic.raw = trafficTestHosts{host: cluster.Host{ID: "local", Name: "test", State: cluster.HostOnline, LastSnapshot: &cluster.HostSnapshot{ReceivedAt: now, Telemetry: contract.HostTelemetry{CollectedAt: now, Network: contract.NetworkSummary{ReceivedBytes: 3 << 30, SentBytes: 3 << 30}}}}}
	s.notifications.Start(context.Background())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		page, err := s.notifications.History(notification.HistoryQuery{Rule: "traffic-total-received"})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) == 1 {
			if page.Items[0].Delivery != "local_only" || !strings.Contains(page.Items[0].Message, "2.0 GB") {
				t.Fatalf("wrong override: %+v", page.Items)
			}
			sent, _ := s.notifications.History(notification.HistoryQuery{Rule: "traffic-total-sent"})
			if len(sent.Items) != 0 {
				t.Fatal("directions mixed")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("saved traffic thresholds did not reach evaluator")
}
