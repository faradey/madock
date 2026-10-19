package configs

import "testing"

// A muted clone has the source's database: its cron would run the source's
// jobs and its mail() would reach the source's customers. Whatever the stored
// values say, the assembled configuration says cron off and mail to mailpit.
func TestAMutedCloneRunsNoCronAndSendsNoMail(t *testing.T) {
	conf := map[string]string{
		"clone/muted":          "true",
		"cron/enabled":         "true",
		"php/sendmail/enabled": "false",
		"php/sendmail/host":    "relay.example.com",
		"php/sendmail/port":    "25",
	}
	applyDerived(conf)
	for key, want := range map[string]string{
		"cron/enabled":         "false",
		"php/sendmail/enabled": "true",
		"php/sendmail/host":    "host.docker.internal",
		"php/sendmail/port":    "1025",
	} {
		if conf[key] != want {
			t.Errorf("muted: %s is %q, want %q", key, conf[key], want)
		}
	}
}

func TestAnUnmutedProjectKeepsItsSettings(t *testing.T) {
	conf := map[string]string{"clone/muted": "false", "cron/enabled": "true", "php/sendmail/host": "relay.example.com"}
	applyDerived(conf)
	if conf["cron/enabled"] != "true" || conf["php/sendmail/host"] != "relay.example.com" {
		t.Errorf("an unmuted project's settings were changed: %v", conf)
	}
}
