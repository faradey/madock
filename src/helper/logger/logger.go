package logger

import (
	"fmt"
	"io"
	"log"
	"os"
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
