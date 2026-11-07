package dockerassets

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEverySrcMountMasksMadock is the gate behind the container-isolation fix.
//
// A service that bind-mounts the project code at /var/www/html mounts the whole
// project root — `.madock` included, because `src` is a symlink to it — so a
// process in that container could write `.madock/docker/docker-compose.<os>.yml`
// and the host's next `start`/`rebuild` would read it and launch whatever it
// named. Mounting `.madock` read-only over the code closes that: the code stays
// writable, the control plane does not.
//
// This asserts every snippet that mounts the code also masks `.madock`. A new
// service copied from an old one without the mask fails here — the whole reason
// the line is enforced by a test rather than left to memory.
//
// The snapshot helper (docker/general/docker-compose-snapshot.yml, outside this
// tree) is deliberately not covered: it runs no project code, and snapshot
// restore untars `.madock` back through it, so it needs the mount writable.
func TestEverySrcMountMasksMadock(t *testing.T) {
	const (
		srcBind = "./src:/var/www/html"
		roMask  = "- ./src/.madock:/var/www/html/.madock:ro"
	)

	err := fs.WalkDir(FS, "snippets/docker-compose", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		body, readErr := fs.ReadFile(FS, path)
		if readErr != nil {
			return readErr
		}
		s := string(body)

		if !strings.Contains(s, srcBind) {
			return nil
		}
		if !strings.Contains(s, roMask) {
			t.Errorf("%s mounts the code at /var/www/html but does not mask .madock read-only.\n"+
				"Add this line to its volumes, or a process in the container can write .madock/docker "+
				"and the host's next start will run it:\n      %s", path, roMask)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
