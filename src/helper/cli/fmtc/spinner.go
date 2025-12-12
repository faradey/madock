package fmtc

import (
	"fmt"
	"sync"
	"time"

	"github.com/faradey/madock/src/helper/cli/color"
)

// Spinner displays an animated spinner with a message
type Spinner struct {
	frames   []string
	message  string
	running  bool
	mutex    sync.Mutex
	done     chan bool
	interval time.Duration
}

// SpinnerStyle defines different spinner animations
type SpinnerStyle int

const (
	SpinnerDots SpinnerStyle = iota
	SpinnerLine
	SpinnerCircle
	SpinnerBounce
)

var spinnerFrames = map[SpinnerStyle][]string{
	SpinnerDots:   {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	SpinnerLine:   {"-", "\\", "|", "/"},
	SpinnerCircle: {"◐", "◓", "◑", "◒"},
	SpinnerBounce: {"⠁", "⠂", "⠄", "⠂"},
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames:   spinnerFrames[SpinnerDots],
		message:  message,
		interval: 80 * time.Millisecond,
		done:     make(chan bool),
	}
}

// NewSpinnerWithStyle creates a spinner with a specific style
func NewSpinnerWithStyle(message string, style SpinnerStyle) *Spinner {
	frames, ok := spinnerFrames[style]
	if !ok {
		frames = spinnerFrames[SpinnerDots]
	}
	return &Spinner{
		frames:   frames,
		message:  message,
		interval: 80 * time.Millisecond,
		done:     make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return
	}
	s.running = true
	s.mutex.Unlock()

	go func() {
		frameIdx := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mutex.Lock()
				if !s.running {
					s.mutex.Unlock()
					return
				}
				frame := s.frames[frameIdx]
				msg := s.message
				s.mutex.Unlock()

				// Clear line and print spinner
				fmt.Printf("\r\033[K%s%s%s %s", color.Cyan, frame, color.Reset, msg)

				frameIdx = (frameIdx + 1) % len(s.frames)
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)
	fmt.Print("\r\033[K") // Clear line
}

// StopWithMessage stops the spinner and shows a final message
func (s *Spinner) StopWithMessage(message string) {
	s.mutex.Lock()
	wasRunning := s.running
	if s.running {
		s.running = false
		close(s.done)
	}
	s.mutex.Unlock()

	if wasRunning {
		time.Sleep(100 * time.Millisecond) // Allow goroutine to exit
	}
	fmt.Printf("\r\033[K%s\n", message)
}

// StopWithSuccess stops and shows a success message
func (s *Spinner) StopWithSuccess(message string) {
	s.StopWithMessage(fmt.Sprintf("%s✓%s %s", color.Green, color.Reset, message))
}

// StopWithError stops and shows an error message
func (s *Spinner) StopWithError(message string) {
	s.StopWithMessage(fmt.Sprintf("%s✗%s %s", color.Red, color.Reset, message))
}

// StopWithWarning stops and shows a warning message
func (s *Spinner) StopWithWarning(message string) {
	s.StopWithMessage(fmt.Sprintf("%s⚠%s %s", color.Yellow, color.Reset, message))
}

// UpdateMessage updates the spinner message while running
func (s *Spinner) UpdateMessage(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.message = message
}

// WithSpinner executes a function while showing a spinner
func WithSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()

	err := fn()

	if err != nil {
		spinner.StopWithError(err.Error())
	} else {
		spinner.StopWithSuccess(message + " - Done")
	}

	return err
}

// WithSpinnerResult executes a function and returns custom success message
func WithSpinnerResult(message string, fn func() (string, error)) error {
	spinner := NewSpinner(message)
	spinner.Start()

	result, err := fn()

	if err != nil {
		spinner.StopWithError(err.Error())
	} else {
		spinner.StopWithSuccess(result)
	}

	return err
}
