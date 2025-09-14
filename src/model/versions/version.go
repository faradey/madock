package versions

type ToolsVersions struct {
	Platform,
	Php,
	Db,
	SearchEngine,
	Elastic,
	OpenSearch,
	Composer,
	Redis,
	Valkey,
	RabbitMQ,
	Xdebug,
	Hosts,
	PwaBackendUrl,
	PlatformVersion,
	NodeJs,
	Yarn string
}

func GetXdebugVersion(phpVer string) string {
	if phpVer >= "8.4" {
		return "3.4.4"
	} else if phpVer >= "8.3" {
		return "3.3.1"
	} else if phpVer >= "8.1" {
		return "3.2.2"
	} else if phpVer >= "7.2" {
		return "3.1.6"
	}

	return "2.7.2"
}
