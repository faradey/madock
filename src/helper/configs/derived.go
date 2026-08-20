package configs

import (
	"sort"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// derivedKeys lists config values that are computed from another value rather
// than configured. A derived key stored in config.xml can only go stale, and a
// hand-set one can only be silently discarded — which is what happened with
// nodejs/major_version: twelve platform presets wrote it at setup, two
// generators recomputed it while rendering, and `config:set nodejs/major_version`
// looked accepted and changed nothing. The map points at the key that decides.
var derivedKeys = map[string]string{
	"nodejs/major_version": "nodejs/version",
}

// IsDerived reports whether an option is computed, and names the key that
// governs it.
func IsDerived(name string) (string, bool) {
	source, ok := derivedKeys[name]
	return source, ok
}

// DerivedFrom lists the computed keys that follow one source, in a stable order.
func DerivedFrom(source string) []string {
	var derived []string
	for key, from := range derivedKeys {
		if from == source {
			derived = append(derived, key)
		}
	}
	sort.Strings(derived)
	return derived
}

// RemoveStoredDerived deletes the keys computed from source out of one config
// file, and reports which ones it removed.
//
// The value is recomputed on every read, so a copy stored in a file can only go
// stale — and a stale copy of a derived key is read by people even when nothing
// reads it. Measured on a live server: `config:set nodejs/version 22.22.0` left
// `major_version 20` sitting in the file, and the next person to open it
// concluded the environment would build Node 20. It would not — the render
// derives 22 — but nothing in the file says so, and the conclusion cost an
// evening. So writing the source now takes the stored derivative with it.
//
// Only the file given is touched. A derived key in a project's own committed
// .madock/config.xml is not madock's to delete, which is why the caller reads
// the result back and says so instead.
func RemoveStoredDerived(file, source, activeScope string) ([]string, error) {
	if !paths.IsFileExist(file) {
		return nil, nil
	}

	stored := ParseXmlFile(file)
	var present []string
	for _, key := range DerivedFrom(source) {
		if _, ok := stored["scopes/"+activeScope+"/"+key]; ok {
			present = append(present, key)
		}
	}
	if len(present) == 0 {
		return nil, nil
	}

	if err := RemoveKeepingComments(file, present, activeScope); err != nil {
		return nil, err
	}
	return present, nil
}

// applyDerived fills the computed keys from their sources. It runs on every
// assembled project config, so no caller has to remember to do it and no
// generator can render a stale value.
func applyDerived(conf map[string]string) {
	if conf == nil {
		return
	}

	// The NodeSource install script is addressed by major version only
	// (setup_20.x), while the image tag is the full one.
	if version := strings.TrimSpace(conf["nodejs/version"]); version != "" {
		conf["nodejs/major_version"] = strings.Split(version, ".")[0]
	} else {
		// No source, so any value present is left over from an older config.
		// Dropping it leaves the placeholder unsubstituted, which fails the
		// image build loudly; keeping it would install some Node nobody asked
		// for and look deliberate.
		delete(conf, "nodejs/major_version")
	}

	// Where the application lives is answered here, once, rather than stored.
	//
	// With deployer managing releases the root is `<mount>/current`, and until
	// now that was a value `deploy:enable` wrote into the config — a copy of a
	// derivation, made at one moment, kept in one file. Every failure this
	// caused was the same failure: something read a copy that was stale,
	// missing, or written out by hand.
	//
	// **Three of them in one day, 2026-08-19.** A cron job with the path spelled
	// out, correct until the project moved to deployer. A systemd unit on the
	// host naming `current/artisan`, which pointed at a deploy root with no
	// artisan in it. And a failed deploy that removed the config file holding
	// the copy, so a rebuild read an older one and installed a scheduler that
	// ran the wrong command for seventeen minutes on a live application.
	//
	// Derived, none of those are possible: there is no copy to go stale, and
	// the answer survives the file that used to hold it being absent — which is
	// exactly the window that broke. A consumer asks `workdir` and gets the
	// truth whether or not deployer is configured, which is the whole point.
	// One key, deliberately. `deploy:enable` also used to write `php/workdir`,
	// and nothing has ever read it: it is declared in no config, rendered by no
	// template and consulted by no command. A second spelling of the same fact
	// is how the two get to disagree, so it is not derived here either — the
	// write is gone from `deploy:enable` instead.
	if strings.TrimSpace(conf["deploy/enabled"]) == "true" {
		conf["workdir"] = deployRelativeRoot(conf["workdir"])
	}

	// msmtp will not send without an envelope sender, and there is no msmtprc in
	// the image to hold a default. A mail transport supplies one; a bare mail()
	// call with four arguments does not, and fails with exit 78. Rendering the
	// whole argument here rather than the address keeps the template free of a
	// conditional it cannot express — an unset address has to produce nothing at
	// all, not an empty --from=.
	if from := strings.TrimSpace(conf["php/sendmail/from"]); from != "" {
		conf["php/sendmail/from_argument"] = " --from=" + from
	} else {
		conf["php/sendmail/from_argument"] = ""
	}
}

// deployRelativeRoot puts the active release between the mount point and the
// application, and says the same thing twice without saying it twice.
//
// Idempotent on purpose. Installations that ran `deploy:enable` before this was
// derived have `/var/www/html/current` sitting in their config, and every read
// of it must produce the same path rather than `current/current`. The same
// property makes the derivation safe to apply to a value a person set by hand.
func deployRelativeRoot(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = defaultWorkdir
	}
	if strings.HasSuffix(base, "/"+deployReleaseLink) {
		return base
	}
	return base + "/" + deployReleaseLink
}

const (
	// defaultWorkdir is where a project lives when nothing says otherwise. The
	// same literal used to be repeated in a dozen fallbacks.
	defaultWorkdir = "/var/www/html"
	// deployReleaseLink is deployer's symlink to the release now serving.
	deployReleaseLink = "current"
)

// IsDeployManaged reports whether a project's source tree belongs to deployer.
//
// It is the question to ask before writing anything into that tree wholesale.
// On an ordinary project the mount point and the application are the same
// directory, so `/var/www/html` means both and no caller ever had to choose.
// Under deployer they are not: the mount holds `releases/`, `shared/`,
// `current` and deployer's own bookkeeping, and the application is one level
// down. Code that keeps meaning "the application" while writing to "the mount"
// is the shape that produced three separate failures on 2026-08-19.
func IsDeployManaged(conf map[string]string) bool {
	return strings.TrimSpace(conf["deploy/enabled"]) == "true"
}
