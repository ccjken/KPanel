//go:build linux

package main

import (
	"github.com/kejilion/kejilion-panel/internal/contract"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNodeCheckStatusRootWriterAndUntrustedPaths(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("root-owned runtime file test")
	}
	path := filepath.Join(t.TempDir(), "runtime", "check-status.json")
	value := contract.ServiceCheckSummary{Epoch: strings.Repeat("a", 32), IntervalSeconds: 300, Available: true, Items: []contract.ServiceCheckStatus{}}
	if err := writeNodeCheckStatus(path, value, 12345); err != nil {
		t.Fatal(err)
	}
	data, err := readTrustedRuntimeFile(path, contract.MaxServiceCheckSummaryBytes)
	if err != nil || decodeNodeCheckStatus(data, time.Now()) == nil {
		t.Fatal("trusted snapshot not readable", err)
	}
	if err = os.Chmod(path, 0660); err != nil {
		t.Fatal(err)
	}
	if _, err = readTrustedRuntimeFile(path, contract.MaxServiceCheckSummaryBytes); err == nil {
		t.Fatal("group-writable snapshot trusted")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("/etc/passwd", path); err != nil {
		t.Fatal(err)
	}
	if _, err = readTrustedRuntimeFile(path, contract.MaxServiceCheckSummaryBytes); err == nil {
		t.Fatal("symlink trusted")
	}
	if err = writeNodeCheckStatus(path, value, 12345); err != nil {
		t.Fatal("atomic publisher should replace rather than follow symlink", err)
	}
}
