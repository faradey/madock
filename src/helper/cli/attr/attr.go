package attr

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/version"
)

var IsParseArgs = true
var IsQuiet = false

// HelpRenderer prints the detailed help of one command. The help package sets it
// in its init — it cannot be called directly from here because help imports this
// package. Nil until then, and Parse falls back to go-arg's own usage block.
var HelpRenderer func(command string)

type Arguments struct {
	Json  bool `arg:"--json,-j" help:"Output in JSON format"`
	Quiet bool `arg:"--quiet,-q" help:"Suppress Docker build/pull output"`
}

func (a *Arguments) GetQuiet() bool {
	return a.Quiet
}

// AttachOutput connects cmd stdout/stderr to os.Stdout/os.Stderr unless quiet mode is active
func AttachOutput(cmd *exec.Cmd) {
	if !IsQuiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
}

type ArgumentsWithArgs struct {
	Arguments
	Args []string `arg:"positional"`
}

func Parse(dest any) any {
	if IsParseArgs && len(os.Args) > 1 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, dest)

		if err != nil {
			logger.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		// go-arg answers -h/--help and --version with sentinel errors instead of
		// printing anything. Passing them to logger.Fatal turned every
		// "madock <command> --help" into a bare "help requested by user" line.
		if errors.Is(err, arg.ErrHelp) {
			if HelpRenderer != nil {
				HelpRenderer(os.Args[1])
			} else {
				p.WriteHelp(os.Stdout)
			}
			os.Exit(0)
		}
		if errors.Is(err, arg.ErrVersion) {
			fmt.Println("madock " + version.Version)
			os.Exit(0)
		}

		if err != nil {
			logger.Fatal(err)
		}

		if a, ok := dest.(interface{ GetQuiet() bool }); ok { //nolint:iface
			IsQuiet = a.GetQuiet()
		}
	}

	IsParseArgs = false
	return dest
}
