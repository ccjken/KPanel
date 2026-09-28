package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

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
	input := clusterHostDetailsInput{ClusterHostDetails: store.ClusterHostDetails{ExpiresOn: "2027-09-28", Price: "$5/month", TrafficResetDay: 31}, ExpectedResourceVersion: initial}
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
	if public.Items[0].Price != "$5/month" || public.Items[0].ExpiresOn != "2027-09-28" || public.Items[0].TrafficResetDay != 31 {
		t.Fatalf("public metadata missing: %s", encoded)
	}
	for _, forbidden := range []string{"resourceVersion", "expectedResourceVersion", "origin", "peerFingerprint", "remoteNodeId"} {
		if strings.Contains(string(encoded), `"`+forbidden+`"`) {
			t.Fatalf("leaked %s", forbidden)
		}
	}
	if result = authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers); result.Code != http.StatusConflict {
		t.Fatalf("stale save: %d", result.Code)
	}
	input.ExpectedResourceVersion = updated.ResourceVersion
	input.TrafficResetDay = 32
	body, _ = json.Marshal(input)
	if result = authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers); result.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid day: %d", result.Code)
	}
	input.ClusterHostDetails = store.ClusterHostDetails{}
	body, _ = json.Marshal(input)
	if result = authenticatedRequest(s, http.MethodPut, path, body, session, csrf, headers); result.Code != http.StatusOK {
		t.Fatalf("clear: %d", result.Code)
	}
	public = s.clusterShareSnapshot(context.Background(), settings, "share-version")
	encoded, _ = json.Marshal(public.Items[0])
	if strings.Contains(string(encoded), "expiresOn") || strings.Contains(string(encoded), "price") || strings.Contains(string(encoded), "trafficResetDay") {
		t.Fatalf("empty metadata exposed: %s", encoded)
	}
	if result = authenticatedRequest(s, http.MethodPut, "/api/v1/cluster/hosts/missing/details", body, session, csrf, headers); result.Code != http.StatusNotFound {
		t.Fatalf("unknown host: %d", result.Code)
	}
}
