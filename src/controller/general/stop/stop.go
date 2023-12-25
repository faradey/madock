package stop

import (
	"github.com/alexflint/go-arg"
	stopCustom "github.com/faradey/madock/src/controller/custom/stop"
	"github.com/faradey/madock/src/controller/general/proxy"
	stopMagento2 "github.com/faradey/madock/src/controller/magento/stop"
	stopPwa "github.com/faradey/madock/src/controller/pwa/stop"
	stopShopify "github.com/faradey/madock/src/controller/shopify/stop"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Execute() {
	getArgs()
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

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
