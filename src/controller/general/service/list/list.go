package list

import (
	service2 "github.com/faradey/madock/src/controller/general/service"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"sort"
	"strings"
)

func Execute() {
	configData := configs.GetCurrentProjectConfig()
	keys := make([]string, 0, len(configData))
	for k := range configData {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	service := ""
	for _, key := range keys {
		serviceName := strings.SplitN(key, "/enabled", 2)
		if serviceName[0] != key {
			service = service2.GetByLong(serviceName[0])
			fmtc.Title(service)
			if configData[key] == "true" {
				fmtc.SuccessLn(" enabled")
			} else {
				fmtc.WarningLn(" disabled")
			}
		}
	}
}
