package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// These run the php image's entrypoint against a fake crontab, because the
// question it answers is behavioural and a rendering test cannot reach it.
//
// What it exists for: nothing in an application image starts cron — the CMD is
// php-fpm — so the daemon lived only as long as the container it was started
// in. Anything that recreates that container takes cron with it and leaves the
// crontab behind: the jobs are there, nothing reads them, nothing fails, and
// every HTTP check stays green. madock re-arms after `service:restart`, and that
// cannot cover a host reboot, where Docker brings the containers back on its own
// and madock is not running at all. Measured on a production machine on
// 2026-08-30: containers up, applications answering, cron down in both projects
// that had it.
//
// The three cases below are the whole contract, and the middle one is the reason
// this is a test rather than a line of shell nobody checks: a Debian crontab is
// never empty — it carries an explanatory header — so "the file has content"
// would start cron for a project that has no jobs at all, and the daemon would
// then be running everywhere, including where cron:disable was used.
func TestPhpEntrypointStartsCronOnlyWhenThereAreJobs(t *testing.T) {
	t.Run("jobs installed: cron is started, then the command runs", func(t *testing.T) {
		out := runPhpEntrypoint(t, "0 * * * * /usr/bin/php /var/www/html/bin/magento cron:run\n")

		if !strings.Contains(out, "service cron start") {
			t.Errorf("cron was not started while jobs were installed:\n%s", out)
		}
		if !strings.Contains(out, "COMMAND RAN") {
			t.Errorf("the entrypoint did not exec the command it was given:\n%s", out)
		}
	})

	t.Run("only Debian's header: cron stays down", func(t *testing.T) {
		// Exactly what `crontab -l` prints for a user who has never had a job.
		out := runPhpEntrypoint(t, "# Edit this file to introduce tasks to be run by cron.\n#\n# m h  dom mon dow   command\n")

		if strings.Contains(out, "service cron start") {
			t.Errorf("cron was started for a crontab holding nothing but comments:\n%s", out)
		}
		if !strings.Contains(out, "COMMAND RAN") {
			t.Errorf("the entrypoint did not exec the command it was given:\n%s", out)
		}
	})

	t.Run("no crontab at all: the command still runs", func(t *testing.T) {
		// `crontab -l` exits 1 and prints to stderr here. The application must
		// not be held hostage by that: a container refusing to serve because its
		// scheduler is unhappy is worse than the silence being fixed.
		out := runPhpEntrypoint(t, "")

		if strings.Contains(out, "service cron start") {
			t.Errorf("cron was started with no crontab present:\n%s", out)
		}
		if !strings.Contains(out, "COMMAND RAN") {
			t.Errorf("a missing crontab stopped the application from starting:\n%s", out)
		}
	})
}

// runPhpEntrypoint renders the php Dockerfile, takes the entrypoint out of it
// and runs it with `crontab` and `service` replaced by scripts that report what
// they were asked to do.
//
// The entrypoint is read out of the rendered file rather than copied here: a
// copy would keep passing after the real one changed, which is the failure this
// whole file is about.
func runPhpEntrypoint(t *testing.T, crontabBody string) string {
	t.Helper()

	projectName := "phpentry"
	installation := testenv.SetupWith(t, projectName, "phpentry.test", map[string]string{
		"platform":    "magento2",
		"php/enabled": "true",
	})

	MakeConf(projectName)

	rendered := testenv.Collect(t, filepath.Join(installation.ExecDir, "aruntime", "projects", projectName), installation)

	var dockerfile string
	for name, body := range rendered {
		if strings.HasSuffix(name, "ctx/php.Dockerfile") {
			dockerfile = body
		}
	}
	if dockerfile == "" {
		t.Fatal("MakeConf produced no php.Dockerfile")
	}

	// entrypointBody cuts at the marker, and the marker sits mid-line — the
	// `RUN cat > … <<'MADOCK_EOF' && chmod +x …` tail comes with it. Dropping
	// everything up to the first newline is what leaves a script sh can read.
	body := entrypointBody(t, dockerfile)
	if _, rest, found := strings.Cut(body, "\n"); found {
		body = rest
	}

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "madock-entrypoint")
	writeExecutable(t, scriptPath, body)

	bin := filepath.Join(dir, "bin")

	// `crontab -u <user> -l` prints the body for any user, or fails when there
	// is none — the empty case is a missing crontab, not an empty one.
	if crontabBody == "" {
		writeExecutable(t, filepath.Join(bin, "crontab"), "#!/bin/sh\necho 'no crontab for user' >&2\nexit 1\n")
	} else {
		writeExecutable(t, filepath.Join(bin, "crontab"), "#!/bin/sh\ncat <<'CRONTAB_EOF'\n"+crontabBody+"CRONTAB_EOF\n")
	}

	// Records to a file rather than to a stream. The entrypoint sends this
	// command's output to /dev/null deliberately — a scheduler that complains
	// must not appear in the application's log — so stdout and stderr are both
	// unavailable to the test, and the first version of this stub silently
	// reported nothing.
	marker := filepath.Join(dir, "service.calls")
	writeExecutable(t, filepath.Join(bin, "service"), "#!/bin/sh\necho \"service $*\" >> "+marker+"\n")

	cmd := exec.Command("sh", scriptPath, "sh", "-c", "echo COMMAND RAN")
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the entrypoint failed: %v\n%s", err, out)
	}

	calls, _ := os.ReadFile(marker)

	return string(out) + string(calls)
}
