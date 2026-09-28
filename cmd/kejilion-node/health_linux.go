//go:build linux

package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func readUpdateHealthFile(path string) ([]byte, error) {
	return readTrustedRuntimeFile(path, maxUpdateHealthBytes)
}

func readTrustedRuntimeFile(path string, maximum int64) ([]byte, error) {
	dir, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if !dir.IsDir() || dir.Mode().Perm() != 0o750 || !rootOwned(dir) {
		return nil, errors.New("unsafe health directory")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	dirStat := dir.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || !rootOwned(info) || info.Mode().Perm() != 0o640 || stat.Gid != dirStat.Gid || stat.Nlink != 1 || info.Size() > maximum {
		return nil, errors.New("unsafe health file")
	}
	content, err := io.ReadAll(io.LimitReader(f, maximum+1))
	if int64(len(content)) > maximum {
		return nil, errors.New("health file limit")
	}
	return content, err
}
