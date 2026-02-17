package fmtc

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/v3/src/helper/cli/color"
)

// ValidationError displays an inline validation error
func ValidationError(message string) {
	fmt.Printf("  %s↳ %s%s\n", color.Red, message, color.Reset)
}

// ValidationWarning displays an inline validation warning
func ValidationWarning(message string) {
	fmt.Printf("  %s↳ %s%s\n", color.Yellow, message, color.Reset)
}

// ValidationHint displays an inline hint
func ValidationHint(message string) {
	fmt.Printf("  %s↳ %s%s\n", color.Gray, message, color.Reset)
}

// ClearLine clears the current line
func ClearLine() {
	fmt.Print("\r\033[K")
}

// ClearLines clears n lines above current position
func ClearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[A\033[K")
	}
}

// InputPrompt displays a styled input prompt
func InputPrompt(label string) {
	fmt.Printf("%s%s%s %s>%s ", color.Cyan, label, color.Reset, color.Green, color.Reset)
}

// InputPromptWithDefault displays a prompt with default value hint
func InputPromptWithDefault(label, defaultVal string) {
	if defaultVal != "" {
		fmt.Printf("%s%s%s %s[%s]%s %s>%s ",
			color.Cyan, label, color.Reset,
			color.Gray, defaultVal, color.Reset,
			color.Green, color.Reset,
		)
	} else {
		InputPrompt(label)
	}
}

// ValidateNumericInput validates that input is a number within range
func ValidateNumericInput(input string, min, max int) (int, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return -1, true // Empty is valid (use default)
	}

	var num int
	_, err := fmt.Sscanf(input, "%d", &num)
	if err != nil {
		return 0, false
	}

	if num < min || num > max {
		return 0, false
	}

	return num, true
}

// ShowInputError shows error and returns cursor for re-input
func ShowInputError(message string) {
	ClearLine()
	ValidationError(message)
}

// PromptWithValidation shows a prompt and validates input
type PromptWithValidation struct {
	Label       string
	Default     string
	Validator   func(string) (string, error)
	ErrorMsg    string
	MaxAttempts int
}

// Run executes the prompt with validation
func (p *PromptWithValidation) Run() (string, error) {
	attempts := 0
	maxAttempts := p.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}

	for attempts < maxAttempts {
		if p.Default != "" {
			InputPromptWithDefault(p.Label, p.Default)
		} else {
			InputPrompt(p.Label)
		}

		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)

		if input == "" && p.Default != "" {
			return p.Default, nil
		}

		if p.Validator != nil {
			result, err := p.Validator(input)
			if err != nil {
				ValidationError(err.Error())
				attempts++
				continue
			}
			return result, nil
		}

		return input, nil
	}

	return "", fmt.Errorf("maximum attempts reached")
}
