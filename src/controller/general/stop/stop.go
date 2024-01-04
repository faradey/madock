package stop

import (
	stopCustom "github.com/faradey/madock/src/controller/custom/stop"
	"github.com/faradey/madock/src/controller/general/proxy"
	stopMagento2 "github.com/faradey/madock/src/controller/magento/stop"
	stopPwa "github.com/faradey/madock/src/controller/pwa/stop"
	stopShopify "github.com/faradey/madock/src/controller/shopify/stop"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
)

func Execute() {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "magento2" {
		stopMagento2.Execute()
	} else if projectConf["platform"] == "pwa" {
		stopPwa.Execute()
	} else if projectConf["platform"] == "shopify" {
		stopShopify.Execute()
	} else if projectConf["platform"] == "custom" {
		stopCustom.Execute()
	}
	if len(paths.GetActiveProjects()) == 0 {
		proxy.Execute("stop")
	}
}
