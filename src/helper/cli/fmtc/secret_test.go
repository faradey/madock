package fmtc

import "testing"

func TestSecretDescribesWithoutRevealing(t *testing.T) {
	const password = "e2e-plaintext-password"

	got := Secret(password)
	if got == password {
		t.Fatal("the value came back as itself")
	}
	if got != "set (22)" {
		t.Fatalf("Secret() = %q, want %q", got, "set (22)")
	}

	// Counted in runes, not bytes: a length reported in bytes would leak that
	// the value is not ASCII, and would be wrong for the person checking it
	// against what they typed.
	if got := Secret("пароль"); got != "set (6)" {
		t.Errorf("Secret() = %q, want %q", got, "set (6)")
	}

	// Absent and present are different answers, and "set (0)" is neither.
	if got := Secret(""); got != "not set" {
		t.Errorf("Secret(\"\") = %q, want %q", got, "not set")
	}
}

// The open-source edition prints the password. It manages a developer's own
// laptop, `db:info` is run to copy the value into a database client, and the
// file it comes from is two directories away — withholding it there adds a flag
// to every use and protects nothing.
func TestSecretOrValuePrintsByDefaultInTheOpenSourceEdition(t *testing.T) {
	const password = "e2e-plaintext-password"

	if HideSecretsByDefault {
		t.Fatal("the open-source edition must not withhold secrets; the layer that runs on servers sets this")
	}
	if got := SecretOrValue(password, false); got != password {
		t.Errorf("the value was withheld without any edition asking for that: %q", got)
	}
	if got := SecretOrValue(password, true); got != password {
		t.Errorf("--show-secrets did not print the value: %q", got)
	}
}

// And the switch the paid edition sets in its init, tested here because that is
// where the behaviour lives — madock-pro registers it, it does not reimplement
// it.
func TestSecretOrValueOnlyRevealsWhenAskedOnceHidingIsOn(t *testing.T) {
	const password = "e2e-plaintext-password"

	HideSecretsByDefault = true
	t.Cleanup(func() { HideSecretsByDefault = false })

	if got := SecretOrValue(password, false); got == password {
		t.Error("the value was printed without being asked for")
	}
	if got := SecretOrValue(password, true); got != password {
		t.Errorf("--show-secrets did not print the value: %q", got)
	}
}
