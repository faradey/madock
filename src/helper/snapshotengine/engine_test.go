package snapshotengine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fake struct{ live bool }

func (f fake) Live() bool                            { return f.live }
func (fake) Capture(Project, string) (string, error) { return "ref", nil }
func (fake) Restore(Project, string) error           { return nil }
func (fake) Clone(Project, Project) error            { return nil }
func (fake) Drop(Project, string) error              { return nil }

func TestNoKeyAndTarAreMadocksOwnPath(t *testing.T) {
	reset()
	for _, v := range []string{"", "tar", " tar "} {
		e, name, err := For(map[string]string{"snapshot/engine": v})
		if e != nil || err != nil || name != "tar" {
			t.Errorf("%q: got %v, %q, %v", v, e, name, err)
		}
	}
}

// A project set to an engine the binary does not have must be refused, not
// copied with tar: that would stop a project the owner expected to keep running.
func TestAnUnregisteredEngineIsRefused(t *testing.T) {
	reset()
	Register("btrfs", fake{live: true})
	e, name, err := For(map[string]string{"snapshot/engine": "zfs"})
	if e != nil || name != "zfs" || !errors.Is(err, ErrUnknown) {
		t.Fatalf("got %v, %q, %v", e, name, err)
	}
	if got := err.Error(); got != `snapshot engine is not available in this build: snapshot/engine is "zfs"; available: tar, btrfs` {
		t.Errorf("message: %s", got)
	}
}

func TestStops(t *testing.T) {
	reset()
	if !Stops(nil, nil) {
		t.Error("tar must stop the project")
	}
	if Stops(fake{live: true}, map[string]string{}) {
		t.Error("a live engine must not stop the project")
	}
	if !Stops(fake{live: true}, map[string]string{"snapshot/live": "stop"}) {
		t.Error("snapshot/live=stop must stop the project")
	}
	if !Stops(fake{live: false}, map[string]string{}) {
		t.Error("an engine that is not live must stop the project")
	}
}

func TestMarker(t *testing.T) {
	dir := t.TempDir()
	if _, ok, err := ReadMarker(dir); ok || err != nil {
		t.Fatalf("a tar snapshot has no marker: %v %v", ok, err)
	}
	if err := WriteMarker(dir, Marker{Engine: "zfs", Ref: "tank/madock/shop@snapshot-x"}); err != nil {
		t.Fatal(err)
	}
	m, ok, err := ReadMarker(dir)
	if !ok || err != nil || m.Engine != "zfs" || m.Ref != "tank/madock/shop@snapshot-x" {
		t.Fatalf("got %+v %v %v", m, ok, err)
	}
	if err := os.WriteFile(filepath.Join(dir, MarkerFile), []byte(`{"engine":"zfs"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadMarker(dir); err == nil {
		t.Error("a marker without a ref must be an error, not a tar snapshot")
	}
}

func TestRegisterTwicePanics(t *testing.T) {
	reset()
	Register("zfs", fake{})
	defer func() {
		if recover() == nil {
			t.Error("second registration did not panic")
		}
	}()
	Register("zfs", fake{})
}
