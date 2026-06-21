package db

import (
	"encoding/json"
	"fmt"
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/remote_sync"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"remote:sync:db"},
		Handler:  Execute,
		Help:     "Sync remote database",
		Category: "remote",
		ArgsType: new(arg_struct.ControllerGeneralRemoteSyncDb),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralRemoteSyncDb)).(*arg_struct.ControllerGeneralRemoteSyncDb)

	projectConf := configs.GetCurrentProjectConfig()
	sshType := "ssh"
	if args.SshType != "" {
		sshType += "_" + args.SshType
	}
	siteRootPath := projectConf[sshType+"/site_root_path"]
	if _, ok := projectConf[sshType+"/site_root_path"]; !ok {
		siteRootPath = projectConf["ssh/site_root_path"]
	}
	conn := remote_sync.Connect(projectConf, sshType)

	// remoteDir is the madock project root (deploy root): the native madock export
	// runs there so the project name is derived from it (the container is
	// madock_<project>-db-1). docRoot is the platform's document root, which on a
	// Deployer/Capistrano layout lives under <root>/current — that is where the
	// cred files (env.php, .env, wp-config.php, ...) actually are.
	remoteDir := siteRootPath
	docRoot := remoteDir
	if out, err := remote_sync.RunCommandSafe(conn, "[ -e '"+remoteDir+"/current' ] && echo current"); err == nil && strings.TrimSpace(out) == "current" {
		docRoot = remoteDir + "/current"
	}
	name := args.Name
	defer func(conn *ssh.Client) {
		err := conn.Close()
		if err != nil {
			logger.Fatal(err)
		}
	}(conn)
	fmt.Println("")
	fmt.Println("Dumping and downloading DB is started")
	result := ""
	if args.DbUser != "" && args.DbPassword != "" && args.DbName != "" {
		if args.DbPort == "" {
			args.DbPort = "3306"
		}
		if args.DbHost == "" {
			args.DbHost = "localhost"
		}
		result = "{\"host\":\"" + args.DbHost + "\",\"dbname\":\"" + args.DbName + "\",\"username\":\"" + args.DbUser + "\",\"password\":\"" + args.DbPassword + "\",\"port\":\"" + args.DbPort + "\"}"
	} else if tryRemoteMadockExport(conn, remoteDir, name, args) {
		// madock is installed on the remote host: the dump was produced inside the
		// remote container (no PHP/mysqldump needed on the host) and downloaded.
		return
	} else {
		result = getRemoteDbCredsJSON(conn, docRoot, projectConf["platform"])
	}

	nOpenBrace := strings.Index(result, "{")
	if nOpenBrace != -1 {
		result = result[nOpenBrace:]
	} else {
		if strings.TrimSpace(result) != "" {
			fmt.Println(result)
		}
		logger.Fatal(fmt.Sprintf("Could not determine the remote database credentials for platform %q.\nNo readable config file was found on the remote host (checked the platform's standard location under ssh/site_root_path).\nProvide them explicitly with --db-host/--db-port/--db-user/--db-password/--db-name, or install madock on the remote host.", projectConf["platform"]))
	}
	if len(result) > 2 {
		dbAuthData := remote_sync.RemoteDbStruct{}
		err := json.Unmarshal([]byte(result), &dbAuthData)
		if err != nil {
			fmt.Println(err)
		}
		curDateTime := time.Now().Format("2006-01-02_15-04-05")
		name = strings.TrimSpace(name)
		if len(name) > 0 {
			name += "_"
		}

		dbType := configs.GetDbType(projectConf)
		var dumpName string
		var dumpCmd string

		switch dbType {
		case "postgresql":
			dumpName = "remote_" + name + curDateTime + ".sql.gz"
			dbPort := ""
			if dbAuthData.Port != "" {
				dbPort = " -p " + dbAuthData.Port
			}
			ignoreTablesStr := ""
			ignoreTables := args.IgnoreTable
			if len(ignoreTables) > 0 {
				for _, t := range ignoreTables {
					ignoreTablesStr += " --exclude-table=" + t
				}
			}
			dumpCmd = "PGPASSWORD=\"" + dbAuthData.Password + "\" pg_dump -U \"" + dbAuthData.Username + "\" -h " + dbAuthData.Host + dbPort + ignoreTablesStr + " " + dbAuthData.Dbname + " | gzip > /tmp/" + dumpName
		case "mongodb":
			dumpName = "remote_" + name + curDateTime + ".archive.gz"
			dbPort := ""
			if dbAuthData.Port != "" {
				dbPort = " --port=" + dbAuthData.Port
			}
			dumpCmd = "mongodump --username=\"" + dbAuthData.Username + "\" --password=\"" + dbAuthData.Password + "\" --host=" + dbAuthData.Host + dbPort + " --authenticationDatabase=admin --db=" + dbAuthData.Dbname + " --archive --gzip > /tmp/" + dumpName
		default:
			dumpName = "remote_" + name + curDateTime + ".sql.gz"
			ignoreTablesStr := ""
			ignoreTables := args.IgnoreTable
			if len(ignoreTables) > 0 {
				ignoreTablesStr = " --ignore-table=" + dbAuthData.Dbname + "." + strings.Join(ignoreTables, " --ignore-table="+dbAuthData.Dbname+".")
			}
			dbPort := ""
			if dbAuthData.Port != "" {
				dbPort = " -P " + dbAuthData.Port
			}
			dumpCmd = "mysqldump -u \"" + dbAuthData.Username + "\" -p\"" + dbAuthData.Password + "\" -h " + dbAuthData.Host + dbPort + " --quick --lock-tables=false --no-tablespaces --triggers" + ignoreTablesStr + " " + dbAuthData.Dbname + " | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/' | gzip > /tmp/" + dumpName
		}

		// Run the dump under `bash -o pipefail` so a failing dumper (missing
		// mysqldump/pg_dump, auth error, unreachable host) propagates a non-zero
		// exit instead of being masked by the trailing `| gzip`, which would
		// otherwise leave a valid but empty 20-byte gzip archive.
		result = remote_sync.RunCommand(conn, "bash -o pipefail -c "+shellSingleQuote(dumpCmd))
		sc, err := sftp.NewClient(conn)
		if err != nil {
			logger.Fatal(err)
		}
		defer func(sc *sftp.Client) {
			err = sc.Close()
			if err != nil {
				logger.Fatal(err)
			}
		}(sc)
		execPath := paths.GetExecDirPath()
		projectName := configs.GetProjectName()
		localPath := execPath + "/projects/" + projectName + "/backup/db/" + dumpName
		err = remote_sync.DownloadFile(sc, "/tmp/"+dumpName, localPath, false, false)
		if err != nil {
			logger.Fatal(err)
		}
		remote_sync.RunCommand(conn, "rm "+"/tmp/"+dumpName)
		assertDumpNotEmpty(localPath, "remote "+dbType+" client on host "+dbAuthData.Host)
		fmt.Println("")
		fmtc.SuccessLn("A database dump was created and saved locally. To import a database dump locally run the command `madock db:import`")
	} else {
		fmt.Println("Failed to get database authentication data (row 110)")
	}
}

