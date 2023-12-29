package attr

import (
	"github.com/alexflint/go-arg"
	"log"
	"os"
)

var IsParseArgs = true

type Arguments struct {
}

type ArgumentsWithArgs struct {
	Arguments
	Args []string `arg:"positional"`
}

func Parse(dest interface{}) interface{} {
	if IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, dest)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	IsParseArgs = false
	return dest
}
