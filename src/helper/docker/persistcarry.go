package docker

import (
	"fmt"
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
// container's anonymous volume into its named volume. It reports what it did
// and never stops the caller: a failure is said and the start goes on, which
// is what would have happened without this.
func CarryToPersisted(projectName string, conf map[string]string) {
	for _, s := range persistedServices {
		if conf[s.key+"/enabled"] != "true" || conf[s.key+"/persist"] != "true" {
			continue
		}
		if err := carryOne(projectName, conf, s); err != nil {
			fmtc.WarningLn(fmt.Sprintf("%s: the data of the old container was not carried to the named volume: %v", s.service, err))
		}
	}
}

func carryOne(projectName string, conf map[string]string, s persistedService) error {
	named := strings.ToLower(composeProjectName(projectName)) + "_" + s.volume
	if _, err := dockerOut("volume", "inspect", named); err == nil {
		return nil // already there: carried before, or persisted from the start
	}
	container := GetContainerName(conf, projectName, s.service)
	source, err := dockerOut("inspect", "--format", `{{range .Mounts}}{{if eq .Destination "`+s.path+`"}}{{.Type}} {{.Name}}{{end}}{{end}}`, container)
	if err != nil || source == "" {
		return nil // no old container, or nothing mounted there: a fresh start
	}
	parts := strings.Fields(source)
	if len(parts) != 2 || parts[0] != "volume" {
		return nil // a bind, not something compose would leave behind
	}
	if s.key == "rabbitmq" {
		fmtc.WarningLn("rabbitmq: rabbitmq/persist starts the broker on a new volume; the old one's messages are not carried (they live under the old node name). Drain the queues before this start if they matter.")
		return nil
	}
	image, err := dockerOut("inspect", "--format", "{{.Config.Image}}", container)
	if err != nil {
		return err
	}
	fmtc.TitleLn(fmt.Sprintf("Carrying %s's data into the named volume %s...", s.service, s.volume))
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
	if _, err := dockerOut("run", "--rm", "--user", "0",
		"-v", parts[1]+":/from:ro", "-v", named+":/to",
		"--entrypoint", "sh", image, "-c", "cp -a /from/. /to/"); err != nil {
		// An empty named volume is worse than none: the next start would
		// adopt it and begin empty, which is what this exists to prevent.
		_, _ = dockerOut("volume", "rm", named)
		return fmt.Errorf("copying %s into %s: %w", parts[1], named, err)
	}
	fmtc.SuccessLn(fmt.Sprintf("%s: data carried from %s (the old volume is kept)", s.service, shortVolume(parts[1])))
	return nil
}

func shortVolume(name string) string {
	if len(name) > 12 {
		return name[:12]
	}
	return name
}
