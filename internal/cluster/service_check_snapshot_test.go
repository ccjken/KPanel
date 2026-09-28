package cluster

import (
	"encoding/json"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"strings"
	"testing"
	"time"
)

func TestServiceCheckSummaryNegotiatedButNotPersisted(t *testing.T) {
	now := time.Now().UTC()
	telemetry := contract.HostTelemetry{CollectedAt: now, ServiceChecks: &contract.ServiceCheckSummary{Epoch: strings.Repeat("a", 32), IntervalSeconds: 300, Available: true, Items: []contract.ServiceCheckStatus{}}}
	data, err := json.Marshal(HostSnapshot{Telemetry: telemetry})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "serviceChecks") {
		t.Fatal("rollback-incompatible snapshot persisted")
	}
	if cloneTelemetry(telemetry).ServiceChecks == nil {
		t.Fatal("runtime copy lost service summary")
	}
	if telemetryForFederation(telemetry, "").ServiceChecks != nil {
		t.Fatal("summary leaked to old center")
	}
	if telemetryForFederation(telemetry, ServiceChecksCapability).ServiceChecks == nil {
		t.Fatal("capable center lost summary")
	}
	telemetry.ServiceChecks.Items = append(telemetry.ServiceChecks.Items, contract.ServiceCheckStatus{ID: "bad"})
	if cloneTelemetry(telemetry).ServiceChecks != nil {
		t.Fatal("malformed optional summary retained")
	}
}
