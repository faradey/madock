package project

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// TestGrafanaWithoutRedisGetsNoRedisDatasource pins what a project with
// monitoring but no Redis is given.
//
// The datasource, its dashboard and two Redis plugins were provisioned
// unconditionally, while `redis/enabled` defaults to false. Measured on a
// Magento project in the VM: Grafana came up with a "Redis" datasource whose
// health endpoint answered `400 Bad Request`, pointing at `redis://redisdb:6379`
// — a host that does not exist, because no such container is rendered. A
// dashboard full of empty panels next to it reads as a broken installation
// rather than as a service nobody switched on.
func TestGrafanaWithoutRedisGetsNoRedisDatasource(t *testing.T) {
	env := testenv.SetupWith(t, "grafananoredis", "grafananoredis.test", map[string]string{
		"grafana/enabled":  "true",
		"redis/enabled":    "false",
		"rabbitmq/enabled": "false",
	})

	MakeConf(env.ProjectName)
	compose := readCompose(t, env)

	if strings.Contains(compose, "redis-datasource") {
		t.Errorf("a project with no redis was given the Redis datasource or its plugin:\n%s", grafanaBlock(compose))
	}
	if strings.Contains(compose, "dashboard-redis.json") {
		t.Errorf("a project with no redis was given the Redis dashboard")
	}
	if strings.Contains(compose, "dashboard-rabbitmq.json") {
		t.Errorf("a project with no rabbitmq was given the RabbitMQ dashboard")
	}

	// The half that must not move: monitoring itself is still provisioned.
	for _, wanted := range []string{"dashboard-mysql.json", "dashboard-loki.json", "name: Loki", "name: Prometheus"} {
		if !strings.Contains(compose, wanted) {
			t.Errorf("%s is missing — the gate removed more than the redis parts", wanted)
		}
	}
}

// TestGrafanaWithRedisGetsTheDatasource is the other side of the same gate: a
// project that does run Redis keeps everything it had.
func TestGrafanaWithRedisGetsTheDatasource(t *testing.T) {
	env := testenv.SetupWith(t, "grafanaredis", "grafanaredis.test", map[string]string{
		"grafana/enabled": "true",
		"redis/enabled":   "true",
	})

	MakeConf(env.ProjectName)
	compose := readCompose(t, env)

	for _, wanted := range []string{"name: Redis", "redis://redisdb:6379", "dashboard-redis.json", "redis-datasource"} {
		if !strings.Contains(compose, wanted) {
			t.Errorf("a project with redis lost %s:\n%s", wanted, grafanaBlock(compose))
		}
	}
}

// TestGrafanaPluginsCarryVersions is the fix for a downgrade nobody reported.
//
// `GF_INSTALL_PLUGINS` listed plugin ids with no versions, and `redis-app`
// installs its own copy of `redis-datasource`: the container log shows 2.2.0
// downloaded and then 2.1.1 written over it, in one start. Versions come from
// the configuration now, so a plugin can also be moved without a new madock.
func TestGrafanaPluginsCarryVersions(t *testing.T) {
	env := testenv.SetupWith(t, "grafanaplugins", "grafanaplugins.test", map[string]string{
		"grafana/enabled": "true",
		"redis/enabled":   "true",
	})

	MakeConf(env.ProjectName)
	compose := readCompose(t, env)

	line := ""
	for _, candidate := range strings.Split(compose, "\n") {
		if strings.Contains(candidate, "GF_INSTALL_PLUGINS") {
			line = candidate
			break
		}
	}
	if line == "" {
		t.Fatal("no GF_INSTALL_PLUGINS line was rendered at all")
	}

	for _, plugin := range strings.Split(strings.Trim(strings.SplitN(line, ":", 2)[1], " \""), ",") {
		if len(strings.Fields(plugin)) != 2 {
			t.Errorf("plugin %q carries no version, so its dependency can replace it with an older build: %s", plugin, line)
		}
	}
}

// grafanaBlock trims the compose file to the grafana service, so a failure
// prints what is worth reading rather than the whole file.
func grafanaBlock(compose string) string {
	start := strings.Index(compose, "  grafana:")
	if start == -1 {
		return "no grafana service was rendered"
	}

	rest := compose[start:]
	if end := strings.Index(rest, "\n  loki:"); end != -1 {
		return rest[:end]
	}

	return rest
}
