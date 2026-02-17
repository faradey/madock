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
	"strings"
	"time"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"remote:sync:db"},
		Handler:  Execute,
		Help:     "Sync remote database",
		Category: "remote",
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

	remoteDir := siteRootPath
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
	} else {
		if projectConf["platform"] == "magento2" {
			result = remote_sync.RunCommand(conn, "php -r \"\\$r1 = include('"+remoteDir+"/app/etc/env.php'); echo json_encode(\\$r1[\\\"db\\\"][\\\"connection\\\"][\\\"default\\\"]);\"")
		} else if projectConf["platform"] == "shopware" {
			result = remote_sync.RunCommand(conn, "php -r \"\\$parsed_url=[];\\$env = include('"+remoteDir+"/.env'); $lines = explode(\"\\n\",\\$env); foreach(\\$lines as \\$line){  preg_match(\"/([^#]+)\\=(.*)/\",\\$line,\\$matches);  if(isset(\\$matches[1]) && \\$matches[1] == \"DATABASE_URL\" && !empty(\\$matches[2])){ \\$parsed_url = parse_url(\\$matches[2]); \\$parsed_url = ['username' => \\$parsed_url['user'],'password' => \\$parsed_url['pass'],'host'     => \\$parsed_url['host'],'port'     => \\$parsed_url['port']??\"3306\",'dbname' => ltrim($parsed_url['path'], '/')];   break;  }} echo json_encode(\\$parsed_url);\"")
		}
	}

	nOpenBrace := strings.Index(result, "{")
	if nOpenBrace != -1 {
		result = result[nOpenBrace:]
	} else {
		fmt.Println(result)
		logger.Fatal("Failed to get database authentication data (row 65)")
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
		dumpName := "remote_" + name + curDateTime + ".sql.gz"

		ignoreTablesStr := ""
		ignoreTables := args.IgnoreTable
		if len(ignoreTables) > 0 {
			ignoreTablesStr = " --ignore-table=" + dbAuthData.Dbname + "." + strings.Join(ignoreTables, " --ignore-table="+dbAuthData.Dbname+".")
		}
		dbPort := ""
		if dbAuthData.Port != "" {
			dbPort = " -P " + dbAuthData.Port
		}
		result = remote_sync.RunCommand(conn, "mysqldump -u \""+dbAuthData.Username+"\" -p\""+dbAuthData.Password+"\" -h "+dbAuthData.Host+dbPort+" --quick --lock-tables=false --no-tablespaces --triggers"+ignoreTablesStr+" "+dbAuthData.Dbname+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/' | gzip > "+"/tmp/"+dumpName)
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
		err = remote_sync.DownloadFile(sc, "/tmp/"+dumpName, execPath+"/projects/"+projectName+"/backup/db/"+dumpName, false, false)
		if err != nil {
			logger.Fatal(err)
		}
		result = remote_sync.RunCommand(conn, "rm "+"/tmp/"+dumpName)
		fmt.Println("")
		fmtc.SuccessLn("A database dump was created and saved locally. To import a database dump locally run the command `madock db:import`")
	} else {
		fmt.Println("Failed to get database authentication data (row 110)")
	}
}
