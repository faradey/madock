package list

import (
	"sort"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	service2 "github.com/faradey/madock/v3/src/controller/general/service"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/cli/output"
	"github.com/faradey/madock/v3/src/helper/configs"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"service:list"},
		Handler:  Execute,
		Help:     "List services. Supports --json (-j) output",
		Category: "service",
	})
}

type ServiceListOutput struct {
	Services []ServiceInfo `json:"services"`
}

type ServiceInfo struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralServiceList)).(*arg_struct.ControllerGeneralServiceList)

	configData := configs.GetCurrentProjectConfig()
	keys := make([]string, 0, len(configData))
	for k := range configData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var services []ServiceInfo
	for _, key := range keys {
		serviceName := strings.SplitN(key, "/enabled", 2)
		if serviceName[0] != key {
			service := service2.GetByLong(serviceName[0])
			enabled := configData[key] == "true"
			services = append(services, ServiceInfo{
				Name:    service,
				Enabled: enabled,
			})
		}
	}

	if args.Json {
		output.PrintJSON(ServiceListOutput{Services: services})
		return
	}

	for _, svc := range services {
		fmtc.Title(svc.Name)
		if svc.Enabled {
			fmtc.SuccessLn(" enabled")
		} else {
			fmtc.WarningLn(" disabled")
		}
	}
}
