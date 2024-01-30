package setup

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/pwa"
)

func Execute(projectName string, projectConf map[string]string, continueSetup bool) {
	if continueSetup {
		toolsDefVersions := pwa.GetVersions()
		tools.NodeJs(&toolsDefVersions.NodeJs)
		tools.Yarn(&toolsDefVersions.Yarn)
		tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
		setMagentoBackendHost(&toolsDefVersions.PwaBackendUrl, projectConf)
		projects.SetEnvForProject(projectName, toolsDefVersions, configs.GetProjectConfigOnly(projectName))
		fmtc.SuccessLn("\n" + "Finish set up environment")

		rebuild.Execute()
	}
}

func setMagentoBackendHost(defVersion *string, projectConf map[string]string) {
	fmtc.TitleLn("BACKEND URL")
	fmt.Println("Input format: example.com")
	host := ""
	if val, ok := projectConf["pwa/backend_url"]; ok && val != "" {
		host = val
		*defVersion = host
		fmt.Println("Recommended host: " + host)
	}

	fmt.Print("> ")
	selected, _ := tools.Waiter()
	if selected != "" {
		*defVersion = selected
		fmtc.SuccessLn("Your choice: " + *defVersion)
	}
}
