package fmtc

import (
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/src/helper/cli/color"
	"golang.org/x/term"
)

// InteractiveSelector displays an interactive selector with arrow key navigation
type InteractiveSelector struct {
	Title       string
	Options     []string
	Selected    int
	Recommended int
	errorMsg    string
}

// NewInteractiveSelector creates a new interactive selector
func NewInteractiveSelector(title string, options []string, recommended int) *InteractiveSelector {
	// Filter out empty options and track the recommended index
	var filteredOptions []string
	newRecommended := -1
	for i, opt := range options {
		if opt != "" {
			if i == recommended {
				newRecommended = len(filteredOptions)
			}
			filteredOptions = append(filteredOptions, opt)
		}
	}

	selected := 0
	if newRecommended >= 0 {
		selected = newRecommended
	}

	return &InteractiveSelector{
		Title:       title,
		Options:     filteredOptions,
		Selected:    selected,
		Recommended: newRecommended,
	}
}

// Run displays the selector and returns the selected index and value
func (s *InteractiveSelector) Run() (int, string) {
	// Check if we're in a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Fallback to non-interactive mode
		return s.Selected, s.Options[s.Selected]
	}

	// Save terminal state and set raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback to non-interactive mode
		return s.Selected, s.Options[s.Selected]
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	s.render()

	// Read input
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}

		if n == 1 {
			switch buf[0] {
			case 13, 10: // Enter
				s.clearAndPrintResult()
				return s.Selected, s.Options[s.Selected]
			case 3: // Ctrl+C
				s.clearLines(len(s.Options) + 4)
				term.Restore(int(os.Stdin.Fd()), oldState)
				os.Exit(0)
			case 'j', 'J': // vim down
				s.moveDown()
			case 'k', 'K': // vim up
				s.moveUp()
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				idx := int(buf[0] - '0')
				if idx < len(s.Options) {
					s.Selected = idx
					s.errorMsg = ""
					s.clearAndPrintResult()
					return s.Selected, s.Options[s.Selected]
				} else {
					s.errorMsg = fmt.Sprintf("Invalid option. Enter 0-%d", len(s.Options)-1)
					s.render()
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up arrow
				s.moveUp()
			case 66: // Down arrow
				s.moveDown()
			}
		}
	}

	return s.Selected, s.Options[s.Selected]
}

func (s *InteractiveSelector) moveUp() {
	if s.Selected > 0 {
		s.Selected--
		s.errorMsg = ""
		s.render()
	}
}

func (s *InteractiveSelector) moveDown() {
	if s.Selected < len(s.Options)-1 {
		s.Selected++
		s.errorMsg = ""
		s.render()
	}
}

func (s *InteractiveSelector) render() {
	// Move cursor up and clear previous render
	extraLines := 0
	if s.errorMsg != "" {
		extraLines = 1
	}
	s.clearLines(len(s.Options) + 4 + extraLines)

	// Calculate width
	maxWidth := len(s.Title) + 4
	for i, opt := range s.Options {
		optLen := len(fmt.Sprintf("%d) %s", i, opt))
		if i == s.Recommended {
			optLen += 14
		}
		if optLen+8 > maxWidth {
			maxWidth = optLen + 8
		}
	}
	if maxWidth < 40 {
		maxWidth = 40
	}

	// Top border
	fmt.Printf("%s┌─ %s%s%s ", color.Cyan, color.Green, s.Title, color.Cyan)
	fmt.Print(strings.Repeat("─", maxWidth-len(s.Title)-4))
	fmt.Printf("┐%s\r\n", color.Reset)

	// Options
	for i, opt := range s.Options {
		isSelected := i == s.Selected
		isRecommended := i == s.Recommended

		fmt.Printf("%s│%s ", color.Cyan, color.Reset)

		if isSelected {
			fmt.Printf("%s▸ %s", color.Green, color.Reset)
		} else {
			fmt.Print("  ")
		}

		if isSelected {
			fmt.Printf("%s%d) %s", color.Green, i, opt)
		} else {
			fmt.Printf("%s%d)%s %s", color.Cyan, i, color.Reset, opt)
		}

		if isRecommended {
			fmt.Printf(" %s(recommended)%s", color.Gray, color.Reset)
		}

		// Padding
		visibleLen := len(fmt.Sprintf("%d) %s", i, opt)) + 4
		if isRecommended {
			visibleLen += 14
		}
		padding := maxWidth - visibleLen
		if padding > 0 {
			fmt.Print(strings.Repeat(" ", padding))
		}

		fmt.Printf(" %s│%s\r\n", color.Cyan, color.Reset)
	}

	// Bottom border
	fmt.Printf("%s└%s┘%s\r\n", color.Cyan, strings.Repeat("─", maxWidth), color.Reset)

	// Help text
	fmt.Printf("%s↑/↓%s Navigate  %s•%s  %sEnter%s Select  %s•%s  %s0-9%s Quick select  %s•%s  %sCtrl+C%s Cancel\r\n",
		color.Cyan, color.Gray,
		color.Gray, color.Reset,
		color.Cyan, color.Gray,
		color.Gray, color.Reset,
		color.Cyan, color.Gray,
		color.Gray, color.Reset,
		color.Cyan, color.Reset,
	)

	// Error message if any
	if s.errorMsg != "" {
		fmt.Printf("  %s↳ %s%s\r\n", color.Red, s.errorMsg, color.Reset)
	}
}

func (s *InteractiveSelector) clearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[2K") // Clear line
		if i < n-1 {
			fmt.Print("\033[A") // Move up
		}
	}
	fmt.Print("\r") // Return to start of line
}

func (s *InteractiveSelector) clearAndPrintResult() {
	s.clearLines(len(s.Options) + 4)
	fmt.Printf("%s✓ %s: %s%s%s\r\n",
		color.Green,
		s.Title,
		color.Cyan,
		s.Options[s.Selected],
		color.Reset,
	)
}
