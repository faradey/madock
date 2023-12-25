package service

import (
	"github.com/faradey/madock/src/helper/configs"
	"log"
	"strings"
)

func IsService(name string) bool {
	upperName := strings.ToLower(name)
	configData := configs.GetCurrentProjectConfig()

	for key := range configData {
		serviceName := strings.SplitN(key, "/enabled", 2)
		if serviceName[0] == upperName {
			return true
		}
	}

	log.Fatalln("The service \"" + name + "\" doesn't exist.")

	return false
}
