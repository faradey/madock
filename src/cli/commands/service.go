package commands

import (
	"github.com/faradey/madock/src/docker/service"
)

func ServiceList() {
	service.ServiceList()
}

func ServiceEnable() {
	service.ServiceEnable()
	Rebuild()
}

func ServiceDisable() {
	service.ServiceDisable()
	Rebuild()
}
