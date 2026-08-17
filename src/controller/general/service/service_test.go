package service

import "testing"

// Most services are a container or an extension, and their switch is
// <name>/enabled. Embedded node is neither — it is a runtime added to whichever
// application image the project has — so its switch is the key itself.
//
// Until 3.9.8 the same thing was reachable as `service:enable php/nodejs`, and
// that worked by accident: it was in no map, and IsService found it only
// because php/nodejs/enabled happened to exist. This pins the replacement so
// the ergonomics survive on purpose.
func TestConfigKeyOf(t *testing.T) {
	for _, c := range []struct {
		name string
		want string
	}{
		{"nodejs/embedded", "nodejs/embedded"},
		{"embedded-node", "nodejs/embedded"},
		{"xdebug", "php/xdebug/enabled"},
		{"php/xdebug", "php/xdebug/enabled"},
		{"yarn", "nodejs/yarn/enabled"},
		{"db", "db/enabled"},
	} {
		if got := ConfigKeyOf(c.name); got != c.want {
			t.Errorf("ConfigKeyOf(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

// A short name has to survive the round trip, or `service:enable embedded-node`
// writes a key nothing reads.
func TestShortNameRoundTrip(t *testing.T) {
	if got := GetByShort("embedded-node"); got != "nodejs/embedded" {
		t.Errorf("GetByShort(\"embedded-node\") = %q", got)
	}
	if got := GetByLong("nodejs/embedded"); got != "embedded-node" {
		t.Errorf("GetByLong(\"nodejs/embedded\") = %q", got)
	}
}
