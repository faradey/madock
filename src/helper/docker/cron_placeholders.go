package docker

import (
	"regexp"
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
)

// A cron job in the config is written once and installed on machines whose
// paths differ. The application root is `/var/www/html` on a plain checkout and
// `/var/www/html/current` where deployer manages releases, so a job that spells
// the path out is correct on one kind of machine and wrong on the other —
// silently, because cron sends its output nowhere. The committed config then
// has to be edited per machine, which is how it stops being one file.
//
// madock's own jobs never had this problem: the magento2 and shopware branches
// build their command from projectConf["workdir"]. This gives the config's jobs
// the same thing rather than a new idea.
//
//	<apply_due>* * * * * {{workdir}}/scripts/cron/poke.sh /api/cron/apply-due</apply_due>
//
// Expanded when the crontab is written, not read back through a variable, so
// `crontab -l` shows the real path — which is what somebody reads at three in
// the morning.
var cronPlaceholder = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_/]+)\s*\}\}`)

// cronPlaceholderKeys are the config keys a job may name. An allow list rather
// than "any config key": a job is installed into a crontab, and `{{db/password}}`
// would put a password there — readable by anything that can read the crontab,
// and copied into every backup of it. Secret keys are refused below as well, so
// that widening this list cannot open that door by accident.
var cronPlaceholderKeys = map[string]bool{
	"workdir": true,
}

// expandCronJob substitutes the placeholders a job names, and reports the ones
// it could not.
//
// An unresolved placeholder is not left in place: a crontab line reading
// `cd {{workdir}} && …` runs every minute, fails every minute, and says so
// nowhere. The caller drops the job and names it instead.
func expandCronJob(job string, conf map[string]string) (string, []string) {
	var unresolved []string

	expanded := cronPlaceholder.ReplaceAllStringFunc(job, func(match string) string {
		name := strings.TrimSpace(cronPlaceholder.FindStringSubmatch(match)[1])

		switch {
		case configs2.SecretKeys[name]:
			// Refused even if somebody adds it to the allow list: the value ends
			// up in a file, and this is not the place to decide it is fine.
			unresolved = append(unresolved, name+" (a secret cannot be written into a crontab)")
		case !cronPlaceholderKeys[name]:
			unresolved = append(unresolved, name+" (not a placeholder madock knows)")
		case strings.TrimSpace(conf[name]) == "":
			unresolved = append(unresolved, name+" (empty in this project's config)")
		default:
			return conf[name]
		}
		return match
	})

	return expanded, unresolved
}
