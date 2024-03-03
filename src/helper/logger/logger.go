package logger

import (
	"log"
	"runtime/debug"
)

func Fatal(v ...any) {
	debug.PrintStack()
	log.Fatal(v)
}

func Fatalln(v ...any) {
	debug.PrintStack()
	log.Fatalln(v)
}
