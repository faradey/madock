package fmtc

import (
	"fmt"
	"github.com/faradey/madock/src/cli/color"
)

func TitleLn(txt string) {
	fmt.Println(color.Blue + txt + color.Reset)
}

func ErrorLn(txt string) {
	fmt.Println(color.Red + txt + color.Reset)
}

func WarningLn(txt string) {
	fmt.Println(color.Yellow + txt + color.Reset)
}
