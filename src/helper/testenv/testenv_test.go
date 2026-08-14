package testenv

import (
	"strings"
	"testing"
)

// TestNormalise pins the thing the golden files depend on and nothing else
// checks: that two different machines normalise to the same text.
//
// Both cases here are real. macOS gives a gid of 20, which the old
// value-substitution normaliser found inside unrelated numbers; a GitHub runner
// gives uid == gid, where replacing the uid first left the gid with nothing to
// match. Either one produced golden files that only the machine that wrote them
// could reproduce.
func TestNormalise(t *testing.T) {
	env := &Env{ExecDir: "/exec", RunDir: "/run"}

	// The same rendered file as three machines would write it: an arm64 laptop
	// (uid 501, gid 20), an amd64 runner (uid 1001, gid 1001), and a Linux
	// desktop (uid 1000, gid 1000).
	render := func(uid, gid, arch string) string {
		return strings.Join([]string{
			`    user: "` + uid + `:` + gid + `"`,
			`      - "34500:80"`,
			`RUN usermod -u ` + uid + ` -o www-data && groupmod -g ` + gid + ` -o www-data`,
			`RUN mkdir /var/www/.npm && chown ` + uid + `:` + gid + ` /var/www/.npm`,
			`    && chown -R ` + uid + `:` + gid + ` /var/www`,
			`    && curl -o ioncube.tar.gz http://x/ioncube_loaders_lin_` + arch + `.tar.gz \\`,
			`RUN sed -i 's/session.cookie_lifetime = 0/session.cookie_lifetime = 2592000/g' php.ini`,
			`        "uid": "PBFA97CFB590B2093"`,
			`  ingestion_burst_size_mb: 20`,
		}, "\n")
	}

	mac := Normalise(render("501", "20", "aarch64"), env)
	ci := Normalise(render("1001", "1001", "x86-64"), env)
	linux := Normalise(render("1000", "1000", "x86-64"), env)

	if mac != ci {
		t.Errorf("macOS and CI normalise differently:\n%s\n---\n%s", mac, ci)
	}
	if ci != linux {
		t.Errorf("two Linux machines normalise differently:\n%s\n---\n%s", ci, linux)
	}

	// The numbers that are not ids must survive, and the ids must not.
	for _, want := range []string{
		`user: "<UID>:<GID>"`,
		`- "<PORT>:80"`,
		"usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data",
		"chown <UID>:<GID> /var/www/.npm",
		"chown -R <UID>:<GID> /var/www",
		"ioncube_loaders_lin_<ARCH>.tar.gz",
		"session.cookie_lifetime = 2592000",
		`"uid": "PBFA97CFB590B2093"`,
		"ingestion_burst_size_mb: 20",
	} {
		if !strings.Contains(mac, want) {
			t.Errorf("normalised output is missing %q:\n%s", want, mac)
		}
	}
}
