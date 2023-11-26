package service

import (
	"github.com/faradey/madock/src/configs"
	"log"
	"strings"
)

func IsService(name string) bool {
	upperName := strings.ToUpper(name)
	configData := configs.GetCurrentProjectConfig()

	for key := range configData {
		serviceName := strings.SplitN(key, "_ENABLED", 2)
		if serviceName[0] == upperName {
			return true
		}
	}

	log.Fatalln("The service \"" + name + "\" doesn't exist.")

	return false
}
