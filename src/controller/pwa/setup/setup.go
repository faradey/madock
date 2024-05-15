package setup

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/pwa"
)

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	if continueSetup {
		toolsDefVersions := pwa.GetVersions()
		if args.NodeJs == "" {
			tools.NodeJs(&toolsDefVersions.NodeJs)
		} else {
			toolsDefVersions.NodeJs = args.NodeJs
		}
		if args.Yarn == "" {
			tools.Yarn(&toolsDefVersions.Yarn)
		} else {
			toolsDefVersions.Yarn = args.Yarn
		}
		if args.Hosts == "" {
			tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
		} else {
			toolsDefVersions.Hosts = args.Hosts
		}
		if args.PwaBackendUrl == "" {
			setMagentoBackendHost(&toolsDefVersions.PwaBackendUrl, projectConf)
		} else {
			toolsDefVersions.PwaBackendUrl = args.PwaBackendUrl
		}
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
