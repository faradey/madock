package stop

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/helper/cli/attr"
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
		builder.StopMagento2()
	} else if projectConf["PLATFORM"] == "pwa" {
		builder.StopPWA()
	} else if projectConf["PLATFORM"] == "shopify" {
		builder.StopShopify()
	} else if projectConf["PLATFORM"] == "custom" {
		builder.StopCustom()
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
