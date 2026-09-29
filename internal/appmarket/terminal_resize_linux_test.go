//go:build linux

package appmarket

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestAppTerminalResizeChild(t *testing.T) {
	if os.Getenv("KPANEL_TEST_RESIZE_CHILD") != "1" {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGWINCH)
	defer signal.Stop(signals)
	fmt.Println("ready")
	for range signals {
		size, err := unix.IoctlGetWinsize(0, unix.TIOCGWINSZ)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%d %d\n", size.Row, size.Col)
	}
}

func TestAppTerminalResizeReachesWorkerAndRedrawsOnReattach(t *testing.T) {
	// Use a short socket path even when the Go test runner uses a long temp root.
	root, err := os.MkdirTemp("", "app-resize-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	// Custom state paths may legally exceed Unix sun_path's 108-byte limit.
	root = filepath.Join(root, strings.Repeat("nested-state-", 10))
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	registry := &appJobRegistry{stateDir: root, jobs: make(map[string]appJobRecord)}
	id := strings.Repeat("a", 32)
	if err := registry.put(appJobRecord{AppJob: AppJob{ID: id, Status: "running", Interactive: true, InputOpen: true, CreatedAt: time.Now().UTC()}}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The child reads the real kernel PTY dimensions. SIGWINCH represents a TUI
	// redraw, including an attachment that has exactly the previous dimensions.
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, executable, "-test.run=^TestAppTerminalResizeChild$")
	command.Env = append(os.Environ(), "KPANEL_TEST_RESIZE_CHILD=1")
	terminal, err := startTerminalProcess(command, 36, 120)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = terminal.Kill(); _ = terminal.Wait(); _ = terminal.Close() }()
	stop, err := serveTerminalResize(registry.resizePath(id), terminal)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	info, err := os.Stat(registry.resizePath(id))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("socket permissions: %v %v", info, err)
	}
	lines := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(terminal)
		for scanner.Scan() {
			lines <- strings.TrimSpace(scanner.Text())
		}
		close(lines)
	}()
	wantLine := func(want string) {
		t.Helper()
		select {
		case got := <-lines:
			if got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		case <-ctx.Done():
			t.Fatal("no PTY redraw")
		}
	}
	wantLine("ready")
	service := &Service{jobs: registry}
	for _, size := range [][2]uint16{{24, 80}, {24, 80}, {40, 140}, {12, 45}} {
		if err := service.ResizeAppJobTerminal(id, size[0], size[1]); err != nil {
			t.Fatal(err)
		}
		wantLine(fmtSize(size))
	}
	// A reloaded Agent uses the same detached worker endpoint without resetting
	// the task or injecting anything into the application's input stream.
	service = &Service{jobs: &appJobRegistry{stateDir: root, jobs: make(map[string]appJobRecord)}}
	if err := service.ResizeAppJobTerminal(id, 30, 90); err != nil {
		t.Fatal(err)
	}
	wantLine("30 90")
	if err := registry.requestCancel(id); err != nil {
		t.Fatal(err)
	}
	if err := service.ResizeAppJobTerminal(id, 24, 80); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func fmtSize(size [2]uint16) string {
	return fmt.Sprintf("%d %d", size[0], size[1])
}
