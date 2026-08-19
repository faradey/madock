package docker

import (
	"fmt"
	"testing"
)

// Named entries and a repeated <job> tag are both documented spellings, and
// after the parser fix they arrive as cron/jobs/<name> and cron/jobs/job/<n>.
// Both have to reach the crontab.
func TestGetCronJobsFromConfigReadsBothSpellings(t *testing.T) {
	conf := map[string]string{
		"platform":            "custom",
		"cron/enabled":        "true",
		"cron/jobs/scheduler": "* * * * * /scheduler.sh",
		"cron/jobs/job/0":     "* * * * * /a.sh",
		"cron/jobs/job/1":     "* * * * * /b.sh",
		"cron/jobs/empty":     "",
	}

	jobs := getCronJobsFromConfig(conf)
	if len(jobs) != 3 {
		t.Fatalf("got %d jobs, want 3: %v", len(jobs), jobs)
	}
}

// Plain string order reads the tenth repeated job as the second one. Cron does
// not care, but a crontab that reshuffles itself between runs is a diff nobody
// can read and a bug report nobody can reproduce.
func TestGetCronJobsFromConfigOrdersIndexNumerically(t *testing.T) {
	conf := map[string]string{}
	for i := 0; i < 12; i++ {
		conf[fmt.Sprintf("cron/jobs/job/%d", i)] = fmt.Sprintf("* * * * * /%d.sh", i)
	}

	jobs := getCronJobsFromConfig(conf)
	if len(jobs) != 12 {
		t.Fatalf("got %d jobs, want 12", len(jobs))
	}
	for i, job := range jobs {
		if want := fmt.Sprintf("* * * * * /%d.sh", i); job != want {
			t.Errorf("position %d = %q, want %q", i, job, want)
		}
	}
}

func TestPlatformInstallsOwnCron(t *testing.T) {
	for platform, want := range map[string]bool{
		"magento2":   true,
		"shopify":    true,
		"shopware":   true,
		"custom":     false,
		"prestashop": false,
		"":           false,
	} {
		if got := platformInstallsOwnCron(platform); got != want {
			t.Errorf("%q: got %v, want %v", platform, got, want)
		}
	}
}
