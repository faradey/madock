package versions

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"os"
)

func V240() {
	execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	projectName := ""
	envFile := ""
	for _, dir := range execProjectsDirs {
		if paths.IsFileExist(execPath + dir + "/env.txt") {
			if paths.IsFileExist(execPath + dir + "/config.xml") {
				os.Rename(execPath+dir+"/config.xml", execPath+dir+"/config.xml.old")
			}
			projectName = dir
			projectConfOnly := configs.GetProjectConfigOnly(projectName)
			projectConf := configs.GetProjectConfig(projectName)
			envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
		}
	}
}

/*func composeXmlFile(projectPath string) bool {

}
*/
