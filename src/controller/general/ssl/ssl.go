package ssl

import (
	"github.com/faradey/madock/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/src/helper/paths"
)

func Execute() {
	ctxPath := paths.MakeDirsByPath(paths.CtxDir())
	nginx.GenerateSslCert(ctxPath, true)
}
