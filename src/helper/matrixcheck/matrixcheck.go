// Package matrixcheck warns when a project runs a service newer than its
// platform version supports, by madock's version matrix.
//
// The matrix gives, for each release of a platform, the newest version of each
// tool that release supports — Adobe's system requirements for Magento, the
// vendors' own for the rest. It has no floor: an older PHP than the matrix
// names may or may not work, and the files do not say. So the warning is for
// the one direction the matrix can answer, a version above its line, and for a
// tool the matrix calls NotCompatible with that release.
//
// A warning, never a refusal (Indemorio IND-6): whoever pinned a version past
// the line may know something the matrix does not, and a start that refused
// would take an environment down over a judgement call.
package matrixcheck

import (
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/detect"
	"github.com/faradey/madock/v4/src/model/versions"
)

// runsFunc says whether the project actually runs a service, given its config
// and what the matrix answers for its platform version.
type runsFunc func(conf map[string]string, want versions.ToolsVersions) bool

// service is one configured version the matrix has an answer for.
type service struct {
	key  string // the config key holding the version
	tool string // the matrix tool answering for it
	runs runsFunc
}

func enabled(key string) runsFunc {
	return func(conf map[string]string, _ versions.ToolsVersions) bool { return conf[key] == "true" }
}

func language(name string) runsFunc {
	return func(conf map[string]string, _ versions.ToolsVersions) bool { return conf["language"] == name }
}

// runsPHP: php is the language when none is named, which is what
// ResolveMainService falls back to.
func runsPHP(conf map[string]string, _ versions.ToolsVersions) bool {
	return conf["language"] == "" || conf["language"] == "php"
}

func runsNode(conf map[string]string, _ versions.ToolsVersions) bool {
	return conf["language"] == "nodejs" || conf["nodejs/enabled"] == "true"
}

// runsMatrixDB: the matrix's db versions are for its own engine, and a MySQL
// 8.4 read against a MariaDB line would be nonsense. A matrix naming no engine
// means MariaDB, as it does to setup (dbDisplayName's default).
func runsMatrixDB(conf map[string]string, want versions.ToolsVersions) bool {
	engine := want.DbType
	if engine == "" {
		engine = "MariaDB"
	}
	return conf["db/enabled"] == "true" && strings.EqualFold(dbType(conf), engine)
}

var services = []service{
	{"php/version", "php", runsPHP},
	{"php/composer/version", "composer", runsPHP},
	{"db/version", "db", runsMatrixDB},
	{"search/opensearch/version", "opensearch", enabled("search/opensearch/enabled")},
	{"search/elasticsearch/version", "elasticsearch", enabled("search/elasticsearch/enabled")},
	{"redis/version", "redis", enabled("redis/enabled")},
	{"valkey/version", "valkey", enabled("valkey/enabled")},
	{"rabbitmq/version", "rabbitmq", enabled("rabbitmq/enabled")},
	{"nodejs/version", "nodejs", runsNode},
	{"python/version", "python", language("python")},
	{"go/version", "golang", language("golang")},
	{"ruby/version", "ruby", language("ruby")},
}

// dbType names the engine the way the matrix does, read as setup reads it back.
func dbType(conf map[string]string) string {
	switch conf["db/type"] {
	case "mysql":
		if conf["db/repository"] == "mysql" {
			return "MySQL"
		}
		return "MariaDB"
	case "postgresql":
		return "PostgreSQL"
	case "mongodb":
		return "MongoDB"
	}
	return ""
}

// Outside lists, one line each, what the project runs past what its platform
// version supports. Empty when everything is within the matrix, or when the
// platform has no matrix.
func Outside(conf map[string]string, platformVersion string) []string {
	platform := conf["platform"]
	want, ok := versions.FromMatrix(platform, platformVersion)
	if !ok {
		return nil
	}

	var out []string
	for _, s := range services {
		have := strings.TrimSpace(conf[s.key])
		if have == "" || !s.runs(conf, want) {
			continue
		}
		line := versions.Answer(platform, s.tool, platformVersion)
		switch {
		case line == versions.NotCompatible:
			out = append(out, s.key+" is set, and the matrix says "+s.tool+" does not run with "+platform+" "+platformVersion)
		case line == "":
			// The matrix has nothing to say about this tool here.
		case !readable(line) || !readable(have):
			// An image reference ("postgres:17", medusa's db) or a tag such as
			// "latest" is no version to compare: Compare would put it below
			// every version and call anything configured "newer".
		case versions.Compare(majorMinor(have), majorMinor(line)) > 0:
			out = append(out, s.key+" "+have+" is newer than "+platform+" "+platformVersion+" supports ("+line+" by the matrix)")
		}
	}
	return out
}

// majorMinor keeps the first two segments: the matrix names lines, and
// OpenSearch 2.12.1 is on the 2.12 line Adobe lists as 2.12.0.
func majorMinor(v string) string {
	base, _, _ := strings.Cut(strings.TrimPrefix(strings.TrimSpace(v), "v"), "-")
	parts := strings.Split(base, ".")
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, ".")
}

// readable reports whether v reads as a version at all.
func readable(v string) bool {
	return versions.AtLeast(majorMinor(v), "0")
}

// detectVersion is the platform and its version found in the project's code;
// replaceable in tests.
var detectVersion = func(dir string) (platform, version string) {
	d := detect.Detect(dir)
	return d.Platform, d.PlatformVersion
}

// Warn prints Outside for the project in dir, on stderr so a JSON answer on
// stdout stays a document. It says nothing when the platform version cannot be
// read from the code, or the code is of another platform than the config
// names: then there is nothing to compare against.
func Warn(conf map[string]string, dir string) {
	platform, version := detectVersion(dir)
	if version == "" || platform != conf["platform"] {
		return
	}
	lines := Outside(conf, version)
	for _, line := range lines {
		fmtc.WarningErrLn("Outside the version matrix: " + line)
	}
	if len(lines) > 0 {
		fmtc.WarningErrLn("A warning only: madock starts it as configured. `madock matrix --platform " + platform + "` shows the lines.")
	}
}
