package commands

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/docker/builder"
)

func IsNotDefine() {
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}

func Ssl() {
	builder.SslRebuild()
}

func Shopify(flag string) {
	builder.Shopify(flag)
}

func ShopifyWeb(flag string) {
	builder.ShopifyWeb(flag)
}

func ShopifyWebFrontend(flag string) {
	builder.ShopifyWebFrontend(flag)
}
