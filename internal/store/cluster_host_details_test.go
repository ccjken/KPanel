package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClusterHostDetailsPersistenceConflictClearAndBoundedCleanup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	initial := ClusterHostDetailsResourceVersion("local", ClusterHostDetails{})
	value := ClusterHostDetails{ExpiresOn: "2028-02-29", Price: "$5/month", TrafficResetDay: 31}
	if err := s.ReplaceClusterHostDetails("local", initial, value, []string{"local", "remote"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceClusterHostDetails("local", initial, ClusterHostDetails{}, []string{"local"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got := s.ClusterHostDetails()["local"]; got != value {
		t.Fatalf("restart lost details: %+v", got)
	}
	copy := s.ClusterHostDetails()
	copy["local"] = ClusterHostDetails{}
	if s.ClusterHostDetails()["local"] != value {
		t.Fatal("read aliases store")
	}
	if err := s.ReplaceClusterHostDetails("remote", ClusterHostDetailsResourceVersion("remote", ClusterHostDetails{}), value, []string{"local", "remote"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceClusterHostDetails("local", ClusterHostDetailsResourceVersion("local", value), ClusterHostDetails{}, []string{"local"}); err != nil {
		t.Fatal(err)
	}
	if len(s.ClusterHostDetails()) != 0 {
		t.Fatal("clear and stale cleanup failed")
	}
}

func TestClusterHostDetailsRejectInvalidInputAndRollbackWriteFailure(t *testing.T) {
	for _, value := range []ClusterHostDetails{
		{ExpiresOn: "2027-02-29"}, {ExpiresOn: "2026-9-28"}, {ExpiresOn: "0000-01-01"},
		{Price: strings.Repeat("贵", 41)}, {Price: "5\n/month"}, {Price: " 5 "}, {Price: string([]byte{0xff})},
		{TrafficResetDay: -1}, {TrafficResetDay: 32},
	} {
		if ValidateClusterHostDetails(value) == nil {
			t.Fatalf("accepted invalid value: %+v", value)
		}
	}
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	version := ClusterHostDetailsResourceVersion("local", ClusterHostDetails{})
	if err := s.ReplaceClusterHostDetails("missing", version, ClusterHostDetails{}, []string{"local"}); !errors.Is(err, ErrInvalidRecord) {
		t.Fatal(err)
	}
	// A file where the parent directory must be forces a failure on every OS.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(blocker, "state.json")
	if err := s.ReplaceClusterHostDetails("local", version, ClusterHostDetails{Price: "$1"}, []string{"local"}); err == nil {
		t.Fatal("write should fail")
	}
	if len(s.ClusterHostDetails()) != 0 {
		t.Fatal("failed write changed memory")
	}
	s.path = path
}
