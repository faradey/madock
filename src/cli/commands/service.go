package commands

import (
	"github.com/faradey/madock/src/docker/service"
)

func ServiceList() {
	service.ServiceList()
}

func ServiceEnable() {
	service.Enable()
	Rebuild()
}

func ServiceDisable() {
	service.Disable()
	Rebuild()
}
