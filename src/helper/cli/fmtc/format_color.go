package fmtc

import (
	"fmt"
	"github.com/faradey/madock/v3/src/helper/cli/color"
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

func Purple(txt string) {
	fmt.Print(color.Purple + txt + color.Reset)
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

// SuccessIcon prints success message with checkmark icon
func SuccessIcon(txt string) {
	fmt.Print(color.Green + "✓ " + color.Reset + txt)
}

// SuccessIconLn prints success message with checkmark icon and newline
func SuccessIconLn(txt string) {
	SuccessIcon(txt + "\n")
}

// ErrorIcon prints error message with cross icon
func ErrorIcon(txt string) {
	fmt.Print(color.Red + "✗ " + color.Reset + txt)
}

// ErrorIconLn prints error message with cross icon and newline
func ErrorIconLn(txt string) {
	ErrorIcon(txt + "\n")
}

// WarningIcon prints warning message with warning icon
func WarningIcon(txt string) {
	fmt.Print(color.Yellow + "⚠ " + color.Reset + txt)
}

// WarningIconLn prints warning message with warning icon and newline
func WarningIconLn(txt string) {
	WarningIcon(txt + "\n")
}

// InfoIcon prints info message with info icon
func InfoIcon(txt string) {
	fmt.Print(color.Blue + "ℹ " + color.Reset + txt)
}

// InfoIconLn prints info message with info icon and newline
func InfoIconLn(txt string) {
	InfoIcon(txt + "\n")
}

// Gray returns the gray color code
func Gray() string {
	return color.Gray
}

// ResetColor returns the reset color code
func ResetColor() string {
	return color.Reset
}
