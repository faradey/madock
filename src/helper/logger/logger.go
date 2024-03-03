package logger

import (
	"log"
	"runtime/debug"
)

func Fatalln(v ...any) {
	debug.PrintStack()
	log.Fatalln(v)
}
