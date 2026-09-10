package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// readGenerated reads one file out of the project's generated runtime.
func readGenerated(t *testing.T, env *testenv.Env, relative string) string {
	t.Helper()

	path := filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName, relative)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}

// The php limits reach the vhost, which is the half the golden fixtures cannot
// show: they render the defaults, and the defaults are the numbers that were
// hardcoded before. What had to be proved is that a project can change them.
//
// They were three copies of one number inside the platform vhosts, and the only
// way to change them was to copy a 240-line vhost into the project's own
// .madock/docker and let it drift from upstream. What runs into them is an
// import or a reindex through the web, and what that looks like is a 500 with
// nothing in the nginx log to explain it.
func TestPhpLimitsReachTheVhost(t *testing.T) {
	env := testenv.SetupWith(t, "limitsproject", "limits.test", map[string]string{
		"php/limits/memory":                 "4096M",
		"php/limits/max_execution_time":     "7200",
		"php/limits/max_execution_time_web": "900",
	})

	MakeConf("limitsproject")

	vhost := readGenerated(t, env, "ctx/nginx.conf")
	for _, want := range []string{
		"memory_limit=4096M",
		"max_execution_time=7200",
		"max_execution_time=900",
	} {
		if !strings.Contains(vhost, want) {
			t.Errorf("missing %q in the generated vhost:\n%s", want, firstLines(vhost, 60))
		}
	}

	if strings.Contains(vhost, "memory_limit=756M") {
		t.Errorf("the vhost still carries the old hardcoded memory limit:\n%s", firstLines(vhost, 60))
	}
}

// The body size a project accepts is its own setting now. Two limits sit in the
// path — this one and proxy/max_body_size in the shared proxy — and an upload
// that dies between them says nothing useful, which is why they are named the
// same thing in both places.
func TestProjectBodySizeReachesTheVhost(t *testing.T) {
	env := testenv.SetupWith(t, "bodysizeproject", "bodysize.test", map[string]string{
		"nginx/max_body_size": "512M",
	})

	MakeConf("bodysizeproject")

	vhost := readGenerated(t, env, "ctx/nginx.conf")
	if !strings.Contains(vhost, "client_max_body_size       512M;") {
		t.Errorf("the configured body size did not reach the vhost:\n%s", firstLines(vhost, 60))
	}
}

