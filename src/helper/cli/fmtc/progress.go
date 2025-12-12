package fmtc

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/src/helper/cli/color"
)

// StepProgress manages and displays setup progress
type StepProgress struct {
	steps       []string
	currentStep int
	totalSteps  int
}

// NewStepProgress creates a new progress tracker
func NewStepProgress(steps []string) *StepProgress {
	return &StepProgress{
		steps:       steps,
		currentStep: 0,
		totalSteps:  len(steps),
	}
}

// SetStep sets the current step (1-indexed for user display)
func (sp *StepProgress) SetStep(step int) {
	if step >= 1 && step <= sp.totalSteps {
		sp.currentStep = step
	}
}

// Display shows the current progress bar and step name
func (sp *StepProgress) Display() {
	if sp.currentStep == 0 || sp.currentStep > sp.totalSteps {
		return
	}

	// Calculate progress bar
	barWidth := 20
	filled := (sp.currentStep * barWidth) / sp.totalSteps
	empty := barWidth - filled

	// Build progress bar
	filledBar := strings.Repeat("\u2588", filled) // █
	emptyBar := strings.Repeat("\u2591", empty)   // ░

	// Get current step name
	stepName := sp.steps[sp.currentStep-1]

	// Format: [████████░░░░░░░░░░░░] Step 3/8: Selecting PHP version
	fmt.Printf("\n%s[%s%s%s%s]%s Step %d/%d: %s%s%s\n",
		color.Cyan,
		color.Green,
		filledBar,
		color.Gray,
		emptyBar,
		color.Reset,
		sp.currentStep,
		sp.totalSteps,
		color.Blue,
		stepName,
		color.Reset,
	)
}

// DisplayCompact shows a compact single-line progress
func (sp *StepProgress) DisplayCompact() {
	if sp.currentStep == 0 || sp.currentStep > sp.totalSteps {
		return
	}

	stepName := sp.steps[sp.currentStep-1]
	fmt.Printf("%s[%d/%d]%s %s\n",
		color.Cyan,
		sp.currentStep,
		sp.totalSteps,
		color.Reset,
		stepName,
	)
}

// Complete displays a completion message
func (sp *StepProgress) Complete() {
	barWidth := 20
	filledBar := strings.Repeat("\u2588", barWidth) // █

	fmt.Printf("\n%s[%s%s%s]%s %sSetup Complete!%s\n",
		color.Cyan,
		color.Green,
		filledBar,
		color.Cyan,
		color.Reset,
		color.Green,
		color.Reset,
	)
}
