package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
)

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
	var dirAbsPath string

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		dirAbsPath = filepath.Dir(ex)
	} else {
		dirAbsPath = filepath.Dir(exReal)
	}
	f, err := os.OpenFile(dirAbsPath+"/debug.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(f)

	if _, err = f.WriteString("\n" + fmt.Sprintln(v...) + string(debug.Stack())); err != nil {
		log.Fatal(err)
	}
}
