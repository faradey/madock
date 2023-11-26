package list

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
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

	for _, key := range keys {
		serviceName := strings.SplitN(key, "_ENABLED", 2)
		if serviceName[0] != key {
			fmtc.Title(strings.ToLower(serviceName[0]))
			if configData[key] == "true" {
				fmtc.SuccessLn(" enabled")
			} else {
				fmtc.WarningLn(" disabled")
			}
		}
	}
}
