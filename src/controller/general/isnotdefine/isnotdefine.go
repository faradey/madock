package isnotdefine

import (
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"os"
	"os/exec"
)

func Execute(originCommand string) {
	projectConf := configs.GetCurrentProjectConfig()
	commands := configs.GetCommands(projectConf)
	for _, command := range commands {
		alias := command["alias"]
		origin := command["origin"]
		if alias != "" && origin != "" && alias == originCommand {
			cmd := exec.Command("bash", "-c", origin)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				logger.Fatal(err)
			} else {
				return
			}
		}
	}
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}
