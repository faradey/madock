//go:build !windows

// Package lockfile keeps one madock process at a time out of a shared file.
package lockfile

import (
	"errors"
	"os"
	"syscall"
)

// Hold takes the exclusive lock on path and returns what releases it, calling
// waiting first if somebody else holds it. The kernel drops it when the process
// ends, so a madock that dies holds nothing. A process that calls it twice
// without releasing waits for itself.
//
// Read-only on purpose: flock needs no more, and a file created by one `sudo
// madock` stays readable for everybody else.
func Hold(path string, waiting func()) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o666)
	if err != nil {
		return nil, err
	}
	fd := int(file.Fd())
	err = flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		waiting()
		err = flock(fd, syscall.LOCK_EX)
	}
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() { _ = file.Close() }, nil
}

func flock(fd, how int) error {
	for {
		err := syscall.Flock(fd, how)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}
