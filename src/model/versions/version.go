package versions

type ToolsVersions struct {
	Platform,
	Language,
	Php,
	DbType,
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
	PlatformVersion,
	NodeJs,
	Yarn,
	Python,
	Golang,
	Ruby string
}

// GetXdebugVersion is the Xdebug that goes with a PHP version, from
// matrix/common.xml.
func GetXdebugVersion(phpVer string) string {
	return AnswerFor("common", "xdebug", phpVer)
}
