package fmtc

import (
	"fmt"
	"github.com/faradey/madock/src/cli/color"
)

func Title(txt string) {
	fmt.Print(color.Blue + txt + color.Reset)
}

func TitleLn(txt string) {
	fmt.Println(color.Blue + txt + color.Reset)
}

func ErrorLn(txt string) {
	fmt.Println(color.Red + txt + color.Reset)
}

func WarningLn(txt string) {
	Warning(txt + "\n")
}

func Warning(txt string) {
	fmt.Print(color.Yellow + txt + color.Reset)
}

func ToDoLn(txt string) {
	ToDo(txt + "\n")
}

func ToDo(txt string) {
	fmt.Print(color.White + txt + color.Reset)
}

func SuccessLn(txt string) {
	Success(txt + "\n")
}

func Success(txt string) {
	fmt.Print(color.Green + txt + color.Reset)
}
