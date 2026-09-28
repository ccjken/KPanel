package monitoring

import (
	"context"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckStatusRealSamplesRedactTargetsAndRejectOldGeneration(t *testing.T) {
	now := time.Now().UTC()
	service, err := New(Config{StateDir: t.TempDir(), System: &fakeSystemSource{summary: testSummary(1, 1)}, OperatorLatency: &fakeOperatorLatencyProber{}, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	check := Check{ID: "health", Kind: "http", Name: "Health", Target: "https://example.com/private?token=secret"}
	_, err = service.ReplaceChecks(ReplaceChecksInput{ExpectedResourceVersion: service.Checks().ResourceVersion, Items: []Check{check}})
	if err != nil {
		t.Fatal(err)
	}
	generation := service.checkGeneration
	for i := 0; i < 3; i++ {
		service.recordCheckStatus([]operatorLatencyResult{{target: check, errorCode: "timeout"}}, now, generation)
		now = now.Add(5 * time.Minute)
	}
	status := service.CheckStatus()
	if !contract.ValidServiceCheckSummary(&status, now) || status.Items[0].Failures != 3 || status.Items[0].Target != "example.com" {
		t.Fatalf("summary=%+v", status)
	}
	status.Items[0].Name = "mutated"
	if service.CheckStatus().Items[0].Name == "mutated" {
		t.Fatal("snapshot aliases live state")
	}
	_, err = service.ReplaceChecks(ReplaceChecksInput{ExpectedResourceVersion: service.Checks().ResourceVersion, Items: []Check{}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ReplaceChecks(ReplaceChecksInput{ExpectedResourceVersion: service.Checks().ResourceVersion, Items: []Check{check}})
	if err != nil {
		t.Fatal(err)
	}
	service.recordCheckStatus([]operatorLatencyResult{{target: check, errorCode: "timeout"}}, now, generation)
	if service.CheckStatus().Items[0].State != "unknown" {
		t.Fatal("old inflight sample resurrected deleted check")
	}
	service.recordCheckStatus([]operatorLatencyResult{{target: check, reachable: true}}, now, service.checkGeneration)
	service.recordCheckStatus([]operatorLatencyResult{{target: check, errorCode: "timeout"}}, now.Add(-time.Minute), service.checkGeneration)
	if service.CheckStatus().Items[0].State != "up" {
		t.Fatal("older observation overwrote latest")
	}
}
func TestCheckStatusSurvivesHistoryWriteFailure(t *testing.T) {
	now := time.Now().UTC()
	service, err := New(Config{StateDir: t.TempDir(), System: &fakeSystemSource{summary: testSummary(1, 1)}, OperatorLatency: &fakeOperatorLatencyProber{}, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	// A regular file in place of the history directory forces an append failure.
	bad := filepath.Join(t.TempDir(), "not-a-directory")
	if err = os.WriteFile(bad, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	service.stateDir = bad
	if service.Sample(context.Background()) == nil {
		t.Fatal("expected history failure")
	}
	summary := service.CheckStatus()
	if summary.Sequence != 1 || len(summary.Items) == 0 || summary.Items[0].State != "up" {
		t.Fatalf("history failure suppressed observations: %+v", summary)
	}
	for _, item := range summary.Items {
		if strings.Contains(item.Target, "?") {
			t.Fatal("secret target")
		}
	}
}
