package configs

import "testing"

func TestApplyDerivedNodeMajorVersion(t *testing.T) {
	cases := map[string]string{
		"20.16.0": "20",
		"22":      "22",
		"18.19.1": "18",
	}

	for version, want := range cases {
		conf := map[string]string{"nodejs/version": version}
		applyDerived(conf)
		if got := conf["nodejs/major_version"]; got != want {
			t.Errorf("nodejs/version %q → major_version %q, want %q", version, got, want)
		}
	}
}

// The value is recomputed on read, so a stale copy left in config.xml by an
// older madock — or set by hand — must not survive into the generated
// Dockerfile.
func TestApplyDerivedOverwritesAStoredValue(t *testing.T) {
	conf := map[string]string{
		"nodejs/version":       "22.11.0",
		"nodejs/major_version": "18",
	}
	applyDerived(conf)

	if got := conf["nodejs/major_version"]; got != "22" {
		t.Errorf("major_version = %q, want 22 derived from the version", got)
	}
}

// Without a source there is nothing to derive, and inventing "0" would put
// `setup_0.x` into the NodeSource URL.
func TestApplyDerivedWithoutASource(t *testing.T) {
	conf := map[string]string{"nodejs/version": ""}
	applyDerived(conf)

	if _, ok := conf["nodejs/major_version"]; ok {
		t.Error("major_version was invented from an empty nodejs/version")
	}
}

func TestIsDerived(t *testing.T) {
	if source, ok := IsDerived("nodejs/major_version"); !ok || source != "nodejs/version" {
		t.Errorf("IsDerived(nodejs/major_version) = %q, %v; want nodejs/version, true", source, ok)
	}
	if _, ok := IsDerived("nodejs/version"); ok {
		t.Error("nodejs/version reported as derived — it is the source")
	}
}
