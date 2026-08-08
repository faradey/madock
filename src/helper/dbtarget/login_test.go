package dbtarget

import "testing"

// The database commands used to authenticate as root unconditionally. That is
// right for a project's own server, where the client runs inside the container
// and root is reachable on localhost, and wrong the moment the database belongs
// to somebody else: the client then connects over the network from another
// container, and MySQL grants root to localhost only. The refusal comes before
// any password is checked — "Host '172.21.0.2' is not allowed to connect" —
// which reads like a network problem and is an account problem.
//
// Found by the end-to-end suite the first time a consumer tried to write to a
// shared database, which is the whole point of the feature.
func TestLoginIsRootOnlyForAProjectsOwnDatabase(t *testing.T) {
	own := Target{
		User:         "app",
		Password:     "app-password",
		RootPassword: "root-password",
	}
	user, password := own.Login()
	if user != "root" || password != "root-password" {
		t.Errorf("a project's own database should be reached as root, got %q/%q", user, password)
	}

	shared := Target{
		User:         "consumer",
		Password:     "consumer-password",
		RootPassword: "the-provider-root-password",
		Shared:       true,
	}
	user, password = shared.Login()
	if user != "consumer" || password != "consumer-password" {
		t.Errorf("a shared database should be reached as the consumer's own account, got %q/%q", user, password)
	}

	// Said explicitly because the field is populated for a shared target too —
	// it is the provider's root password, and using it is exactly the defect.
	if password == shared.RootPassword {
		t.Error("a consumer must not authenticate with the provider's root password")
	}
}
