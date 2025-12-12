package fmtc

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/src/helper/cli/color"
)

// PresetOption represents a preset option for selection
type PresetOption struct {
	Name        string
	Description string
	IsCustom    bool
}

// SelectPreset displays an interactive preset selector and returns the selected index
func SelectPreset(title string, presets []PresetOption) int {
	options := make([]string, len(presets))
	for i, preset := range presets {
		if preset.IsCustom {
			options[i] = fmt.Sprintf("%s - %s", preset.Name, preset.Description)
		} else {
			options[i] = fmt.Sprintf("%s - %s", preset.Name, preset.Description)
		}
	}

	// Last option (Custom) is not recommended, first preset is
	selector := NewInteractiveSelector(title, options, 0)
	idx, _ := selector.Run()
	return idx
}

// DisplayPresetInfo shows information about a selected preset
func DisplayPresetInfo(name, description string, details map[string]string) {
	fmt.Println("")
	fmt.Printf("%s┌─ %sPreset Selected%s ", color.Cyan, color.Green, color.Cyan)
	fmt.Print(strings.Repeat("─", 40))
	fmt.Printf("┐%s\n", color.Reset)

	fmt.Printf("%s│%s  %s%-20s%s %s│%s\n",
		color.Cyan, color.Reset,
		color.Green, name,
		strings.Repeat(" ", 38-len(name)),
		color.Cyan, color.Reset)

	fmt.Printf("%s│%s  %s%-57s%s %s│%s\n",
		color.Cyan, color.Reset,
		color.Gray, description,
		"",
		color.Cyan, color.Reset)

	fmt.Printf("%s├%s┤%s\n", color.Cyan, strings.Repeat("─", 58), color.Reset)

	for key, value := range details {
		padding := 55 - len(key) - len(value)
		if padding < 0 {
			padding = 0
		}
		fmt.Printf("%s│%s  %s%-15s%s %s%s %s│%s\n",
			color.Cyan, color.Reset,
			color.Gray, key+":",
			color.Reset, value,
			strings.Repeat(" ", padding),
			color.Cyan, color.Reset)
	}

	fmt.Printf("%s└%s┘%s\n", color.Cyan, strings.Repeat("─", 58), color.Reset)
}
