package status

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/cli/output"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"status"},
		Handler:  Execute,
		Help:     "Show container status. Supports --json (-j) output",
		Category: "general",
	})
}

type InfoStruct struct {
	Name    string `json:"Name"`
	Project string `json:"Project"`
	Service string `json:"Service"`
	State   string `json:"State"`
}

type StatusOutput struct {
	Services []ServiceStatus `json:"services"`
	Proxy    []ServiceStatus `json:"proxy"`
	Tools    ToolsStatus     `json:"tools"`
}

type ServiceStatus struct {
	Name    string `json:"name"`
	Service string `json:"service"`
	State   string `json:"state"`
	Running bool   `json:"running"`
}

type ToolsStatus struct {
	CronEnabled     bool `json:"cron_enabled"`
	DebuggerEnabled bool `json:"debugger_enabled"`
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralStatus)).(*arg_struct.ControllerGeneralStatus)

	projectName := configs.GetProjectName()
	pp := paths.NewProjectPaths(projectName)

	// Get services status
	servicesData := getContainerStatus(pp.DockerCompose())

	// Get proxy status
	proxyData := getContainerStatus(paths.ProxyDockerCompose())

	// Get tools status
	projectConf := configs.GetCurrentProjectConfig()
	toolsStatus := ToolsStatus{
		CronEnabled:     strings.ToLower(projectConf["cron/enabled"]) == "true",
		DebuggerEnabled: strings.ToLower(projectConf["php/xdebug/enabled"]) == "true",
	}

	if args.Json {
		statusOutput := StatusOutput{
			Services: servicesData,
			Proxy:    proxyData,
			Tools:    toolsStatus,
		}
		output.PrintJSON(statusOutput)
		return
	}

	// Text output
	fmtc.TitleLn("Services:")
	if len(servicesData) > 0 {
		for _, val := range servicesData {
			row := fmt.Sprintf("%s %s", val.Service, val.State)
			if val.Running {
				fmtc.SuccessLn(row)
			} else {
				fmtc.WarningLn(row)
			}
		}
	} else {
		fmtc.WarningLn("No services found")
	}

	fmtc.TitleLn("Proxy:")
	if len(proxyData) > 0 {
		for _, val := range proxyData {
			row := fmt.Sprintf(" %s %s", val.Service, val.State)
			if val.Running {
				fmtc.SuccessLn(row)
			} else {
				fmtc.WarningLn(row)
			}
		}
	} else {
		fmtc.WarningLn("No services found")
	}

	fmtc.TitleLn("Tools:")
	if toolsStatus.CronEnabled {
		fmtc.SuccessLn(" Cron is running")
	} else {
		fmtc.WarningLn(" Cron is not running")
	}

	if toolsStatus.DebuggerEnabled {
		fmtc.SuccessLn(" Debugger is enabled")
	} else {
		fmtc.WarningLn(" Debugger is disabled")
	}
}

func getContainerStatus(composePath string) []ServiceStatus {
	cmd := exec.Command("docker", "compose", "-f", composePath, "ps", "--format", "json")
	result, err := cmd.CombinedOutput()
	if err != nil {
		logger.Fatal(err)
	}

	var statusData []ServiceStatus
	if len(result) > 0 {
		result = parseJson(result)
		var rawData []InfoStruct
		err = json.Unmarshal(result, &rawData)
		if err != nil {
			return statusData
		}
		for _, val := range rawData {
			statusData = append(statusData, ServiceStatus{
				Name:    val.Name,
				Service: val.Service,
				State:   val.State,
				Running: val.State == "running",
			})
		}
	}
	return statusData
}

func parseJson(data []byte) []byte {
	str := strings.TrimSpace(string(data))
	if strings.Contains(str, "}{") || strings.Contains(str, "}\n{") {
		str = strings.ReplaceAll(str, "}\n{", "}{")
		str = "[" + strings.ReplaceAll(str, "}{", "},{") + "]"
	}

	return []byte(str)
}
