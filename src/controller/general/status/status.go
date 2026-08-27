package status

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"status"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "Show container status. Supports --json (-j) output",
		Category:   "general",
		ArgsType:   new(arg_struct.ControllerGeneralStatus),
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
	// Orphan marks a container that belongs to this compose project but is not
	// among the services its files declare — left behind by an earlier version
	// of the configuration. Absent for the ordinary case, so a script reading
	// this can treat its presence as the exception it is.
	Orphan bool `json:"orphan,omitempty"`
}

type ToolsStatus struct {
	CronEnabled bool `json:"cron_enabled"`
	CronRunning bool `json:"cron_running"`
	// CronJobs is how many jobs the container's crontab actually holds, and
	// CronJobsKnown whether that could be established. A running daemon with an
	// empty crontab reads as healthy and runs nothing; -1 means the question
	// could not be asked, which is never rounded down to none.
	CronJobs        int  `json:"cron_jobs"`
	CronJobsKnown   bool `json:"cron_jobs_known"`
	DebuggerEnabled bool `json:"debugger_enabled"`
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralStatus)).(*arg_struct.ControllerGeneralStatus)

	projectName := configs.GetProjectName()
	pp := paths.NewProjectPaths(projectName)

	// Get services status
	servicesData := getContainerStatus(pp.DockerCompose())
	knownServices, servicesKnown := definedServices(pp.DockerCompose(), pp.DockerComposeOverride())
	servicesData = markOrphans(servicesData, knownServices, servicesKnown)

	// Get proxy status
	proxyData := getContainerStatus(paths.ProxyDockerCompose())
	// The proxy has the same problem and a documented instance of it: disabling
	// mailpit leaves its container behind, and it went on being listed as one of
	// the proxy's services.
	knownProxy, proxyKnown := definedServices(paths.ProxyDockerCompose(), "")
	proxyData = markOrphans(proxyData, knownProxy, proxyKnown)

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
	if toolsStatus.CronRunning {
		toolsStatus.CronJobs, toolsStatus.CronJobsKnown = docker.CronJobCount(projectName)
	}
	if !toolsStatus.CronJobsKnown {
		toolsStatus.CronJobs = -1
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
			printServiceRow(val, "")
		}
	} else {
		fmtc.WarningLn("No services found")
	}

	fmtc.TitleLn("Proxy:")
	if len(proxyData) > 0 {
		for _, val := range proxyData {
			printServiceRow(val, " ")
		}
	} else {
		fmtc.WarningLn("No services found")
	}

	fmtc.TitleLn("Tools:")
	if toolsStatus.CronRunning {
		// "Cron is running" answers a question nobody asks. What an operator
		// wants to know is whether anything is scheduled, and the daemon can be
		// up with an empty crontab — which is exactly the state a live project
		// sat in, looking healthy, running nothing.
		switch {
		case !toolsStatus.CronJobsKnown:
			fmtc.SuccessLn(" Cron is running (installed jobs: unknown)")
		case toolsStatus.CronJobs == 0:
			fmtc.WarningLn(" Cron is running but no jobs are installed — nothing runs on a schedule")
		case toolsStatus.CronJobs == 1:
			fmtc.SuccessLn(" Cron is running (1 job)")
		default:
			fmtc.SuccessLn(fmt.Sprintf(" Cron is running (%d jobs)", toolsStatus.CronJobs))
		}
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

// printServiceRow writes one line of the human output.
//
// An orphan is printed as a warning whatever its state, and said in words rather
// than left to be inferred from a name the reader is not expecting: a container
// running here is not the project working, it is the previous configuration
// still running, and those two look identical in a list.
func printServiceRow(val ServiceStatus, indent string) {
	row := fmt.Sprintf("%s%s %s", indent, val.Service, val.State)
	if val.Orphan {
		fmtc.WarningLn(row + " — orphan: not in this project's configuration any more")
		return
	}
	if val.Running {
		fmtc.SuccessLn(row)
	} else {
		fmtc.WarningLn(row)
	}
}

func getContainerStatus(composePath string) []ServiceStatus {
	cmd := exec.Command("docker", "compose", "-f", composePath, "ps", "--format", "json")

	// Separate streams, not CombinedOutput.
	//
	// The error message below needs docker's own words, and CombinedOutput was
	// how it got them — but it also folded docker's warnings into the JSON.
	// Compose writes those to stderr and the data to stdout, so a compose file
	// still carrying the obsolete top-level `version` key put
	// "the attribute `version` is obsolete, it will be ignored" in front of the
	// first object, and every single `madock status` on that project printed
	// "Could not read the container status: invalid character 'i' in literal
	// true (expecting 'r')" — a JSON parser complaining about English.
	//
	// Keeping stderr in its own buffer answers both: the parser sees only data,
	// and the failure message still quotes docker.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	result, err := cmd.Output()
	if err != nil {
		// The output, not just the exit code. `logger.Fatal(err)` printed
		// "exit status 1" and threw away the only sentence that said why —
		// docker's own error, which is in the combined output. On a server
		// where every project answered that way, the message was equally
		// consistent with a missing compose file, a daemon that is not running
		// and a compose file docker refuses to parse, and there was no way to
		// tell from madock at all.
		logger.Fatal(fmt.Errorf("docker compose ps failed for %s: %w\n%s%s", composePath, err, string(result), stderr.String()))
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
		// Only objects. Every line compose prints here is one, so anything else
		// is docker talking rather than answering — a deprecation warning, a
		// credential-helper notice, an update banner. The caller now reads
		// stdout alone, which keeps those on stderr where they belong, but a
		// warning has reached stdout before and the cost of ignoring one is a
		// line of prose, while the cost of parsing one is the whole status.
		if strings.HasPrefix(line, "{") {
			objects = append(objects, line)
		}
	}

	return []byte("[" + strings.Join(objects, ",") + "]")
}
