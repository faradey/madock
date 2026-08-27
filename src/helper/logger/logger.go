package logger

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var (
	customWriter io.Writer
	customPath   string
)

// SetWriter overrides the debug log destination.
// When set, all debug output goes to w instead of the default file.
func SetWriter(w io.Writer) {
	customWriter = w
}

// SetLogPath overrides the debug log file path.
// By default, debug.log is written next to the executable.
func SetLogPath(path string) {
	customPath = path
}

func Fatal(v ...any) {
	debugger(v...)
	log.Fatal(v...)
}

// FatalChild ends the command with the exit code of the program madock ran,
// rather than with 1.
//
// The code is the answer, not decoration. `madock cli bash -c "exit 137"` exited
// 1, and so did `exit 3` and a failing test suite — so a script could tell that
// something went wrong and nothing about what: 137 is the OOM killer and means
// "give it more memory", 1 from a test runner means "fix the code", and the two
// were indistinguishable. Measured while building a Shopware administration
// bundle, which Vite ends by being killed.
//
// madock had the number the whole time. `exec.Cmd.Run` returns an *exec.ExitError
// carrying it, the pass-through commands hand that straight to the logger, and
// the debug log even prints it ("exit status 137") — it just never reached the
// caller, because log.Fatal exits 1 unconditionally.
//
// Anything that is not a child's failure still exits 1: docker refusing to start,
// a container that does not exist, a path that cannot be read. Those are madock
// failing, and inventing a code for them would make the number mean two things.
func FatalChild(err error) {
	debugger(err)
	log.Print(err)
	os.Exit(childExitCode(err))
}

// childExitCode picks the code to end with. Separated from FatalChild because
// the exit itself cannot be tested in-process, and the decision is the part
// worth pinning.
func childExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// -1 is "killed by a signal, no exit status" — a code Unix has no room
		// for, and passing it on would exit 255 for reasons unrelated to the
		// child. 1 is the honest answer there.
		if code := exitErr.ExitCode(); code > 0 {
			return code
		}
	}

	return 1
}

func Fatalln(v ...any) {
	debugger(v...)
	log.Fatalln(v...)
}

func Println(v ...any) {
	debugger(v...)
	log.Println(v...)
}

func debugger(v ...any) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := "\n[" + timestamp + "] " + fmt.Sprintln(v...) + string(debug.Stack())

	if customWriter != nil {
		_, _ = io.WriteString(customWriter, msg)
		return
	}

	logPath := customPath
	if logPath == "" {
		logPath = defaultLogPath()
	}

	// Failing to write the debug log must never end the command. This function
	// is called on the way to reporting a real error, and dying here replaces
	// that error with a complaint about a file nobody asked for. Seen for real:
	// an installation directory with no write permission turned every failure
	// into "open debug.log: read-only file system".
	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		warnOnce(err)
		return
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err = f.WriteString(msg); err != nil {
		warnOnce(err)
	}
}

// warnOnce reports that the debug log is unusable, and does it a single time.
// Repeating it on every call would bury the error the user actually needs.
func warnOnce(err error) {
	warnOnceGuard.Do(func() {
		fmt.Fprintln(os.Stderr, "madock: the debug log could not be written ("+err.Error()+"). Continuing.")
	})
}

var warnOnceGuard sync.Once

// Path is where the debug log is being written.
//
// Exported because messages that send somebody to it have to name it. "See
// debug.log for details" was measured on a live server on 2026-08-27 to be
// unusable advice: the file is in the installation directory, the person looked
// in the project and in their home, found nothing, and had to work the failure
// out from the state of the crontab instead. The path is known here; printing
// it costs nothing.
func Path() string {
	if customPath != "" {
		return customPath
	}
	return defaultLogPath()
}

// defaultLogPath puts debug.log in the installation directory.
//
// MADOCK_EXEC_DIR is read here rather than through the paths package, which
// would be an import cycle — paths logs through this one. The duplication is
// small and the alternative is worse: without it the log lands next to the
// binary, so a madock installed somewhere read-only cannot log at all, and a
// second installation pointed at its own directory would still write into the
// first one's.
func defaultLogPath() string {
	if execDir := strings.TrimSpace(os.Getenv("MADOCK_EXEC_DIR")); execDir != "" {
		return filepath.Join(execDir, "debug.log")
	}

	ex, err := os.Executable()
	if err != nil {
		// Nowhere sensible to write, and no way to report it except the return
		// value. The caller handles an unwritable path already.
		return "debug.log"
	}
	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		return filepath.Join(filepath.Dir(ex), "debug.log")
	}
	return filepath.Join(filepath.Dir(exReal), "debug.log")
}
