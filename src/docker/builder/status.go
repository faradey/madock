package builder

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

type StatusInfoStruct struct {
	Name    string `json:"Name"`
	Project string `json:"Project"`
	Service string `json:"Service"`
	State   string `json:"State"`
}

func Status() {
	projectName := configs.GetProjectName()
	cmd := exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "ps", "--format", "json")
	result, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	statusData := []StatusInfoStruct{}
	err = json.Unmarshal([]byte(result), &statusData)
	if err != nil {
		fmt.Println(err)
	}
	fmtc.TitleLn("Services:")
	for _, val := range statusData {
		row := fmt.Sprintf("%s %s", val.Service, val.State)
		if val.State == "running" {
			fmtc.SuccessLn(row)
		} else {
			fmtc.WarningLn(row)
		}
	}
	fmtc.TitleLn("Proxy:")
	cmd = exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "ps", "--format", "json")
	result, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	statusData = []StatusInfoStruct{}
	err = json.Unmarshal([]byte(result), &statusData)
	if err != nil {
		fmt.Println(err)
	}
	for _, val := range statusData {
		row := fmt.Sprintf(" %s %s", val.Service, val.State)
		if val.State == "running" {
			fmtc.SuccessLn(row)
		} else {
			fmtc.WarningLn(row)
		}
	}
	fmtc.TitleLn("Tools:")
	projectConfig := configs.GetCurrentProjectConfig()

	if strings.ToLower(projectConfig["CRON_ENABLED"]) == "true" {
		fmtc.SuccessLn(" Cron is running")
	} else {
		fmtc.WarningLn(" Cron is not running")
	}

	if strings.ToLower(projectConfig["XDEBUG_ENABLED"]) == "true" {
		fmtc.SuccessLn(" Debugger is enabled")
	} else {
		fmtc.WarningLn(" Debugger is disabled")
	}
}