// madockExportOutput mirrors the JSON emitted by `madock db:export --json` on the
// remote host. The command prints through the shared output.PrintJSON wrapper, so
// the payload is {"success":true,"data":{"file":"..."}} — the file path lives under
// "data", not at the top level.
type madockExportOutput struct {
	Success bool `json:"success"`
	Data    struct {
		File string `json:"file"`
	} `json:"data"`
}

// dumpPathRe matches an absolute remote dump path emitted by `madock db:export`.
// Used as a fallback for older remote madock versions whose db:export ignores
// --json and prints verbose dumper logging followed by the bare file path.
var dumpPathRe = regexp.MustCompile(`/\S+\.(?:sql\.gz|archive\.gz|sql)`)

// parseMadockExportFile extracts the dump file path from the output of
// `madock db:export --json` on the remote host. A current remote madock wraps
// the path in JSON via output.PrintJSON ({"success":true,"data":{"file":"..."}}),
// possibly preceded by verbose dumper logging on the combined ssh stream, so the
// JSON object is located by its outer braces. An older remote madock ignores
// --json and prints only verbose logging plus the bare dump path, so when JSON
// is absent we fall back to the last path-looking token in the output.
// Returns ok=false when no path can be recovered.
func parseMadockExportFile(out string) (string, bool) {
	nOpen := strings.Index(out, "{")
	nClose := strings.LastIndex(out, "}")
	if nOpen != -1 && nClose > nOpen {
		export := madockExportOutput{}
		if err := json.Unmarshal([]byte(out[nOpen:nClose+1]), &export); err == nil {
			if file := strings.TrimSpace(export.Data.File); file != "" {
				return file, true
			}
		}
	}

	// Fallback: older remote madock without --json support — recover the bare
	// dump path (the last matching token, since logging precedes it).
	if matches := dumpPathRe.FindAllString(out, -1); len(matches) > 0 {
		return strings.TrimSpace(matches[len(matches)-1]), true
	}
	return "", false
}

