package appmarket

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// A job's PTY belongs to its detached worker, not the Agent process. This
// private socket forwards only a bounded window size and acknowledges ioctl
// completion. It never injects a command into the application's stdin.
func serveTerminalResize(path string, terminal terminalProcess) (func(), error) {
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = listener.Close()
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			_ = connection.SetDeadline(time.Now().Add(time.Second))
			var size [4]byte
			if _, err := io.ReadFull(connection, size[:]); err == nil {
				rows, columns := binary.BigEndian.Uint16(size[:2]), binary.BigEndian.Uint16(size[2:])
				accepted := byte(0)
				if validAppTerminalSize(rows, columns) && terminal.Resize(rows, columns) == nil {
					accepted = 1
				}
				_, _ = connection.Write([]byte{accepted})
			}
			_ = connection.Close()
		}
	}()
	return func() { _ = listener.Close(); <-done }, nil
}

func validAppTerminalSize(rows, columns uint16) bool {
	return rows > 0 && rows <= 500 && columns > 0 && columns <= 1000
}

func (registry *appJobRegistry) resizePath(id string) string {
	return registry.statePath(id) + ".resize"
}

func (s *Service) ResizeAppJobTerminal(id string, rows, columns uint16) error {
	if s.jobs == nil || !appJobIDPattern.MatchString(id) {
		return ErrNotFound
	}
	if !validAppTerminalSize(rows, columns) {
		return fmt.Errorf("%w: invalid terminal dimensions", ErrForbidden)
	}
	record, err := s.jobs.read(id)
	if err != nil {
		return ErrNotFound
	}
	if !record.Interactive || !record.InputOpen || record.Status != "running" || s.jobs.cancelRequested(id) {
		return fmt.Errorf("%w: interactive terminal is not open", ErrConflict)
	}
	connection, err := net.DialTimeout("unix", s.jobs.resizePath(id), time.Second)
	if err != nil {
		return fmt.Errorf("%w: terminal resize is unavailable", ErrConflict)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(time.Second))
	var size [4]byte
	binary.BigEndian.PutUint16(size[:2], rows)
	binary.BigEndian.PutUint16(size[2:], columns)
	if _, err := connection.Write(size[:]); err != nil {
		return fmt.Errorf("%w: terminal resize failed", ErrConflict)
	}
	var response [1]byte
	if _, err := io.ReadFull(connection, response[:]); err != nil || response[0] != 1 {
		return fmt.Errorf("%w: terminal resize was not applied", ErrConflict)
	}
	return nil
}

// Never unlink a live worker's socket during Agent restart or job recovery.
// A stopped worker removes its own socket; the bounded job pruner removes
// stale sockets left by an unclean worker exit together with the job record.
func removeTerminalResize(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
