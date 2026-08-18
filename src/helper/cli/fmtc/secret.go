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

// SecretOrValue prints the value when the caller has explicitly asked for it,
// and describes it otherwise. The flag exists because reading a password out of
// the tool is a legitimate thing to want — copying it into a database client —
// and the answer to that is one visible word on the command line, not printing
// it to everybody by default.
func SecretOrValue(value string, show bool) string {
	if show {
		return value
	}
	return Secret(value)
}
