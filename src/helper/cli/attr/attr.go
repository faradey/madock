package attr

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/v3/src/helper/logger"
	"os"
)

var IsParseArgs = true
var IsQuiet = false

type Arguments struct {
	Json  bool `arg:"--json,-j" help:"Output in JSON format"`
	Quiet bool `arg:"--quiet,-q" help:"Suppress Docker build/pull output"`
}

func (a *Arguments) GetQuiet() bool {
	return a.Quiet
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
