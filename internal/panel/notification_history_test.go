package panel

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestNotificationHistoryRequiresSessionAndValidatesFilters(t *testing.T) {
	server, tokenPath := newTestServer(t)
	path := clusterNotificationsPath + "/history"
	if response := performRequest(server, http.MethodGet, path, nil, nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d", response.Code)
	}
	session, csrf := bootstrapCookies(t, server, tokenPath)
	for _, query := range []string{"", "?host=local&rule=cpu&kind=alert&delivery=local_only&limit=20", "?since=2026-09-01T00:00:00Z&until=2026-09-28T00:00:00Z"} {
		response := authenticatedRequest(server, http.MethodGet, path+query, nil, session, csrf, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("query %s = %d %s", query, response.Code, response.Body.String())
		}
	}
	for _, query := range []string{"?rule=unknown", "?delivery=secret", "?kind=x", "?cursor=0", "?cursor=-1", "?limit=101", "?limit=-1", "?host=a&host=b", "?since=invalid", "?until=2020-01-01T00:00:00Z&since=2021-01-01T00:00:00Z", "?token=secret", "?search=%zz"} {
		response := authenticatedRequest(server, http.MethodGet, path+query, nil, session, csrf, nil)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("query %s = %d %s", query, response.Code, response.Body.String())
		}
	}
}

func TestNotificationHistoryUnicodeSearchLimits(t *testing.T) {
	for _, search := range []string{strings.Repeat("中", 200), strings.Repeat("😀", 100)} {
		raw := url.Values{"search": {search}, "host": {strings.Repeat("机", 85)}, "rule": {"cpu"}, "kind": {"alert"}, "delivery": {"local_only"}, "since": {"2026-09-01T00:00:00Z"}}.Encode()
		if _, err := parseNotificationHistoryQuery(raw); err != nil {
			t.Fatalf("valid Unicode search rejected: %v", err)
		}
	}
	for _, search := range []string{strings.Repeat("中", 201), strings.Repeat("😀", 101), string([]byte{0xff})} {
		if _, err := parseNotificationHistoryQuery(url.Values{"search": {search}}.Encode()); err == nil {
			t.Fatal("invalid or oversized search accepted")
		}
	}
	if _, err := parseNotificationHistoryQuery("search=" + strings.Repeat("a", 4096)); err == nil {
		t.Fatal("oversized raw query accepted")
	}
}
