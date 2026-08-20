package fmtc

import "fmt"

// Secret describes a secret instead of printing it.
//
// The distinction that matters here is radius, not secrecy: the value is in a
// config file on the same machine either way, but a file is read on purpose
// while the output of a command goes wherever the output goes — terminal
// scrollback, a CI log, a screenshot, an issue, a chat with somebody helping.
// `db:info` is run to find a host and a port, and it was printing the shared
// database's **root** password alongside them, which is a key to every other
// project's schema on that server.
//
// The length is kept because it answers the question people actually ask of
// this line — whether the value is there, and whether it is the short one they
// typed by hand or the generated one. Masking that reveals the first and last
// character was the previous behaviour of `madock info`; it gives away two
// characters of a password in exchange for nothing.
func Secret(value string) string {
	if value == "" {
		return "not set"
	}
	return fmt.Sprintf("set (%d)", len([]rune(value)))
}

// HideSecretsByDefault decides what happens when the caller has *not* asked for
// the value, and the open-source edition answers "print it".
//
// The two editions are used on different machines, and that is the whole of the
// difference. madock is a developer's local environment manager: the password it
// prints belongs to a container on that developer's own laptop, `db:info` is run
// to copy it into a database client, and describing it instead adds a flag to
// every such use for no gain — the file it comes from is two directories away.
// madock-pro runs on servers, where the same command prints a shared database's
// **root** password, the output goes into a ticket or a screen share, and one of
// those passwords is a key to every other project's schema on that host.
//
// So the layer that knows which machine this is decides: madock-pro sets this in
// its init, madock leaves it alone. Not a config option, because a setting can be
// wrong on a server and nothing would say so; the edition cannot be.
var HideSecretsByDefault = false

// SecretOrValue prints the value when the caller has explicitly asked for it,
// and otherwise defers to the edition. The flag exists because reading a
// password out of the tool is a legitimate thing to want — copying it into a
// database client — and where the value is withheld by default, the answer to
// that is one visible word on the command line.
func SecretOrValue(value string, show bool) string {
	if show || !HideSecretsByDefault {
		return value
	}
	return Secret(value)
}
