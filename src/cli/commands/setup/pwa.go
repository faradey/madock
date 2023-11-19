package setup

import (
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs/projects"
	"github.com/faradey/madock/src/versions/pwa"
)

func PWA(projectName string, projectConf map[string]string, continueSetup bool) {
	if continueSetup {
		toolsDefVersions := pwa.GetVersions()
		NodeJs(&toolsDefVersions.NodeJs)
		Yarn(&toolsDefVersions.Yarn)
		Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
		setMagentoBackendHost(&toolsDefVersions.PwaBackendUrl, projectConf)
		projects.SetEnvForProject(projectName, toolsDefVersions, projectConf)
		fmtc.SuccessLn("\n" + "Finish set up environment")
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
	selected, _ := Waiter()
	if selected != "" {
		*defVersion = selected
		fmtc.SuccessLn("Your choice: " + *defVersion)
	}
}
