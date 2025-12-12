package fmtc

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/src/helper/cli/color"
)

// SelectorOption represents a single option in the selector
type SelectorOption struct {
	Key         string
	Value       string
	Recommended bool
}

// Selector displays a styled selection box
func Selector(title string, options []SelectorOption, recommendedKey string) {
	// Calculate max width
	maxWidth := len(title) + 4
	for _, opt := range options {
		optLen := len(fmt.Sprintf("%s) %s", opt.Key, opt.Value))
		if opt.Recommended {
			optLen += 14 // " (recommended)"
		}
		if optLen+6 > maxWidth {
			maxWidth = optLen + 6
		}
	}

	// Ensure minimum width
	if maxWidth < 40 {
		maxWidth = 40
	}

	// Top border with title
	fmt.Printf("%s┌─ %s%s%s ", color.Cyan, color.Green, title, color.Cyan)
	remaining := maxWidth - len(title) - 4
	fmt.Print(strings.Repeat("─", remaining))
	fmt.Printf("┐%s\n", color.Reset)

	// Options
	for _, opt := range options {
		var line string
		if opt.Key == recommendedKey || opt.Recommended {
			// Highlighted recommended option
			line = fmt.Sprintf("%s→ %s%s) %s%s (recommended)%s",
				color.Green,
				color.Cyan,
				opt.Key,
				color.Green,
				opt.Value,
				color.Reset,
			)
		} else {
			// Regular option
			line = fmt.Sprintf("  %s%s) %s%s",
				color.Cyan,
				opt.Key,
				color.Reset,
				opt.Value,
			)
		}

		// Calculate visible length (without ANSI codes)
		visibleLen := len(opt.Key) + len(opt.Value) + 4 // "X) value"
		if opt.Key == recommendedKey || opt.Recommended {
			visibleLen += 16 // "→ " + " (recommended)"
		} else {
			visibleLen += 2 // "  "
		}

		padding := maxWidth - visibleLen
		if padding < 0 {
			padding = 0
		}

		fmt.Printf("%s│%s %s%s %s│%s\n",
			color.Cyan, color.Reset,
			line,
			strings.Repeat(" ", padding),
			color.Cyan, color.Reset,
		)
	}

	// Bottom border
	fmt.Printf("%s└%s┘%s\n",
		color.Cyan,
		strings.Repeat("─", maxWidth),
		color.Reset,
	)
}

// SelectorSimple displays options without a box (for backward compatibility)
func SelectorSimple(title string, options []string, recommendedIdx int) {
	fmt.Printf("\n%s%s%s\n", color.Blue, title, color.Reset)

	for i, opt := range options {
		if opt == "" {
			continue
		}
		if i == recommendedIdx {
			fmt.Printf("%s→ %d) %s (recommended)%s\n", color.Green, i, opt, color.Reset)
		} else {
			fmt.Printf("  %s%d)%s %s\n", color.Cyan, i, color.Reset, opt)
		}
	}
}
