package docker

import (
	"errors"
	"strings"
	"testing"
)

// fakeDocker answers inspect calls from a table and records everything else.
type fakeDocker struct {
	volumes map[string]bool   // named volumes that exist
	mounts  map[string]string // container -> "volume <name>" at the data path
	failRun bool
	calls   []string
}

func (f *fakeDocker) run(args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch {
	case args[0] == "volume" && args[1] == "inspect":
		if f.volumes[args[2]] {
			return "[]", nil
		}
		return "", errors.New("no such volume")
	case args[0] == "inspect" && args[2] == "{{.Id}}":
		if _, ok := f.mounts[args[3]]; ok {
			return "id", nil
		}
		return "", errors.New("no such container")
	case args[0] == "inspect" && strings.Contains(args[2], ".Mounts"):
		if m, ok := f.mounts[args[3]]; ok {
			return m, nil
		}
		return "", errors.New("no such container")
	case args[0] == "inspect":
		return "madock_shop-redisdb:latest", nil
	case args[0] == "run" && f.failRun:
		return "", errors.New("cp failed")
	}
	return "", nil
}

func (f *fakeDocker) did(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func withFake(t *testing.T, f *fakeDocker) {
	old := dockerOut
	dockerOut = f.run
	t.Cleanup(func() { dockerOut = old })
}

var redisOn = map[string]string{"redis/enabled": "true", "redis/persist": "true", "container_name_prefix": "madock_"}

// The case it exists for: an old container with its data in an anonymous
// volume. It is stopped, the named volume is made with compose's labels, and
// the data is copied in — nothing is removed.
func TestTheOldAnonymousVolumeIsCopiedIntoTheNamedOne(t *testing.T) {
	f := &fakeDocker{mounts: map[string]string{"madock_shop-redisdb-1": "volume 0123456789abcdef"}}
	withFake(t, f)
	if err := CarryToPersisted("shop", redisOn); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"stop madock_shop-redisdb-1",
		"volume create --label com.docker.compose.project=madock_shop --label com.docker.compose.volume=redisdata madock_shop_redisdata",
		"run --rm --user 0 -v 0123456789abcdef:/from:ro -v madock_shop_redisdata:/to",
	} {
		if !f.did(want) {
			t.Errorf("no %q in\n%s", want, strings.Join(f.calls, "\n"))
		}
	}
	if f.did("volume rm") {
		t.Error("a volume was removed")
	}
}

// Already persisted, or carried before: nothing is touched.
func TestAnExistingNamedVolumeIsLeftAlone(t *testing.T) {
	f := &fakeDocker{volumes: map[string]bool{"madock_shop_redisdata": true}, mounts: map[string]string{"madock_shop-redisdb-1": "volume 0123456789abcdef"}}
	withFake(t, f)
	CarryToPersisted("shop", redisOn)
	if f.did("stop") || f.did("run") || f.did("volume create") {
		t.Errorf("acted on a project already persisted:\n%s", strings.Join(f.calls, "\n"))
	}
}

// A copy that fails takes the empty named volume back out: left there, the
// next start would adopt it and begin empty.
func TestAFailedCopyLeavesNoEmptyVolume(t *testing.T) {
	f := &fakeDocker{failRun: true, mounts: map[string]string{"madock_shop-redisdb-1": "volume 0123456789abcdef"}}
	withFake(t, f)
	CarryToPersisted("shop", redisOn)
	if !f.did("volume rm madock_shop_redisdata") {
		t.Errorf("the empty volume stayed:\n%s", strings.Join(f.calls, "\n"))
	}
}

// RabbitMQ is said, not copied: its files live under the old node name.
func TestRabbitMQIsNotCopied(t *testing.T) {
	f := &fakeDocker{mounts: map[string]string{"madock_shop-rabbitmq-1": "volume 0123456789abcdef"}}
	withFake(t, f)
	CarryToPersisted("shop", map[string]string{"rabbitmq/enabled": "true", "rabbitmq/persist": "true", "container_name_prefix": "madock_"})
	if f.did("run") || f.did("stop") {
		t.Errorf("RabbitMQ was copied:\n%s", strings.Join(f.calls, "\n"))
	}
}

// Persist off, or a bind mount: nothing happens.
func TestNothingHappensWithoutPersistOrWithABind(t *testing.T) {
	f := &fakeDocker{mounts: map[string]string{"madock_shop-redisdb-1": "bind "}}
	withFake(t, f)
	CarryToPersisted("shop", map[string]string{"redis/enabled": "true", "redis/persist": "false"})
	CarryToPersisted("shop", redisOn)
	if f.did("stop") || f.did("run") {
		t.Errorf("acted:\n%s", strings.Join(f.calls, "\n"))
	}
}

// The case that lost a production store's sessions: an image with no VOLUME
// for the data path keeps the data in the container's own layer. It is
// streamed out of the container, and a failure stops the start rather than
// letting the recreate take the data.
func TestDataInTheContainersLayerIsCopiedOrTheStartStops(t *testing.T) {
	valkeyOn := map[string]string{"valkey/enabled": "true", "valkey/persist": "true", "container_name_prefix": "madock_"}
	f := &fakeDocker{mounts: map[string]string{"madock_shop-valkeydb-1": ""}}
	withFake(t, f)
	var copied []string
	old := copyFromLayer
	t.Cleanup(func() { copyFromLayer = old })
	copyFromLayer = func(container, path, named, image string) error {
		copied = append(copied, container+" "+path+" "+named)
		return nil
	}
	if err := CarryToPersisted("shop", valkeyOn); err != nil {
		t.Fatal(err)
	}
	if strings.Join(copied, ",") != "madock_shop-valkeydb-1 /data madock_shop_valkeydata" {
		t.Errorf("the layer was not copied: %v\n%s", copied, strings.Join(f.calls, "\n"))
	}

	f2 := &fakeDocker{mounts: map[string]string{"madock_shop-valkeydb-1": ""}}
	withFake(t, f2)
	copyFromLayer = func(string, string, string, string) error { return errors.New("docker cp failed") }
	err := CarryToPersisted("shop", valkeyOn)
	if err == nil || !strings.Contains(err.Error(), "Nothing was recreated") {
		t.Errorf("a failed copy from the layer did not stop the start: %v", err)
	}
	if !f2.did("volume rm madock_shop_valkeydata") {
		t.Error("the empty volume stayed after a failed copy")
	}
}

// No old container is a fresh start: nothing to carry, nothing stopped.
func TestNoOldContainerIsAFreshStart(t *testing.T) {
	f := &fakeDocker{mounts: map[string]string{}}
	withFake(t, f)
	if err := CarryToPersisted("shop", redisOn); err != nil {
		t.Fatal(err)
	}
	if f.did("stop") || f.did("volume create") {
		t.Errorf("acted without an old container:\n%s", strings.Join(f.calls, "\n"))
	}
}
