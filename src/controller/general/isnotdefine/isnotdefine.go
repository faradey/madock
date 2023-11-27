package isnotdefine

import (
	"github.com/faradey/madock/src/helper/cli/fmtc"
)

func Execute() {
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}
