package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// A command that does work — copies a volume, starts containers, untars into
// a helper — prints as it goes, and so do the docker and tar processes it
// starts. A JSON document on the same stream as that progress cannot be
// parsed. Begin diverts stdout to stderr for the rest of the run and keeps the
// real stdout for the one document written at the end; a subprocess takes
// os.Stdout when it is started, so its output goes the same way.
//
// The contract for every command that uses it:
//
//   - stdout carries one Response: {"success": true, "data": …} or
//     {"success": false, "error": "…"}
//   - the exit status agrees with "success"
//   - a failure that ends the process from somewhere the command cannot see
//     (logger.Fatal deep in a helper) leaves stdout empty and exits non-zero,
//     with the reason on stderr. An empty stdout is therefore never a success.

// Requested reports whether the arguments after the command name ask for JSON:
// exact tokens, and nothing after a `--`, which belongs to whatever the command
// passes it to.
//
// One copy for madock's dispatcher and for madock-pro's commands, which read
// os.Args by hand. Two copies of "what counts as --json" decide independently
// whether stdout is diverted, and they had already drifted apart — one counted
// from the command name, the other from the first argument.
func Requested(argv []string) bool {
	for _, arg := range argv {
		if arg == "--" {
			return false
		}
		if arg == "--json" || arg == "-j" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}

// Writer holds the real stdout while progress goes to stderr.
type Writer struct {
	out io.Writer
}

// Begin diverts stdout to stderr for the rest of the process and returns the
// writer for the one document. Stdout is not restored: the document is the
// last thing the command prints.
func Begin() *Writer {
	w := &Writer{out: os.Stdout}
	os.Stdout = os.Stderr
	return w
}

// Print writes a success document.
func (w *Writer) Print(data any) {
	w.write(Response{Success: true, Data: data})
}

// Fail writes a failure document and exits 1.
func (w *Writer) Fail(format string, a ...any) {
	w.FailWith(nil, format, a...)
}

// FailWith writes a failure document that still carries what was done — a
// prune that removed four environments of five — and exits 1.
func (w *Writer) FailWith(data any, format string, a ...any) {
	w.write(Response{Success: false, Data: data, Error: strings.TrimSpace(fmt.Sprintf(format, a...))})
	exit(1)
}

// PrintRaw writes a success document whose data is already JSON, copied from
// r as it is: db:execute's rows, which may be a table dump and are not worth
// decoding only to encode them again. r must hold one JSON value.
func (w *Writer) PrintRaw(r io.Reader) error {
	if _, err := io.WriteString(w.out, `{"success": true, "data": `); err != nil {
		return err
	}
	if _, err := io.Copy(w.out, r); err != nil {
		return err
	}
	_, err := io.WriteString(w.out, "}\n")
	return err
}

// exit is replaceable so a test can see a failure without ending the run.
var exit = os.Exit

func (w *Writer) write(response Response) {
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not encode the JSON answer: %v\n", err)
		exit(1)
		return
	}
	fmt.Fprintln(w.out, string(data))
}
