package commands

import (
	"github.com/faradey/madock/src/controller/def/rebuild"
	"github.com/faradey/madock/src/docker/service"
)

func ServiceList() {
	service.ServiceList()
}

func ServiceEnable() {
	service.Enable()
	rebuild.Execute()
}

func ServiceDisable() {
	service.Disable()
	rebuild.Execute()
}
