package credentials

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs"
)

func TestASnapshotGivesBackThePasswordsItWasTakenWith(t *testing.T) {
	execDir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", execDir)
	configs.CleanCache()
	if err := os.MkdirAll(filepath.Join(execDir, "projects", "shop"), 0o755); err != nil {
		t.Fatal(err)
	}

	snapshot := t.TempDir()
	Record(snapshot, map[string]string{
		"db/root_password":  "root-then",
		"db/password":       "user-then",
		"rabbitmq/password": "broker-then",
		"php/version":       "8.4",
	})

	// Since the snapshot, the root and broker passwords changed; the user's did not.
	now := map[string]string{
		"db/root_password":  "root-now",
		"db/password":       "user-then",
		"rabbitmq/password": "broker-now",
	}
	changed := Apply("shop", "default", snapshot, now)
	if len(changed) != 2 || changed[0] != "db/root_password" || changed[1] != "rabbitmq/password" {
		t.Fatalf("changed %v, want the root and broker passwords only", changed)
	}

	written := configs.ParseXmlFile(filepath.Join(execDir, "projects", "shop", "config.xml"))
	for key, want := range map[string]string{
		"scopes/default/db/root_password":  "root-then",
		"scopes/default/rabbitmq/password": "broker-then",
	} {
		if written[key] != want {
			t.Errorf("%s = %q, want %q", key, written[key], want)
		}
	}
	if _, ok := written["scopes/default/db/password"]; ok {
		t.Error("a password the snapshot agrees with was written anyway")
	}

	recorded, _ := Recorded(snapshot)
	if _, ok := recorded["php/version"]; ok {
		t.Error("the record holds a key that is not a password")
	}
}

func TestASnapshotWithoutARecordChangesNothing(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())
	configs.CleanCache()
	if changed := Apply("shop", "default", t.TempDir(), map[string]string{"db/password": "x"}); changed != nil {
		t.Fatalf("changed %v from a snapshot that recorded nothing", changed)
	}
}
