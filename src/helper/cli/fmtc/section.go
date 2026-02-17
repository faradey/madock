package fmtc

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/v3/src/helper/cli/color"
)

// SectionItem represents a key-value pair in a section
type SectionItem struct {
	Key   string
	Value string
}

// Section displays a grouped section with a title and items
func Section(title string, items []SectionItem) {
	// Calculate max key length for alignment
	maxKeyLen := 0
	for _, item := range items {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
	}

	// Section header
	repeatCount := 40 - len(title) - 4
	if repeatCount < 0 {
		repeatCount = 0
	}
	fmt.Printf("\n%s── %s %s%s\n",
		color.Cyan,
		title,
		strings.Repeat("─", repeatCount),
		color.Reset,
	)

	// Items
	for _, item := range items {
		padding := strings.Repeat(" ", maxKeyLen-len(item.Key))
		fmt.Printf("   %s%s:%s%s %s%s%s\n",
			color.Gray,
			item.Key,
			padding,
			color.Reset,
			color.Green,
			item.Value,
			color.Reset,
		)
	}
}

// SectionBox displays a boxed section with items
func SectionBox(title string, items []SectionItem) {
	// Calculate dimensions
	maxKeyLen := 0
	maxValLen := 0
	for _, item := range items {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
		if len(item.Value) > maxValLen {
			maxValLen = len(item.Value)
		}
	}

	width := maxKeyLen + maxValLen + 6
	if width < len(title)+4 {
		width = len(title) + 4
	}
	if width < 40 {
		width = 40
	}

	// Top border with title
	topRepeat := width - len(title) - 4
	if topRepeat < 0 {
		topRepeat = 0
	}
	fmt.Printf("\n%s╭─ %s%s%s ", color.Cyan, color.Green, title, color.Cyan)
	fmt.Print(strings.Repeat("─", topRepeat))
	fmt.Printf("╮%s\n", color.Reset)

	// Items
	for _, item := range items {
		keyPadding := strings.Repeat(" ", maxKeyLen-len(item.Key))
		visibleLen := len(item.Key) + maxKeyLen - len(item.Key) + 3 + len(item.Value)
		valPadding := width - visibleLen
		if valPadding < 0 {
			valPadding = 0
		}

		fmt.Printf("%s│%s  %s%s%s%s %s│%s\n",
			color.Cyan,
			color.Reset,
			color.Gray,
			item.Key,
			keyPadding,
			color.Reset,
			item.Value+strings.Repeat(" ", valPadding),
			color.Cyan,
		)
	}

	// Bottom border
	fmt.Printf("%s╰%s╯%s\n", color.Cyan, strings.Repeat("─", width), color.Reset)
}

// ConfigSummary displays a complete configuration summary with multiple sections
type ConfigSummary struct {
	Title    string
	Sections []ConfigSection
}

// ConfigSection represents a section in the summary
type ConfigSection struct {
	Name  string
	Items []SectionItem
}

// Display shows the configuration summary
func (cs *ConfigSummary) Display() {
	// Calculate overall width
	width := 44

	// Top border
	fmt.Printf("\n%s╭%s╮%s\n", color.Cyan, strings.Repeat("─", width), color.Reset)

	// Title (centered)
	titlePadding := (width - len(cs.Title)) / 2
	if titlePadding < 0 {
		titlePadding = 0
	}
	titlePaddingRight := width - titlePadding - len(cs.Title)
	if titlePaddingRight < 0 {
		titlePaddingRight = 0
	}
	fmt.Printf("%s│%s%s%s%s%s%s│%s\n",
		color.Cyan,
		color.Reset,
		strings.Repeat(" ", titlePadding),
		color.Green+cs.Title+color.Reset,
		strings.Repeat(" ", titlePaddingRight),
		"",
		color.Cyan,
		color.Reset,
	)

	// Separator
	fmt.Printf("%s├%s┤%s\n", color.Cyan, strings.Repeat("─", width), color.Reset)

	// Sections
	for i, section := range cs.Sections {
		// Section header
		fmt.Printf("%s│%s %s%-*s%s %s│%s\n",
			color.Cyan,
			color.Reset,
			color.Blue,
			width-2,
			section.Name,
			color.Reset,
			color.Cyan,
			color.Reset,
		)

		// Items
		for _, item := range section.Items {
			// Calculate padding: width - "  " - Key - ":" - " " - Value - " " = width - 5 - len(Key) - len(Value)
			padding := width - 5 - len(item.Key) - len(item.Value)
			if padding < 0 {
				padding = 0
			}
			fmt.Printf("%s│%s  %s%s:%s %s%s%s%s %s│%s\n",
				color.Cyan,
				color.Reset,
				color.Gray,
				item.Key,
				color.Reset,
				color.Green,
				item.Value,
				color.Reset,
				strings.Repeat(" ", padding),
				color.Cyan,
				color.Reset,
			)
		}

		// Section separator (except for last)
		if i < len(cs.Sections)-1 {
			fmt.Printf("%s│%s%s│%s\n", color.Cyan, color.Reset, strings.Repeat(" ", width), color.Cyan)
		}
	}

	// Bottom border
	fmt.Printf("%s╰%s╯%s\n", color.Cyan, strings.Repeat("─", width), color.Reset)
}

// SimpleSummary displays a simple two-column summary
func SimpleSummary(title string, items []SectionItem) {
	// Calculate max key length
	maxKeyLen := 0
	for _, item := range items {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
	}

	width := maxKeyLen + 20
	if width < 40 {
		width = 40
	}

	// Header
	fmt.Printf("\n%s%s%s\n", color.Green, title, color.Reset)
	fmt.Printf("%s%s%s\n", color.Cyan, strings.Repeat("─", width), color.Reset)

	// Items
	for _, item := range items {
		padding := strings.Repeat(" ", maxKeyLen-len(item.Key))
		fmt.Printf("  %s%s%s  %s%s%s\n",
			color.Gray,
			item.Key+padding,
			color.Reset,
			color.Cyan,
			item.Value,
			color.Reset,
		)
	}
}
