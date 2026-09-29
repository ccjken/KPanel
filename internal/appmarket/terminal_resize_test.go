package appmarket

import (
	"errors"
	"testing"
	"time"
)

func TestAppTerminalResizeRejectsInvalidAndInactiveJobs(t *testing.T) {
	registry := &appJobRegistry{stateDir: t.TempDir(), jobs: make(map[string]appJobRecord)}
	service := &Service{jobs: registry}
	id := "0123456789abcdef0123456789abcdef"
	if err := service.ResizeAppJobTerminal("../bad", 24, 80); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := service.ResizeAppJobTerminal(id, 24, 80); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	for _, size := range [][2]uint16{{0, 80}, {24, 0}, {501, 80}, {24, 1001}} {
		if err := service.ResizeAppJobTerminal(id, size[0], size[1]); !errors.Is(err, ErrForbidden) {
			t.Fatalf("size=%v err=%v", size, err)
		}
	}
	for _, status := range []string{"queued", "succeeded", "failed", "cancelled", "running"} {
		record := appJobRecord{AppJob: AppJob{ID: id, Status: status, Interactive: true, InputOpen: status != "running", CreatedAt: time.Now().UTC()}}
		if err := registry.put(record); err != nil {
			t.Fatal(err)
		}
		if err := service.ResizeAppJobTerminal(id, 24, 80); !errors.Is(err, ErrConflict) {
			t.Fatalf("status=%s err=%v", status, err)
		}
	}
}
