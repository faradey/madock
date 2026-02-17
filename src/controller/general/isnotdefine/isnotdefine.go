package isnotdefine

import (
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"os"
	"os/exec"
	"strings"
)

func Execute(originCommand string) {
	projectConf := configs.GetCurrentProjectConfig()
	commands := configs.GetCommands(projectConf)
	for _, command := range commands {
		alias := command["alias"]
		origin := strings.Replace(command["origin"], "_args_", strings.Join(os.Args[2:], " "), 1)
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
