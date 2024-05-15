package arg_struct

import "github.com/faradey/madock/src/helper/cli/attr"

//TODO relocate here other ArgsStruct

type ControllerGeneralHelp struct {
	attr.Arguments
}

type ControllerGeneralSetup struct {
	attr.Arguments
	Download        bool   `arg:"-d,--download" help:"Download code from repository"`
	Install         bool   `arg:"-i,--install" help:"Install service (Magento, PWA, Shopify SDK, etc.)"`
	SampleData      bool   `arg:"-s,--sample-data" help:"Sample data"`
	Platform        string `arg:"--platform" help:"Platform"`
	PlatformEdition string `arg:"--edition" help:"Platform edition"`
	PlatformVersion string `arg:"--edition" help:"Platform version"`
	Php             string `arg:"--php" help:"PHP version"`
	Db              string `arg:"--db" help:"DB version"`
	Composer        string `arg:"--composer" help:"Composer version"`
	SearchEngine    string `arg:"--search-engine" help:"Search Engine"`
	Elastic         string `arg:"--elastic" help:"Elastic version"`
	OpenSearch      string `arg:"--opensearch" help:"OpenSearch version"`
	Redis           string `arg:"--redis" help:"Redis version"`
	RabbitMQ        string `arg:"--rabbitmq" help:"RabbitMQ version"`
	Hosts           string `arg:"--hosts" help:"Hosts"`
	NodeJs          string `arg:"--nodejs" help:"Node.js version"`
	Yarn            string `arg:"--yarn" help:"Yarn version"`
	PwaBackendUrl   string `arg:"--pwa-backend-url" help:"PWA backend url"`
}
