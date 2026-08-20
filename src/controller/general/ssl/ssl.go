package ssl

import (
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v4/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"ssl:rebuild"},
		Handler:  Execute,
		Help:     "Rebuild SSL certificates",
		Category: "general",
	})
}

func Execute() {
	ctxPath := paths.MakeDirsByPath(paths.CtxDir())
	nginx.GenerateSslCert(ctxPath, true)
}
