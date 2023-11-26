package status

import (
	"encoding/json"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os/exec"
	"strings"
)

type InfoStruct struct {
	Name    string `json:"Name"`
	Project string `json:"Project"`
	Service string `json:"Service"`
	State   string `json:"State"`
}

func Execute() {
	projectName := configs.GetProjectName()
	cmd := exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "ps", "--format", "json")
	result, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	if len(result) > 0 {
		result = parseJson(result)
		var statusData []InfoStruct
		err = json.Unmarshal(result, &statusData)
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
	} else {
		fmtc.WarningLn("No services found")
	}

	fmtc.TitleLn("Proxy:")
	cmd = exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "ps", "--format", "json")
	result, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	if len(result) > 0 {
		result = parseJson(result)
		var statusData []InfoStruct
		err = json.Unmarshal(result, &statusData)
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
	} else {
		fmtc.WarningLn("No services found")
	}

	fmtc.TitleLn("Tools:")
	projectConf := configs.GetCurrentProjectConfig()

	if strings.ToLower(projectConf["CRON_ENABLED"]) == "true" {
		fmtc.SuccessLn(" Cron is running")
	} else {
		fmtc.WarningLn(" Cron is not running")
	}

	if strings.ToLower(projectConf["XDEBUG_ENABLED"]) == "true" {
		fmtc.SuccessLn(" Debugger is enabled")
	} else {
		fmtc.WarningLn(" Debugger is disabled")
	}
}

func parseJson(data []byte) []byte {
	str := strings.TrimSpace(string(data))
	if strings.Contains(str, "}{") || strings.Contains(str, "}\n{") {
		str = strings.ReplaceAll(str, "}\n{", "}{")
		str = "[" + strings.ReplaceAll(str, "}{", "},{") + "]"
	}

	return []byte(str)
}
