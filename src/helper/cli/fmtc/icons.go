package fmtc

import (
	"fmt"

	"github.com/faradey/madock/src/helper/cli/color"
)

// Icons for CLI output
const (
	IconSuccess  = "✓"
	IconError    = "✗"
	IconWarning  = "⚠"
	IconInfo     = "ℹ"
	IconArrow    = "→"
	IconBullet   = "•"
	IconCheck    = "✔"
	IconCross    = "✘"
	IconStar     = "★"
	IconDot      = "●"
	IconCircle   = "○"
	IconSquare   = "■"
	IconTriangle = "▲"
	IconHeart    = "♥"
	IconSpinner  = "◐"
)

// PrintSuccess prints a success message with icon
func PrintSuccess(message string) {
	fmt.Printf("%s%s%s %s\n", color.Green, IconSuccess, color.Reset, message)
}

// PrintError prints an error message with icon
func PrintError(message string) {
	fmt.Printf("%s%s%s %s\n", color.Red, IconError, color.Reset, message)
}

// PrintWarning prints a warning message with icon
func PrintWarning(message string) {
	fmt.Printf("%s%s%s %s\n", color.Yellow, IconWarning, color.Reset, message)
}

// PrintInfo prints an info message with icon
func PrintInfo(message string) {
	fmt.Printf("%s%s%s %s\n", color.Blue, IconInfo, color.Reset, message)
}

// PrintBullet prints a bullet point
func PrintBullet(message string) {
	fmt.Printf("  %s%s%s %s\n", color.Cyan, IconBullet, color.Reset, message)
}

// PrintArrow prints a message with arrow
func PrintArrow(message string) {
	fmt.Printf("%s%s%s %s\n", color.Cyan, IconArrow, color.Reset, message)
}

// PrintStep prints a step indicator
func PrintStep(current, total int, message string) {
	fmt.Printf("%s[%d/%d]%s %s\n", color.Cyan, current, total, color.Reset, message)
}

// PrintKeyValue prints a key-value pair with formatting
func PrintKeyValue(key, value string) {
	fmt.Printf("  %s%s:%s %s%s%s\n", color.Gray, key, color.Reset, color.Cyan, value, color.Reset)
}

// PrintKeyValueSuccess prints a key-value pair with success icon
func PrintKeyValueSuccess(key, value string) {
	fmt.Printf("  %s%s%s %s%s:%s %s\n", color.Green, IconSuccess, color.Reset, color.Gray, key, color.Reset, value)
}

// PrintKeyValueError prints a key-value pair with error icon
func PrintKeyValueError(key, value string) {
	fmt.Printf("  %s%s%s %s%s:%s %s\n", color.Red, IconError, color.Reset, color.Gray, key, color.Reset, value)
}

// PrintList prints a list with bullets
func PrintList(items []string) {
	for _, item := range items {
		PrintBullet(item)
	}
}

// PrintNumberedList prints a numbered list
func PrintNumberedList(items []string) {
	for i, item := range items {
		fmt.Printf("  %s%d.%s %s\n", color.Cyan, i+1, color.Reset, item)
	}
}

// PrintDivider prints a horizontal divider
func PrintDivider(width int) {
	if width <= 0 {
		width = 40
	}
	fmt.Printf("%s%s%s\n", color.Gray, repeatChar("─", width), color.Reset)
}

// PrintHeader prints a styled header
func PrintHeader(title string) {
	width := len(title) + 4
	if width < 30 {
		width = 30
	}
	fmt.Printf("\n%s%s %s%s%s %s%s\n",
		color.Cyan,
		repeatChar("─", 2),
		color.Green,
		title,
		color.Cyan,
		repeatChar("─", width-len(title)-3),
		color.Reset,
	)
}

// PrintSubHeader prints a sub-header
func PrintSubHeader(title string) {
	fmt.Printf("\n%s%s %s%s\n", color.Blue, IconArrow, title, color.Reset)
}

// StatusLine prints a status line with icon based on success
func StatusLine(message string, success bool) {
	if success {
		PrintSuccess(message)
	} else {
		PrintError(message)
	}
}

// TaskStatus prints a task with its status
func TaskStatus(task string, status string) {
	var icon, statusColor string
	switch status {
	case "done", "completed", "success":
		icon = IconSuccess
		statusColor = color.Green
	case "failed", "error":
		icon = IconError
		statusColor = color.Red
	case "warning", "skipped":
		icon = IconWarning
		statusColor = color.Yellow
	case "running", "in_progress":
		icon = IconSpinner
		statusColor = color.Cyan
	default:
		icon = IconCircle
		statusColor = color.Gray
	}
	fmt.Printf("  %s%s%s %s %s[%s]%s\n", statusColor, icon, color.Reset, task, color.Gray, status, color.Reset)
}

func repeatChar(char string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += char
	}
	return result
}