// tryRemoteMadockExport produces the dump natively when madock is installed on the
// remote host. It runs `madock db:export --json` from the project root, which dumps
// the database inside the remote container (no PHP or mysqldump needed on the host),
// then downloads the resulting archive locally and removes it from the remote host.
// Returns false on any failure so the caller can fall back to the PHP/mysqldump flow.
func tryRemoteMadockExport(conn *ssh.Client, remoteDir, name string, args *arg_struct.ControllerGeneralRemoteSyncDb) bool {
	// Probe and run through a login shell (bash -lc) so the user's profile PATH is
	// loaded: a plain ssh exec session is non-login/non-interactive, so madock
	// installed under ~/bin or /usr/local/bin (added to PATH in ~/.profile or
	// ~/.bash_profile) is not on PATH and the probe would falsely report it missing.
	if out, err := remote_sync.RunCommandSafe(conn, "bash -lc "+shellSingleQuote("command -v madock")); err != nil || strings.TrimSpace(out) == "" {
		return false
	}

	exportCmd := "cd '" + remoteDir + "' && madock db:export --json"
	if name = strings.TrimSpace(name); name != "" {
		exportCmd += " -n '" + name + "'"
	}
	for _, t := range args.IgnoreTable {
		exportCmd += " --ignore-table '" + t + "'"
	}

	out, err := remote_sync.RunCommandSafe(conn, "bash -lc "+shellSingleQuote(exportCmd))
	if err != nil {
		fmt.Println(out)
		return false
	}

	remoteFile, ok := parseMadockExportFile(out)
	if !ok {
		return false
	}

	sc, err := sftp.NewClient(conn)
	if err != nil {
		logger.Fatal(err)
	}
	defer func(sc *sftp.Client) {
		if cerr := sc.Close(); cerr != nil {
			logger.Fatal(cerr)
		}
	}(sc)

	execPath := paths.GetExecDirPath()
	projectName := configs.GetProjectName()
	localName := "remote_" + strings.TrimPrefix(filepath.Base(remoteFile), "local_")
	localPath := execPath + "/projects/" + projectName + "/backup/db/" + localName
	if err = remote_sync.DownloadFile(sc, remoteFile, localPath, false, false); err != nil {
		logger.Fatal(err)
	}

	remote_sync.RunCommand(conn, "rm '"+remoteFile+"'")
	// Guard against an older remote madock whose db:export does not yet fail on a
	// broken dump: a successful-looking run can still hand us an empty archive.
	assertDumpNotEmpty(localPath, "remote madock db:export")
	fmt.Println("")
	fmtc.SuccessLn("A database dump was created and saved locally. To import a database dump locally run the command `madock db:import`")
	return true
}

