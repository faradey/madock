package ports

import (
	"fmt"
	"sort"

	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/cli/output"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/ports"
)

type PortInfo struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
}

type PortsOutput struct {
	Project string     `json:"project"`
	Ports   []PortInfo `json:"ports"`
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralInfoPorts)).(*arg_struct.ControllerGeneralInfoPorts)

	projectName := configs.GetProjectName()
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
			Project: projectName,
			Ports:   portInfos,
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
