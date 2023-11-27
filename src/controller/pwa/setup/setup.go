package setup

import (
	"fmt"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/pwa"
)

func Execute(projectName string, projectConf map[string]string, continueSetup, withVolumes, withChown bool) {
	if continueSetup {
		toolsDefVersions := pwa.GetVersions()
		tools.NodeJs(&toolsDefVersions.NodeJs)
		tools.Yarn(&toolsDefVersions.Yarn)
		tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
		setMagentoBackendHost(&toolsDefVersions.PwaBackendUrl, projectConf)
		projects.SetEnvForProject(projectName, toolsDefVersions, projectConf)
		fmtc.SuccessLn("\n" + "Finish set up environment")

		builder.Down(withVolumes)
		builder.StartMagento2(withChown, projectConf)
	}
}

func setMagentoBackendHost(defVersion *string, projectConf map[string]string) {
	fmtc.TitleLn("BACKEND URL")
	fmt.Println("Input format: example.com")
	host := ""
	if val, ok := projectConf["PWA_BACKEND_URL"]; ok && val != "" {
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
