package panel

import (
	"encoding/json"
	"github.com/kejilion/kejilion-panel/internal/notification"
	"net/http"
	"strings"
	"testing"
)

func TestServiceCheckNotificationsAuthenticationCSRFAndConflict(t *testing.T) {
	server, tokenPath := newTestServer(t)
	if res := performRequest(server, http.MethodGet, serviceCheckNotificationsPath, nil, nil); res.Code != http.StatusUnauthorized {
		t.Fatal(res.Code)
	}
	session, csrf := bootstrapCookies(t, server, tokenPath)
	res := authenticatedRequest(server, http.MethodGet, serviceCheckNotificationsPath, nil, session, csrf, nil)
	if res.Code != http.StatusOK || strings.Contains(res.Body.String(), "credential") {
		t.Fatal(res.Code, res.Body.String())
	}
	var snapshot notification.CheckAlertSnapshot
	if err := json.Unmarshal(res.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	input := notification.CheckAlertInput{CheckAlertSettings: snapshot.CheckAlertSettings, ExpectedResourceVersion: snapshot.ResourceVersion}
	input.Enabled = true
	body, _ := json.Marshal(input)
	headers := map[string]string{"Content-Type": "application/json", "Origin": "http://panel.test"}
	if res = authenticatedRequest(server, http.MethodPut, serviceCheckNotificationsPath, body, session, csrf, headers); res.Code != http.StatusForbidden {
		t.Fatal(res.Code)
	}
	headers["X-CSRF-Token"] = csrf.Value
	if res = authenticatedRequest(server, http.MethodPut, serviceCheckNotificationsPath, body, session, csrf, headers); res.Code != http.StatusOK {
		t.Fatal(res.Code, res.Body.String())
	}
	if res = authenticatedRequest(server, http.MethodPut, serviceCheckNotificationsPath, body, session, csrf, headers); res.Code != http.StatusConflict {
		t.Fatal(res.Code, res.Body.String())
	}
	if res = authenticatedRequest(server, http.MethodGet, serviceCheckNotificationsPath+"?unexpected=1", nil, session, csrf, nil); res.Code != http.StatusBadRequest {
		t.Fatal(res.Code)
	}
}
func TestServiceCheckBackupKeepsSubscriptionsButDisablesDelivery(t *testing.T) {
	name := "notifications/service-check-alerts-v1.json"
	if !panelBackupPath(name) || panelBackupPath(name+".previous") {
		t.Fatal("backup allowlist")
	}
	data := []byte(`{"schemaVersion":1,"settings":{"enabled":true,"repeat":false,"subscriptions":[{"hostId":"node","checkId":"health","revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]},"generation":42,"incidents":{"node:health":{"active":true}}}`)
	clean, err := sanitizeBackupFile(name, data)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Settings   notification.CheckAlertSettings `json:"settings"`
		Incidents  map[string]any                  `json:"incidents"`
		Generation uint64                          `json:"generation"`
	}
	if err = json.Unmarshal(clean, &result); err != nil {
		t.Fatal(err)
	}
	if result.Settings.Enabled || len(result.Settings.Subscriptions) != 1 || len(result.Incidents) != 0 || result.Generation != 0 {
		t.Fatalf("unsafe restore data: %s", clean)
	}
}
