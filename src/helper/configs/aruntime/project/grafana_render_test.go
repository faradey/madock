package project

import (
	"os"
	"path/filepath"
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

	plugins := grafanaPlugins(t, env)
	if len(plugins) == 0 {
		t.Fatal("the Grafana image installs no plugins at all")
	}
	for _, plugin := range plugins {
		if len(strings.Fields(plugin)) != 2 {
			t.Errorf("plugin %q carries no version, so its dependency can replace it with an older build: %v", plugin, plugins)
		}
	}
}

// TestGrafanaPluginsAreBuiltIntoTheImage is why the plugins left the start.
//
// GF_INSTALL_PLUGINS made Grafana fetch them from grafana.com on every start.
// Three of four full end-to-end runs of madock Pro came up without the Redis
// datasource — "plugin not registered", and once "failed to download plugin
// archive ... i/o timeout" — while the same test alone passed every time. A
// server restarted with grafana.com slow would be the same. The image installs
// them now, and the start asks nothing of the network.
func TestGrafanaPluginsAreBuiltIntoTheImage(t *testing.T) {
	for _, redis := range []string{"true", "false"} {
		env := testenv.SetupWith(t, "grafanabuilt"+redis, "grafanabuilt"+redis+".test", map[string]string{
			"grafana/enabled": "true",
			"redis/enabled":   redis,
		})
		MakeConf(env.ProjectName)
		block := grafanaBlock(readCompose(t, env))

		if strings.Contains(block, "GF_INSTALL_PLUGINS") {
			t.Errorf("redis %s: the container still installs plugins on start:\n%s", redis, block)
		}
		if !strings.Contains(block, "dockerfile: grafana.Dockerfile") {
			t.Errorf("redis %s: Grafana is not built from its Dockerfile:\n%s", redis, block)
		}

		plugins := strings.Join(grafanaPlugins(t, env), ",")
		if !strings.Contains(plugins, "grafana-clock-panel") {
			t.Errorf("redis %s: the image lost the plugins every project has: %s", redis, plugins)
		}
		if got := strings.Contains(plugins, "redis-datasource"); got != (redis == "true") {
			t.Errorf("redis %s: redis-datasource in the image is %v: %s", redis, got, plugins)
		}
	}
}

// TestGrafanaDatasourcesCarryNoVersion is why a changed datasource stopped
// reaching Grafana.
//
// Every provisioned datasource said `version: 1`. Grafana stores a version of
// its own, and once it had written a datasource twice it held 2 and took no
// more from a file saying 1: measured in the VM, a Redis password set after
// that left the datasource with no secure fields, NOAUTH on every query, and a
// restart changed nothing. With the line gone the same container took the
// password at once. It stayed hidden while the plugins were downloaded on
// start, because the first start came up without the plugin and the datasource
// was written once fewer.
func TestGrafanaDatasourcesCarryNoVersion(t *testing.T) {
	env := testenv.SetupWith(t, "grafanaversion", "grafanaversion.test", map[string]string{
		"grafana/enabled": "true",
		"redis/enabled":   "true",
	})
	MakeConf(env.ProjectName)
	block := grafanaBlock(readCompose(t, env))

	if !strings.Contains(block, "datasources:") {
		t.Fatalf("no datasources rendered:\n%s", block)
	}
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == "version: 1" || strings.HasPrefix(strings.TrimSpace(line), "version: ") {
			t.Errorf("a provisioned datasource pins its version, and Grafana stops taking changes past it: %q", line)
		}
	}
}

// grafanaPlugins reads the plugin list the Grafana Dockerfile installs.
func grafanaPlugins(t *testing.T, env *testenv.Env) []string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName, "ctx", "grafana.Dockerfile"))
	if err != nil {
		t.Fatalf("no Grafana Dockerfile was rendered: %v", err)
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `echo "`) {
			continue
		}
		list := strings.SplitN(strings.TrimPrefix(line, `echo "`), `"`, 2)[0]
		var plugins []string
		for _, p := range strings.Split(list, ",") {
			if strings.TrimSpace(p) != "" {
				plugins = append(plugins, strings.TrimSpace(p))
			}
		}
		return plugins
	}
	t.Fatalf("the Grafana Dockerfile names no plugin list:\n%s", body)
	return nil
}

// TestPromtailWatchesWhatTheConfigSays is the reason the scrape config stopped
// being a fixed list.
//
// It held four jobs, all of them Magento's own log files, and the only directory
// promtail could see was the application's `var/log`. A project that is not
// Magento collected nothing at all, and Loki answered an empty list of labels
// while looking healthy — measured in the VM, `label/job/values` came back empty
// on a freshly started project with monitoring on.
//
// The `all` job is asserted by name because the shipped Loki dashboard queries
// `{job="all"}`: renaming it empties the dashboard, and nothing else would say
// so.
func TestPromtailWatchesWhatTheConfigSays(t *testing.T) {
	env := testenv.SetupWith(t, "grafanalogs", "grafanalogs.test", map[string]string{
		"grafana/enabled":                "true",
		"grafana/logs/paths/queueworker": "/var/log/queue/*.log",
	})

	MakeConf(env.ProjectName)

	rendered, err := os.ReadFile(filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName, "ctx", "grafana", "promtail-config.yml"))
	if err != nil {
		t.Fatalf("no promtail config was rendered: %v", err)
	}
	config := string(rendered)

	for _, wanted := range []string{
		"job_name: queueworker",
		"__path__: /var/log/queue/*.log",
		"job: all",
		"__path__: /var/log/madock/*.log",
	} {
		if !strings.Contains(config, wanted) {
			t.Errorf("%q is missing from the scrape config:\n%s", wanted, config)
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
