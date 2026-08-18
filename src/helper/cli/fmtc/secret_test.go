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

func TestSecretOrValueOnlyRevealsWhenAsked(t *testing.T) {
	const password = "e2e-plaintext-password"

	if got := SecretOrValue(password, false); got == password {
		t.Error("the value was printed without being asked for")
	}
	if got := SecretOrValue(password, true); got != password {
		t.Errorf("--show-secrets did not print the value: %q", got)
	}
}
