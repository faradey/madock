package fmtc

import (
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/src/helper/cli/color"
	"golang.org/x/term"
)

// Confirm displays a confirmation prompt and returns the user's choice
// defaultYes determines if Enter defaults to Yes (true) or No (false)
func Confirm(message string, defaultYes bool) bool {
	var hint string
	if defaultYes {
		hint = "[Y/n]"
	} else {
		hint = "[y/N]"
	}

	fmt.Printf("%s%s%s %s%s%s ",
		color.Cyan,
		message,
		color.Reset,
		color.Gray,
		hint,
		color.Reset,
	)

	// Try interactive mode first
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer term.Restore(int(os.Stdin.Fd()), oldState)

			buf := make([]byte, 1)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil || n == 0 {
					break
				}

				switch buf[0] {
				case 'y', 'Y':
					fmt.Printf("%sYes%s\r\n", color.Green, color.Reset)
					return true
				case 'n', 'N':
					fmt.Printf("%sNo%s\r\n", color.Yellow, color.Reset)
					return false
				case 13, 10: // Enter
					if defaultYes {
						fmt.Printf("%sYes%s\r\n", color.Green, color.Reset)
						return true
					}
					fmt.Printf("%sNo%s\r\n", color.Yellow, color.Reset)
					return false
				case 3: // Ctrl+C
					fmt.Printf("%sCancelled%s\r\n", color.Red, color.Reset)
					term.Restore(int(os.Stdin.Fd()), oldState)
					os.Exit(0)
				}
			}
		}
	}

	// Fallback to line-based input
	var input string
	fmt.Scanln(&input)
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}

// ConfirmWithOptions displays a confirmation with custom options
func ConfirmWithOptions(message string, options []string, defaultIdx int) int {
	// Build options string
	var optStrs []string
	for i, opt := range options {
		if i == defaultIdx {
			optStrs = append(optStrs, fmt.Sprintf("%s[%s]%s", color.Green, opt, color.Reset))
		} else {
			optStrs = append(optStrs, opt)
		}
	}

	fmt.Printf("%s%s%s %s(%s)%s ",
		color.Cyan,
		message,
		color.Reset,
		color.Gray,
		strings.Join(optStrs, "/"),
		color.Reset,
	)

	var input string
	fmt.Scanln(&input)
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultIdx
	}

	for i, opt := range options {
		if strings.ToLower(opt) == input || (len(input) == 1 && strings.ToLower(string(opt[0])) == input) {
			return i
		}
	}

	return defaultIdx
}

// ConfirmBox displays a boxed confirmation prompt
func ConfirmBox(title string, message string, defaultYes bool) bool {
	width := len(message) + 4
	if width < len(title)+4 {
		width = len(title) + 4
	}
	if width < 40 {
		width = 40
	}

	// Top border
	fmt.Printf("\n%s╭─ %s%s%s %s╮%s\n",
		color.Cyan,
		color.Yellow,
		title,
		color.Cyan,
		strings.Repeat("─", width-len(title)-4),
		color.Reset,
	)

	// Message
	msgPadding := width - len(message)
	fmt.Printf("%s│%s %s%s %s│%s\n",
		color.Cyan,
		color.Reset,
		message,
		strings.Repeat(" ", msgPadding-1),
		color.Cyan,
		color.Reset,
	)

	// Bottom border
	fmt.Printf("%s╰%s╯%s\n",
		color.Cyan,
		strings.Repeat("─", width),
		color.Reset,
	)

	return Confirm("", defaultYes)
}

// ProceedPrompt displays a "Press Enter to continue" prompt
func ProceedPrompt(message string) {
	if message == "" {
		message = "Press Enter to continue..."
	}
	fmt.Printf("%s%s%s ", color.Gray, message, color.Reset)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer term.Restore(int(os.Stdin.Fd()), oldState)

			buf := make([]byte, 1)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil || n == 0 {
					break
				}
				if buf[0] == 13 || buf[0] == 10 { // Enter
					fmt.Print("\r\n")
					return
				}
				if buf[0] == 3 { // Ctrl+C
					term.Restore(int(os.Stdin.Fd()), oldState)
					os.Exit(0)
				}
			}
		}
	}

	// Fallback
	fmt.Scanln()
}
