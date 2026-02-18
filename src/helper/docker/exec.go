package docker

import (
	"os"
	"os/exec"

	"golang.org/x/term"
)

// CommandInterceptor allows enterprise to intercept docker exec commands
// for auditing, sanitization, or blocking dangerous patterns.
type CommandInterceptor interface {
	BeforeExec(container string, command []string) ([]string, error)
	AfterExec(container string, command []string, execErr error)
}

var commandInterceptor CommandInterceptor

// IsTTYAvailable checks whether a TTY is available for docker exec.
// It respects the MADOCK_TTY_ENABLED env var (0=off, 1=on) and
// falls back to checking if stdin is a terminal.
func IsTTYAvailable() bool {
	switch os.Getenv("MADOCK_TTY_ENABLED") {
	case "0":
		return false
	case "1":
		return true
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// SetCommandInterceptor sets a custom interceptor for docker exec commands.
func SetCommandInterceptor(i CommandInterceptor) {
	commandInterceptor = i
}

// ContainerExec runs a command inside a Docker container with standard I/O (os.Stdin/Stdout/Stderr).
func ContainerExec(container, user string, interactive bool, command ...string) error {
	cmd, err := PrepareContainerExec(container, user, interactive, command...)
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	execErr := cmd.Run()
	NotifyExecDone(container, command, execErr)
	return execErr
}

// PrepareContainerExec creates a docker exec *exec.Cmd with interceptor support.
// Caller sets Stdin/Stdout/Stderr, calls cmd.Run(), then calls NotifyExecDone().
func PrepareContainerExec(container, user string, interactive bool, command ...string) (*exec.Cmd, error) {
	if commandInterceptor != nil {
		var err error
		command, err = commandInterceptor.BeforeExec(container, command)
		if err != nil {
			return nil, err
		}
	}

	args := []string{"exec"}
	if interactive && IsTTYAvailable() {
		args = append(args, "-it")
	} else {
		args = append(args, "-i")
	}
	if user != "" {
		args = append(args, "-u", user)
	}
	args = append(args, container)
	args = append(args, command...)

	return exec.Command("docker", args...), nil
}

// NotifyExecDone notifies the interceptor that a command has completed.
// Call after PrepareContainerExec + cmd.Run().
func NotifyExecDone(container string, command []string, execErr error) {
	if commandInterceptor != nil {
		commandInterceptor.AfterExec(container, command, execErr)
	}
}
