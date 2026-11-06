//go:build !windows

package lockfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestASecondHolderWaitsForTheFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	release, err := Hold(path, func() { t.Error("the first holder was told to wait") })
	if err != nil {
		t.Fatal(err)
	}

	got := make(chan struct{})
	waited := make(chan struct{})
	go func() {
		second, err := Hold(path, func() { close(waited) })
		if err != nil {
			t.Error(err)
		} else {
			second()
		}
		close(got)
	}()

	select {
	case <-got:
		t.Fatal("the second holder got the lock while the first held it")
	case <-waited:
	case <-time.After(5 * time.Second):
		t.Fatal("the second holder was not told it waits")
	}

	release()

	select {
	case <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("the second holder did not get the lock after it was released")
	}
}

// One `sudo madock` leaves the file root's and 0644; everybody else still locks it.
func TestAFileOthersCannotWriteCanStillBeHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	if err := os.WriteFile(path, nil, 0o444); err != nil {
		t.Fatal(err)
	}
	release, err := Hold(path, func() {})
	if err != nil {
		t.Fatal(err)
	}
	release()
}
