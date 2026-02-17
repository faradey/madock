package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
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

	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(f)

	if _, err = f.WriteString(msg); err != nil {
		log.Fatal(err)
	}
}

func defaultLogPath() string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		return filepath.Dir(ex) + "/debug.log"
	}
	return filepath.Dir(exReal) + "/debug.log"
}
