package commands

import (
	"strings"

	"github.com/faradey/madock/src/docker/service"
)

func SwitchService(name, action string) {
	action = strings.ToLower(action)
	if name == "list" {
		service.ServiceList()
	} else if action == "on" {
		service.ServiceOn(name)
	} else if action == "off" {
		service.ServiceOff(name)
	}
	Rebuild()
}