func firstLines(text string, n int) string {
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// The search engine's memory is the project's to set, and both numbers are
// needed rather than one.
//
// They were `memory: 2512m` and `-Xms800m -Xmx800m` inside the compose
// snippets, so every project paid the same regardless of catalogue size.
// Measured on a live Magento store on 2026-09-09: java held 1650 MB of RSS against
// 12.4 MB of indices and 21 products, on a machine with 5.8 GB. The only way
// to change it was to copy the snippet into the project's own .madock/docker,
// which is a copy that then drifts from the shipped one in silence — and that
// copy existed on that machine when this was written.
//
// Two keys, because the container limit and the JVM heap are not the same
// number and cannot be derived from each other safely: the heap has to leave
// room for lucene's off-heap memory, and a limit below the heap makes the
// kernel kill the container instead of java throwing.
func TestSearchMemoryReachesTheComposeFile(t *testing.T) {
	for _, engine := range []string{"opensearch", "elasticsearch"} {
		other := "elasticsearch"
		if engine == "elasticsearch" {
			other = "opensearch"
		}

		t.Run(engine, func(t *testing.T) {
			// The other engine is switched off explicitly: this platform ships
			// both blocks, so leaving it on renders a second service carrying
			// the defaults, and the assertion below — that the old numbers are
			// gone — would fail against the neighbour rather than the code.
			env := testenv.SetupWith(t, "search"+engine, engine+".test", map[string]string{
				"search/engine":                      engine,
				"search/" + engine + "/enabled":      "true",
				"search/" + engine + "/memory_limit": "768m",
				"search/" + engine + "/heap":         "256m",
				"search/" + other + "/enabled":       "false",
			})

			MakeConf("search" + engine)

			compose := readGenerated(t, env, "docker-compose.yml")
			for _, want := range []string{"memory: 768m", "-Xms256m -Xmx256m"} {
				if !strings.Contains(compose, want) {
					t.Errorf("%s: missing %q in the generated compose file:\n%s",
						engine, want, firstLines(compose, 80))
				}
			}

			// The numbers that used to be hardcoded, named so that a template
			// which ignores the setting fails here rather than passing on a
			// substring that happens to match.
			for _, gone := range []string{"memory: 2512m", "-Xms800m -Xmx800m"} {
				if strings.Contains(compose, gone) {
					t.Errorf("%s: the compose file still carries the hardcoded %q", engine, gone)
				}
			}
		})
	}
}

// The logs have to outlive the container, and the generated files are the first
// half of that claim.
//
// Measured on production during an intrusion review on 2026-09-09: three deploys
// recreated the containers and the HTTP records for the minute under
// investigation were gone from disk and from `madock logs`. Docker keeps a
// container's stdout inside the container's own directory, so a rebuild deletes
// it — the services have to write files on a mounted path as well.
func TestTheServicesWriteLogsWhereARebuildCannotReach(t *testing.T) {
	env := testenv.SetupWith(t, "logsproject", "logs.test", nil)

	MakeConf("logsproject")

	compose := readGenerated(t, env, "docker-compose.yml")
	if strings.Count(compose, "logsdata:/var/log/madock") < 2 {
		t.Errorf("the log volume is not mounted into both the web server and the database:\n%s",
			firstLines(compose, 80))
	}
	// Declared as well as mounted, or compose refuses the file outright.
	if !strings.Contains(compose, "\n  logsdata:") {
		t.Errorf("the log volume is mounted and never declared:\n%s", firstLines(compose, 80))
	}

	// The directives live in their own file, mounted into conf.d, which nginx
	// includes inside the http block. They were in the vhost until 2026-09-10
	// and that failed in the one place it mattered: a project may ship its own
	// `nginx/conf/default.conf` in `.madock/docker/`, which replaces madock's
	// template wholesale — the volume was mounted, the database wrote its slow
	// log, and nginx wrote nothing at all.
	logsConf := readGenerated(t, env, "ctx/madock-logs.conf")
	for _, want := range []string{
		"access_log /var/log/madock/nginx-access.log;",
		// And to the stream as well, or `madock logs` goes quiet — the two are
		// not alternatives.
		"access_log /dev/stdout;",
		"error_log  /var/log/madock/nginx-error.log warn;",
	} {
		if !strings.Contains(logsConf, want) {
			t.Errorf("missing %q in the generated logging fragment:\n%s", want, logsConf)
		}
	}
	if !strings.Contains(compose, "madock-logs.conf:/etc/nginx/conf.d/00-madock-logs.conf") {
		t.Errorf("the logging fragment is generated and never mounted:\n%s", firstLines(compose, 60))
	}

	// The slow query log, which was never configured at all: log_error,
	// general_log and log_slow_query were all commented out in the shipped
	// config, so no madock project has ever recorded a slow query anywhere.
	//
	// `log_error` stays unset, and that is asserted below rather than left to
	// chance: MariaDB writes its error log to a file or to stderr, never both,
	// and pointing it at the file empties the stream `madock logs -s db` reads.
	// The first version of this change did exactly that, and CI answered with
	// "the database never wrote anything recognisable to its log" plus a project
	// whose database never became ready at all.
	mycnf := readGenerated(t, env, "ctx/my.cnf")
	if strings.Contains(mycnf, "log_error =") {
		t.Errorf("log_error was pointed at a file — `madock logs -s db` goes silent:\n%s", firstLines(mycnf, 40))
	}
	for _, want := range []string{
		"slow_query_log = 1",
		"slow_query_log_file = /var/log/madock/mysql-slow.log",
	} {
		if !strings.Contains(mycnf, want) {
			t.Errorf("missing %q in the generated my.cnf:\n%s", want, firstLines(mycnf, 40))
		}
	}

	// A named volume rather than a directory under aruntime, and the reason is
	// worth keeping: the bind mount put root-owned files into the project's
	// runtime directory, and `project:remove` — which runs as the person, not as
	// root — then could not delete the project at all. CI found it as a database
	// that would not start, because the leftover volume of the undeleted project
	// still carried the old root password.
	if strings.Contains(compose, "./logs:") {
		t.Error("the logs went back to a bind mount, which project:remove cannot clean up")
	}
}

// Off means off: a machine that says so gets what it had before, with no mount
// and no directives.
func TestLogPersistenceCanBeTurnedOff(t *testing.T) {
	env := testenv.SetupWith(t, "logsoffproject", "logsoff.test", map[string]string{
		"logs/persist/enabled": "false",
	})

	MakeConf("logsoffproject")

	if compose := readGenerated(t, env, "docker-compose.yml"); strings.Contains(compose, "/var/log/madock") {
		t.Error("the log mount was rendered for a project that turned it off")
	}
	if vhost := readGenerated(t, env, "ctx/nginx.conf"); strings.Contains(vhost, "/var/log/madock") {
		t.Error("the log directives were rendered for a project that turned it off")
	}
}

// nginx has to start after every PHP container its vhost names, and the second
// one was missing.
//
// nginx resolves upstream hostnames when it loads its configuration, not on the
// first request, so a container that is not yet in docker's DNS is not a slow
// start — it is `[emerg] host not found in upstream "php_without_xdebug:9000"`
// and an nginx that exits 1 and stays down. Reported from outside as issue #150
// on 2026-09-10, with compose output showing nginx up at 0.2s and
// php_without_xdebug at 0.8s.
func TestNginxWaitsForBothPhpContainers(t *testing.T) {
	env := testenv.SetupWith(t, "xdebugorder", "xdebugorder.test", map[string]string{
		"php/enabled":        "true",
		"php/xdebug/enabled": "true",
	})

	MakeConf("xdebugorder")

	compose := readGenerated(t, env, "docker-compose.yml")
	if !strings.Contains(compose, "php_without_xdebug") {
		t.Fatalf("the second php container is not in this stack at all:\n%s", firstLines(compose, 60))
	}

	depends := composeSection(compose, "  nginx:")
	if !strings.Contains(depends, "- php_without_xdebug") {
		t.Errorf("nginx does not wait for php_without_xdebug:\n%s", depends)
	}
	if !strings.Contains(depends, "- php") {
		t.Errorf("nginx does not wait for php:\n%s", depends)
	}
}

// And it must not name a service that is not there: compose refuses the whole
// file over a dependency on an undefined service, which would take down every
// project that runs without xdebug — that is, most of them.
func TestNginxDoesNotWaitForAContainerThatDoesNotExist(t *testing.T) {
	env := testenv.SetupWith(t, "noxdebugorder", "noxdebugorder.test", map[string]string{
		"php/enabled":        "true",
		"php/xdebug/enabled": "false",
	})

	MakeConf("noxdebugorder")

	compose := readGenerated(t, env, "docker-compose.yml")
	if strings.Contains(compose, "- php_without_xdebug") {
		t.Errorf("nginx depends on a container this stack never renders:\n%s", firstLines(compose, 60))
	}
}

// composeSection returns one service's block, from its key to the next service
// at the same indentation.
func composeSection(compose, key string) string {
	start := strings.Index(compose, key)
	if start < 0 {
		return ""
	}
	rest := compose[start+len(key):]
	for offset := 0; offset < len(rest); offset++ {
		if rest[offset] != '\n' {
			continue
		}
		line := rest[offset+1:]
		if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' {
			return compose[start : start+len(key)+offset]
		}
	}

	return compose[start:]
}
