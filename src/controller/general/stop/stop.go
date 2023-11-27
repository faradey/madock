package stop

import (
	stopCustom "github.com/faradey/madock/src/controller/custom/stop"
	"github.com/faradey/madock/src/controller/general/proxy"
	stopMagento2 "github.com/faradey/madock/src/controller/magento/stop"
	stopPwa "github.com/faradey/madock/src/controller/pwa/stop"
	stopShopify "github.com/faradey/madock/src/controller/shopify/stop"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Execute() {
	getArgs()
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "magento2" {
		stopMagento2.Execute()
	} else if projectConf["PLATFORM"] == "pwa" {
		stopPwa.Execute()
	} else if projectConf["PLATFORM"] == "shopify" {
		stopShopify.Execute()
	} else if projectConf["PLATFORM"] == "custom" {
		stopCustom.Execute()
	}
	if len(paths.GetActiveProjects()) == 0 {
		proxy.Execute("stop")
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