// getRemoteDbCredsJSON reads the remote database credentials from the application's
// own config file and returns them as a RemoteDbStruct-compatible JSON string.
// The remote config is fetched with `cat` and parsed locally in Go, so no language
// runtime (php, node, python, ...) is required on the remote host. The config file
// location depends on the platform:
//
//   - magento2: app/etc/env.php (db.connection.default)
//   - shopware/sylius/medusa/saleor/spree: DATABASE_URL in .env(.local)
//   - woocommerce: wp-config.php DB_* constants
//   - prestashop: app/config/parameters.php database_* parameters
//
// Returns "" for SaaS/static frontends (shopify, bigcommerce) and custom projects
// that have no standard DB config — those require `--db-*` flags or remote madock.
func getRemoteDbCredsJSON(conn *ssh.Client, remoteDir, platform string) string {
	switch platform {
	case "magento2":
		return credsFromMagentoEnv(conn, remoteDir)
	case "shopware", "sylius", "medusa", "saleor", "spree":
		return credsFromDatabaseURL(conn, remoteDir)
	case "woocommerce":
		return credsFromWpConfig(conn, remoteDir)
	case "prestashop":
		return credsFromPrestashop(conn, remoteDir)
	}
	return ""
}

// shellSingleQuote wraps s in single quotes for safe use as a single shell
// argument, escaping any embedded single quotes.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// assertDumpNotEmpty aborts with a helpful message when the downloaded dump is
// suspiciously small (an empty gzip stream is ~20 bytes), which means the remote
// dump produced no data. source describes where the dump came from so the message
// can point at the likely cause.
func assertDumpNotEmpty(localPath, source string) {
	info, err := os.Stat(localPath)
	if err != nil {
		logger.Fatal(err)
	}
	if info.Size() < 100 {
		_ = os.Remove(localPath)
		logger.Fatal(fmt.Sprintf("The downloaded dump is empty (%d bytes). The remote dump (%s) produced no data.\nLikely causes:\n  - the DB client (mysqldump/pg_dump/mongodump) is missing on the remote host\n  - the database host from the remote config is not reachable from the remote shell (e.g. a docker-internal name)\n  - an outdated remote madock that does not detect a failed dump\nFix: install the DB client or madock on the server, or pass --db-host/--db-port/--db-user/--db-password/--db-name explicitly.", info.Size(), source))
	}
}

// reFirstValue returns the first capture-group match of pattern in content, or "".
func reFirstValue(content, pattern string) string {
	if m := regexp.MustCompile(pattern).FindStringSubmatch(content); len(m) == 2 {
		return m[1]
	}
	return ""
}

// credsToJSON builds the RemoteDbStruct-compatible JSON consumed by the caller.
func credsToJSON(host, port, dbname, user, password string) string {
	return fmt.Sprintf("{\"host\":\"%s\",\"dbname\":\"%s\",\"username\":\"%s\",\"password\":\"%s\",\"port\":\"%s\"}",
		host, dbname, user, password, port)
}

// splitHostPort splits an optional "host:port" suffix. The suffix is only treated
// as a port when it is all digits, so a unix-socket path (e.g.
// "localhost:/var/run/mysqld/mysqld.sock") stays in the host part.
func splitHostPort(hostPort string) (string, string) {
	if i := strings.LastIndex(hostPort, ":"); i != -1 {
		if port := hostPort[i+1:]; port != "" && strings.IndexFunc(port, func(r rune) bool { return r < '0' || r > '9' }) == -1 {
			return hostPort[:i], port
		}
	}
	return hostPort, ""
}

// credsFromMagentoEnv reads db.connection.default from app/etc/env.php.
func credsFromMagentoEnv(conn *ssh.Client, remoteDir string) string {
	out, err := remote_sync.RunCommandSafe(conn, "cat '"+remoteDir+"/app/etc/env.php' 2>/dev/null")
	if err != nil {
		return ""
	}
	return parseMagentoEnv(out)
}

