package status

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/cli/output"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"status"},
		Handler:  Execute,
		Help:     "Show container status. Supports --json (-j) output",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralStatus),
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
	CronRunning     bool `json:"cron_running"`
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
		// Two different questions, and they can disagree. cron_enabled is what
		// the configuration asks for; cron_running is what the container has.
		// Starting cron is a command that can fail, and reporting the setting as
		// if it were the outcome is how a project ends up with nothing on a
		// schedule and a status that says otherwise.
		CronEnabled:     strings.ToLower(projectConf["cron/enabled"]) == "true",
		CronRunning:     docker.CronRunning(projectName),
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
	if toolsStatus.CronRunning {
		fmtc.SuccessLn(" Cron is running")
	} else if toolsStatus.CronEnabled {
		fmtc.WarningLn(" Cron is enabled but not running")
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
		// The output, not just the exit code. `logger.Fatal(err)` printed
		// "exit status 1" and threw away the only sentence that said why —
		// docker's own error, which is in the combined output. On a server
		// where every project answered that way, the message was equally
		// consistent with a missing compose file, a daemon that is not running
		// and a compose file docker refuses to parse, and there was no way to
		// tell from madock at all.
		logger.Fatal(fmt.Errorf("docker compose ps failed for %s: %w\n%s", composePath, err, string(result)))
	}

	var statusData []ServiceStatus
	if len(result) > 0 {
		result = parseJson(result)
		var rawData []InfoStruct
		err = json.Unmarshal(result, &rawData)
		if err != nil {
			// Reported rather than swallowed. This returned an empty list in
			// silence, which reads as "nothing is running" — the one answer a
			// status command must never give when it does not know.
			fmtc.WarningLn("Could not read the container status: " + err.Error())
			logger.Println(err, string(result))
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

// parseJson turns what `docker compose ps --format json` prints into a JSON
// array.
//
// Compose emits one object per line rather than an array, so the count decides
// the shape: two services look like `{…}\n{…}` and one looks like `{…}`. This
// used to wrap only when it found a `}{` boundary, so a stack with exactly one
// service was handed to the array decoder as a bare object, failed to decode,
// and — because the error was swallowed — was reported as no services at all.
//
// Seen in the field: disabling mailpit left the proxy with nginx alone, and
// `status` began answering "Proxy: No services found" while every site it was
// serving stayed up.
func parseJson(data []byte) []byte {
	str := strings.TrimSpace(string(data))
	if str == "" {
		return []byte("[]")
	}

	// Some versions print a real array already.
	if strings.HasPrefix(str, "[") {
		return []byte(str)
	}

	var objects []string
	for _, line := range strings.Split(str, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			objects = append(objects, line)
		}
	}

	return []byte("[" + strings.Join(objects, ",") + "]")
}
