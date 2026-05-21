// Package info registers the Magento 2 handler for the "info" command.
// It delegates to the existing magento-info.php script which inspects the
// project's composer.json/lock and prints Magento + third-party module data.
package info

import (
	"github.com/faradey/madock/v3/src/helper/docker"
	inforeg "github.com/faradey/madock/v3/src/info"
)

type Handler struct{}

func (h *Handler) Print(ctx *inforeg.InfoContext) error {
	containerName := docker.GetContainerName(ctx.ProjectConf, ctx.ProjectName, ctx.Service)
	return docker.ContainerExec(containerName, "", true, "php", "/var/www/scripts/php/magento-info.php", ctx.ProjectConf["workdir"])
}

func init() {
	inforeg.Register("magento2", &Handler{})
}
