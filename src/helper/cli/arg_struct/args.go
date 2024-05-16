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
	PlatformEdition string `arg:"--platform-edition" help:"Platform edition"`
	PlatformVersion string `arg:"--platform-version" help:"Platform version"`
	Php             string `arg:"--php" help:"PHP version"`
	Db              string `arg:"--db" help:"DB version"`
	Composer        string `arg:"--composer" help:"Composer version"`
	SearchEngine    string `arg:"--search-engine" help:"Search Engine"`
	Elastic         string `arg:"--elastic" help:"Elasticsearch version"`
	OpenSearch      string `arg:"--opensearch" help:"OpenSearch version"`
	Redis           string `arg:"--redis" help:"Redis version"`
	RabbitMQ        string `arg:"--rabbitmq" help:"RabbitMQ version"`
	Hosts           string `arg:"--hosts" help:"Hosts"`
	NodeJs          string `arg:"--nodejs" help:"Node.js version"`
	Yarn            string `arg:"--yarn" help:"Yarn version"`
	PwaBackendUrl   string `arg:"--pwa-backend-url" help:"PWA backend url"`
}

type ControllerGeneralStart struct {
	attr.Arguments
	WithChown bool `arg:"-c,--with-chown" help:"With Chown"`
}

type ControllerGeneralSnapshotCreate struct {
	attr.Arguments
	Name string `arg:"-n,--name" help:"Name"`
}

type ControllerGeneralSetupEnv struct {
	attr.Arguments
	Force bool   `arg:"-f,--force" help:"Force"`
	Host  string `arg:"-h,--host" help:"Host"`
}

type ControllerGeneralServiceEnable struct {
	attr.ArgumentsWithArgs
	Global bool `arg:"-g,--global" help:"Global"`
}

type ControllerGeneralServiceDisable struct {
	attr.ArgumentsWithArgs
	Global bool `arg:"-g,--global" help:"Global"`
}

type ControllerGeneralBash struct {
	attr.Arguments
	Service string `arg:"-s,--service" help:"Service name (php, nginx, db, etc.)"`
	User    string `arg:"-u,--user" help:"User"`
}

type ControllerGeneralCleanCache struct {
	attr.Arguments
	User string `arg:"-u,--user" help:"User"`
}

type ControllerGeneralConfig struct {
	attr.Arguments
	Name  string `arg:"-n,--name" help:"Parameter name"`
	Value string `arg:"-v,--value" help:"Parameter value"`
}

type ControllerGeneralDbExport struct {
	attr.Arguments
	Name          string   `arg:"-n,--name" help:"Name of the archive file"`
	DBServiceName string   `arg:"-s,--service" help:"DB service name. For example: db"`
	IgnoreTable   []string `arg:"--ignore-table" help:"Ignore db table"`
	User          string   `arg:"-u,--user" help:"Ignore db table"`
}

type ControllerGeneralDbImport struct {
	attr.Arguments
	Force         bool   `arg:"-f,--force" help:"Install Magento"`
	DBServiceName string `arg:"-s,--service" help:"DB service name. For example: db"`
	User          string `arg:"-u,--user" help:"User"`
}
