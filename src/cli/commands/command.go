package commands

import (
	"github.com/faradey/madock/src/cli/fmtc"
)

func IsNotDefine() {
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}
