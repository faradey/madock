package attr

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/v3/src/helper/logger"
	"os"
)

var IsParseArgs = true

type Arguments struct {
	Json bool `arg:"--json,-j" help:"Output in JSON format"`
}

type ArgumentsWithArgs struct {
	Arguments
	Args []string `arg:"positional"`
}

func Parse(dest interface{}) interface{} {
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
	}

	IsParseArgs = false
	return dest
}
