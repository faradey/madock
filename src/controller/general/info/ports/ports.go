package ports

import (
	"fmt"
	"sort"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/ports"
)

type PortInfo struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
}

type PortsOutput struct {
	Project string     `json:"project"`
	Ports   []PortInfo `json:"ports"`
	// Published is false under ports/publish false. The registry still holds
	// the project's numbers — the shared proxy names its upstreams by them —
	// but nothing listens on any of them, and a script that connected to one
	// would conclude the service is down.
	Published bool `json:"published"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"info:ports"},
		JSONOutput: true,
		Handler:    Execute,
		Help:       "Show project ports. Supports --json (-j) output",
		Category:   "general",
		ArgsType:   new(arg_struct.ControllerGeneralInfoPorts),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralInfoPorts)).(*arg_struct.ControllerGeneralInfoPorts)

	projectName := configs.GetProjectName()
	if configs.GetCurrentProjectConfig()["ports/publish"] == "false" {
		if args.Json {
			output.PrintJSON(PortsOutput{Project: projectName, Ports: []PortInfo{}, Published: false})
			return
		}
		fmtc.TitleLn("Ports for project: " + projectName)
		fmtc.WarningLn("No port is published: ports/publish is false. The project is reached only through the shared proxy.")
		return
	}
	registry := ports.GetRegistry()
	projectPorts := registry.GetAllForProject(projectName)

	// Sort by service name
	var services []string
	for service := range projectPorts {
		services = append(services, service)
	}
	sort.Strings(services)

	var portInfos []PortInfo
	for _, service := range services {
		portInfos = append(portInfos, PortInfo{
			Service: service,
			Port:    projectPorts[service],
		})
	}

	if args.Json {
		portsOutput := PortsOutput{
			Project:   projectName,
			Ports:     portInfos,
			Published: true,
		}
		output.PrintJSON(portsOutput)
		return
	}

	// Text output
	fmtc.TitleLn("Ports for project: " + projectName)
	if len(portInfos) > 0 {
		for _, p := range portInfos {
			fmt.Printf("  %-25s %d\n", p.Service, p.Port)
		}
	} else {
		fmtc.WarningLn("No ports allocated yet. Run 'madock start' first.")
	}
}
