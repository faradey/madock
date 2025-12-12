package fmtc

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/src/helper/cli/color"
)

// HintItem represents a single hint with key and description
type HintItem struct {
	Key         string
	Description string
}

// Hints displays a help hints bar
func Hints(items []HintItem) {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s%s%s %s",
			color.Cyan, item.Key, color.Reset, item.Description))
	}
	fmt.Printf("%s%s%s\n", color.Gray, strings.Join(parts, "  •  "), color.Reset)
}

// HintsCompact displays hints in a more compact format
func HintsCompact(items []HintItem) {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s%s%s:%s",
			color.Cyan, item.Key, color.Reset, item.Description))
	}
	fmt.Printf("%s%s%s\n", color.Gray, strings.Join(parts, " | "), color.Reset)
}

// HintsLine displays a single line of hints with separator
func HintsLine(hints string) {
	fmt.Printf("%s%s%s\n", color.Gray, hints, color.Reset)
}

// CommonHints provides pre-defined hint sets
var CommonHints = struct {
	Navigation    []HintItem
	Confirmation  []HintItem
	Selection     []HintItem
	Editor        []HintItem
	Cancel        []HintItem
}{
	Navigation: []HintItem{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "Ctrl+C", Description: "Cancel"},
	},
	Confirmation: []HintItem{
		{Key: "Y", Description: "Yes"},
		{Key: "N", Description: "No"},
		{Key: "Enter", Description: "Default"},
	},
	Selection: []HintItem{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "0-9", Description: "Quick select"},
	},
	Editor: []HintItem{
		{Key: "Ctrl+S", Description: "Save"},
		{Key: "Ctrl+C", Description: "Cancel"},
		{Key: "Esc", Description: "Exit"},
	},
	Cancel: []HintItem{
		{Key: "Ctrl+C", Description: "Cancel"},
	},
}

// NavigationHints displays standard navigation hints
func NavigationHints() {
	Hints(CommonHints.Navigation)
}

// SelectionHints displays selection hints
func SelectionHints() {
	Hints(CommonHints.Selection)
}

// ConfirmationHints displays confirmation hints
func ConfirmationHints() {
	Hints(CommonHints.Confirmation)
}

// HelpBar displays a formatted help bar at the bottom
func HelpBar(items []HintItem) {
	width := 60
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s%s%s %s",
			color.Cyan, item.Key, color.Gray, item.Description))
	}
	content := strings.Join(parts, "  •  ")

	// Add padding to fill width
	padding := width - len(stripAnsi(content))
	if padding < 0 {
		padding = 0
	}

	fmt.Printf("%s%s%s%s\n", color.Gray, content, strings.Repeat(" ", padding), color.Reset)
}

// HelpBarBoxed displays hints in a boxed format
func HelpBarBoxed(items []HintItem) {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s%s%s %s",
			color.Cyan, item.Key, color.Reset, item.Description))
	}
	content := strings.Join(parts, "  │  ")

	fmt.Printf("%s┌%s┐%s\n", color.Gray, strings.Repeat("─", len(stripAnsi(content))+2), color.Reset)
	fmt.Printf("%s│%s %s %s│%s\n", color.Gray, color.Reset, content, color.Gray, color.Reset)
	fmt.Printf("%s└%s┘%s\n", color.Gray, strings.Repeat("─", len(stripAnsi(content))+2), color.Reset)
}

// ContextualHint displays a contextual hint message
func ContextualHint(message string) {
	fmt.Printf("%s💡 %s%s\n", color.Gray, message, color.Reset)
}

// Tip displays a tip message
func Tip(message string) {
	fmt.Printf("%sTip:%s %s\n", color.Cyan, color.Reset, message)
}

// stripAnsi removes ANSI escape codes for length calculation
func stripAnsi(str string) string {
	result := str
	// Remove common ANSI codes
	codes := []string{
		color.Reset, color.Red, color.Green, color.Yellow,
		color.Blue, color.Purple, color.Cyan, color.Gray, color.White,
	}
	for _, code := range codes {
		result = strings.ReplaceAll(result, code, "")
	}
	return result
}
