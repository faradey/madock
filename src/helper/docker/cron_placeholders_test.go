package docker

import (
	"strings"
	"testing"
)

func TestExpandCronJobSubstitutesWorkdir(t *testing.T) {
	conf := map[string]string{"workdir": "/var/www/html/current"}

	got, unresolved := expandCronJob("* * * * * {{workdir}}/scripts/poke.sh >> /var/www/html/logs/cron.log 2>&1", conf)

	if len(unresolved) != 0 {
		t.Fatalf("unresolved: %v", unresolved)
	}
	if want := "* * * * * /var/www/html/current/scripts/poke.sh >> /var/www/html/logs/cron.log 2>&1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The same line has to be right on a machine with no deployer, which is the
// reason the placeholder exists at all.
func TestExpandCronJobFollowsTheMachine(t *testing.T) {
	job := "* * * * * {{workdir}}/scripts/poke.sh"

	deployed, _ := expandCronJob(job, map[string]string{"workdir": "/var/www/html/current"})
	plain, _ := expandCronJob(job, map[string]string{"workdir": "/var/www/html"})

	if deployed == plain {
		t.Fatalf("both machines produced %q", deployed)
	}
	if !strings.HasSuffix(deployed, "/current/scripts/poke.sh") || !strings.HasSuffix(plain, "/html/scripts/poke.sh") {
		t.Errorf("deployed=%q plain=%q", deployed, plain)
	}
}

func TestExpandCronJobToleratesSpacesInThePlaceholder(t *testing.T) {
	got, unresolved := expandCronJob("* * * * * {{ workdir }}/x.sh", map[string]string{"workdir": "/app"})
	if len(unresolved) != 0 || got != "* * * * * /app/x.sh" {
		t.Errorf("got %q, unresolved %v", got, unresolved)
	}
}

// A crontab is a file, and a job is not the place to put a password.
func TestExpandCronJobRefusesSecrets(t *testing.T) {
	conf := map[string]string{"workdir": "/app", "db/password": "hunter2"}

	got, unresolved := expandCronJob("* * * * * mysql -p{{db/password}}", conf)

	if len(unresolved) != 1 {
		t.Fatalf("expected one refusal, got %v", unresolved)
	}
	if !strings.Contains(unresolved[0], "secret") {
		t.Errorf("refusal does not say why: %q", unresolved[0])
	}
	if strings.Contains(got, "hunter2") {
		t.Errorf("secret was substituted: %q", got)
	}
}

func TestExpandCronJobRefusesUnknownAndEmpty(t *testing.T) {
	_, unknown := expandCronJob("* * * * * {{public_dir}}/x.sh", map[string]string{})
	if len(unknown) != 1 || !strings.Contains(unknown[0], "not a placeholder") {
		t.Errorf("unknown placeholder: %v", unknown)
	}

	_, empty := expandCronJob("* * * * * {{workdir}}/x.sh", map[string]string{"workdir": "  "})
	if len(empty) != 1 || !strings.Contains(empty[0], "empty") {
		t.Errorf("empty value: %v", empty)
	}
}

// A line still carrying a placeholder would run every minute and fail every
// minute, into /dev/null. It is dropped and named instead.
func TestResolveCronJobsDropsWhatItCannotResolve(t *testing.T) {
	conf := map[string]string{"workdir": "/app"}
	jobs := []string{
		"* * * * * {{workdir}}/good.sh",
		"* * * * * {{nonsense}}/bad.sh",
	}

	resolved, refusals := resolveCronJobs(jobs, conf)

	if len(resolved) != 1 || resolved[0] != "* * * * * /app/good.sh" {
		t.Errorf("resolved: %v", resolved)
	}
	if len(refusals) != 1 || !strings.Contains(refusals[0], "bad.sh") {
		t.Errorf("refusals: %v", refusals)
	}
}

// A job with no placeholder is left exactly as written.
func TestResolveCronJobsLeavesPlainJobsAlone(t *testing.T) {
	jobs := []string{"* * * * * /usr/bin/php /var/www/html/bin/magento cron:run"}

	resolved, refusals := resolveCronJobs(jobs, map[string]string{"workdir": "/app"})

	if len(refusals) != 0 || len(resolved) != 1 || resolved[0] != jobs[0] {
		t.Errorf("resolved=%v refusals=%v", resolved, refusals)
	}
}
