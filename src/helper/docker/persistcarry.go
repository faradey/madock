package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
)

// Turning `redis/persist` or `valkey/persist` on gives the service a named
// volume. Until now the next start began it empty: the data stayed in the
// anonymous volume the old container had, which compose does not carry over
// and nothing else points at. On a server that is real data — a Packeton keeps
// its download statistics in Redis, an application its queued jobs.
//
// CarryToPersisted copies it across once, before the start or rebuild that
// would create the new volume: the old container is stopped (Redis and Valkey
// write their dump on shutdown), the named volume is created with compose's
// own labels so compose adopts it, and the old volume's files are copied in
// with the service's own image. Nothing is removed — the old volume stays,
// dangling, as the way back.
//
// RabbitMQ is not carried: its data lives under the node name, and the old
// container's node name was its random hostname, so copied files would be
// ignored by a broker now called rabbit@rabbitmq. It is said instead.

// persistedService is one service a persist key moves to a named volume.
type persistedService struct {
	key, service, volume, path string
}

var persistedServices = []persistedService{
	{"redis", "redisdb", "redisdata", "/data"},
	{"valkey", "valkeydb", "valkeydata", "/data"},
	{"rabbitmq", "rabbitmq", "rabbitmqdata", "/var/lib/rabbitmq"},
}

// dockerOut runs docker and returns its trimmed stdout; replaced in tests.
var dockerOut = func(args ...string) (string, error) {
	out, err := exec.Command("docker", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// CarryToPersisted copies each newly persisted service's data from the old
// container into its named volume: from its anonymous volume, or — when the
// image declares no volume for the data path — from the container's own layer.
//
// An error stops the caller before anything is recreated. Going on would
// recreate the container and lose the data, which is what this exists to
// prevent: on 2026-10-20 a Valkey whose image declares no VOLUME kept 10 724
// sessions in its layer, the first version of this found no anonymous volume,
// took it for a fresh start, and compose's recreate took the sessions with the
// old container.
func CarryToPersisted(projectName string, conf map[string]string) error {
	for _, s := range persistedServices {
		if conf[s.key+"/enabled"] != "true" || conf[s.key+"/persist"] != "true" {
			continue
		}
		if err := carryOne(projectName, conf, s); err != nil {
			return fmt.Errorf("%s: the old container's data could not be carried to the named volume %s, and recreating it now would lose that data: %w. Nothing was recreated; fix it, or set %s/persist to false to keep the old container", s.service, s.volume, err, s.key)
		}
	}
	return nil
}

func carryOne(projectName string, conf map[string]string, s persistedService) error {
	named := strings.ToLower(composeProjectName(projectName)) + "_" + s.volume
	if _, err := dockerOut("volume", "inspect", named); err == nil {
		return nil // already there: carried before, or persisted from the start
	}
	container := GetContainerName(conf, projectName, s.service)
	if _, err := dockerOut("inspect", "--format", "{{.Id}}", container); err != nil {
		return nil // no old container: a fresh start, nothing to carry
	}
	source, err := dockerOut("inspect", "--format", `{{range .Mounts}}{{if eq .Destination "`+s.path+`"}}{{.Type}} {{.Name}}{{end}}{{end}}`, container)
	if err != nil {
		return fmt.Errorf("reading %s's mounts: %w", container, err)
	}
	parts := strings.Fields(source)
	fromLayer := len(parts) == 0
	switch {
	case fromLayer:
		// The image declares no volume for the path: the data is in the
		// container's own layer, and a recreate removes it with the container.
	case parts[0] == "bind":
		return nil // the host's directory: it survives the recreate as it is
	case parts[0] != "volume" || len(parts) != 2:
		return fmt.Errorf("unexpected mount at %s: %q", s.path, source)
	}
	if s.key == "rabbitmq" {
		fmtc.WarningLn("rabbitmq: rabbitmq/persist starts the broker on a new volume; the old one's messages are not carried (they live under the old node name). Drain the queues before this start if they matter.")
		return nil
	}
	image, err := dockerOut("inspect", "--format", "{{.Config.Image}}", container)
	if err != nil {
		return err
	}
	from := "the container's own layer"
	if !fromLayer {
		from = shortVolume(parts[1])
	}
	fmtc.TitleLn(fmt.Sprintf("Carrying %s's data from %s into the named volume %s...", s.service, from, s.volume))
	// A dump first, for a server whose configuration does not save on
	// shutdown; without a password set this is the one that counts, and with
	// one the shutdown's own save is.
	_, _ = dockerOut("exec", container, "sh", "-c", "(redis-cli SAVE || valkey-cli SAVE) >/dev/null 2>&1; true")
	if _, err := dockerOut("stop", container); err != nil {
		return fmt.Errorf("stopping %s: %w", container, err)
	}
	if _, err := dockerOut("volume", "create",
		"--label", "com.docker.compose.project="+strings.ToLower(composeProjectName(projectName)),
		"--label", "com.docker.compose.volume="+s.volume,
		named); err != nil {
		return fmt.Errorf("creating %s: %w", named, err)
	}
	if fromLayer {
		err = copyFromLayer(container, s.path, named, image)
	} else {
		_, err = dockerOut("run", "--rm", "--user", "0",
			"-v", parts[1]+":/from:ro", "-v", named+":/to",
			"--entrypoint", "sh", image, "-c", "cp -a /from/. /to/")
	}
	if err != nil {
		// An empty named volume is worse than none: the next start would
		// adopt it and begin empty, which is what this exists to prevent.
		_, _ = dockerOut("volume", "rm", named)
		return fmt.Errorf("copying into %s: %w", named, err)
	}
	kept := "the old volume is kept"
	if fromLayer {
		kept = "the copy is now the only one: the old container goes with this start"
	}
	fmtc.SuccessLn(fmt.Sprintf("%s: data carried from %s (%s)", s.service, from, kept))
	return nil
}

// copyFromLayer streams path out of the (stopped) container and into the
// named volume. `docker cp` reads a stopped container's layer; its archive
// names the directory itself, which the extraction strips.
var copyFromLayer = func(container, path, named, image string) error {
	out := exec.Command("docker", "cp", container+":"+path, "-")
	in := exec.Command("docker", "run", "-i", "--rm", "--user", "0", "-v", named+":/to",
		"--entrypoint", "sh", image, "-c", "tar -x -C /to --strip-components=1")
	// os.Pipe rather than StdoutPipe: Wait on the reading side's pipe closes
	// it when the writer exits, which can cut off what the reader has not yet
	// taken. Each end is closed here only once its process is done with it.
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	out.Stdout, in.Stdin = w, r
	var outErr, inErr strings.Builder
	out.Stderr, in.Stderr = &outErr, &inErr
	if err := in.Start(); err != nil {
		w.Close()
		r.Close()
		return err
	}
	runErr := out.Run()
	w.Close()
	waitErr := in.Wait()
	r.Close()
	if runErr != nil {
		return fmt.Errorf("docker cp %s:%s: %v %s", container, path, runErr, strings.TrimSpace(outErr.String()))
	}
	if waitErr != nil {
		return fmt.Errorf("unpacking into %s: %v %s", named, waitErr, strings.TrimSpace(inErr.String()))
	}
	return nil
}

func shortVolume(name string) string {
	if len(name) > 12 {
		return name[:12]
	}
	return name
}