// parseMagentoEnv extracts the default DB connection from a Magento 2 env.php.
// env.php is a PHP array with many 'default' keys (cache, session, ...), so the
// scan is narrowed to the db -> connection -> default block before reading the
// positional keys. The default connection is the first one inside 'connection'.
func parseMagentoEnv(content string) string {
	// Walk down 'db' -> 'connection' -> 'default' so the cache/session 'default'
	// blocks and the 'indexer' connection cannot shadow the real credentials.
	for _, anchor := range []string{"'db'", "'connection'", "'default'"} {
		if i := strings.Index(content, anchor); i != -1 {
			content = content[i+len(anchor):]
		}
	}
	keyVal := func(key string) string {
		return reFirstValue(content, `['"]`+key+`['"]\s*=>\s*['"]([^'"]*)['"]`)
	}
	dbname := keyVal("dbname")
	if dbname == "" {
		return ""
	}
	host, port := splitHostPort(keyVal("host"))
	return credsToJSON(host, port, dbname, keyVal("username"), keyVal("password"))
}

// credsFromDatabaseURL reads DATABASE_URL from the remote .env / .env.local.
func credsFromDatabaseURL(conn *ssh.Client, remoteDir string) string {
	out, err := remote_sync.RunCommandSafe(conn, "cat '"+remoteDir+"/.env.local' '"+remoteDir+"/.env' 2>/dev/null")
	if err != nil {
		return ""
	}
	return parseDatabaseURL(out)
}

// parseDatabaseURL extracts DB credentials from a DATABASE_URL entry in env file
// content. Symfony-style overrides apply: .env.local is concatenated first, so the
// first matching line wins.
func parseDatabaseURL(content string) string {
	dbURL := ""
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "export "))
		if !strings.HasPrefix(line, "DATABASE_URL") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if dbURL = strings.Trim(strings.TrimSpace(parts[1]), "\"'"); dbURL != "" {
			break
		}
	}
	if dbURL == "" {
		return ""
	}

	u, perr := url.Parse(dbURL)
	if perr != nil || u.Host == "" {
		return ""
	}
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		return ""
	}
	password, _ := u.User.Password()
	return credsToJSON(u.Hostname(), u.Port(), dbname, u.User.Username(), password)
}

// credsFromWpConfig reads the DB_* constants from the remote wp-config.php.
func credsFromWpConfig(conn *ssh.Client, remoteDir string) string {
	out, err := remote_sync.RunCommandSafe(conn, "cat '"+remoteDir+"/wp-config.php' 2>/dev/null")
	if err != nil {
		return ""
	}
	return parseWpConfig(out)
}

// parseWpConfig extracts DB credentials from WordPress/WooCommerce wp-config.php
// content. DB_HOST may carry a host:port suffix.
func parseWpConfig(content string) string {
	get := func(name string) string {
		return reFirstValue(content, `define\(\s*['"]`+name+`['"]\s*,\s*['"]([^'"]*)['"]`)
	}
	dbname := get("DB_NAME")
	if dbname == "" {
		return ""
	}
	host, port := splitHostPort(get("DB_HOST"))
	return credsToJSON(host, port, dbname, get("DB_USER"), get("DB_PASSWORD"))
}

// credsFromPrestashop reads database_* params from the remote parameters.php.
func credsFromPrestashop(conn *ssh.Client, remoteDir string) string {
	out, err := remote_sync.RunCommandSafe(conn, "cat '"+remoteDir+"/app/config/parameters.php' 2>/dev/null")
	if err != nil {
		return ""
	}
	return parsePrestashop(out)
}

// parsePrestashop extracts DB credentials from PrestaShop parameters.php content.
func parsePrestashop(content string) string {
	get := func(name string) string {
		return reFirstValue(content, `['"]`+name+`['"]\s*=>\s*['"]([^'"]*)['"]`)
	}
	dbname := get("database_name")
	if dbname == "" {
		return ""
	}
	return credsToJSON(get("database_host"), get("database_port"), dbname, get("database_user"), get("database_password"))
}
