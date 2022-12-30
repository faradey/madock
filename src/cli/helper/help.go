package helper

import (
	"fmt"

	"github.com/faradey/madock/src/cli/fmtc"
)

func Help() {
	/* 16 commands */
	fmtc.WarningLn("Usage:")
	tab()
	fmt.Println("command [arguments]")
	fmtc.Warning("Available commands:")
	describeByLevel("bash", "Connect into container using bash", 0)
	describeByLevel("[name of container]", "Name of container. Optional. Default container: php. For example: php, node, db, nginx", 1)
	describeByLevel("c:f", "Cleaning up static and generated files", 0)
	describeByLevel("composer", "Execute composer inside php container", 0)
	describeByLevel("compress", "Compress a project to archive", 0)
	describeByLevel("config", "Viewing and changing the project configuration", 0)
	describeByLevel("show", "List all project environment settings", 1)
	describeByLevel("set", "Set parameters", 1)
	describeByLevel("--hosts", "Domains and code of project websites. Separated by commas. For example: one.example.com:base two.example.com:two_code. Optional", 2)
	describeByLevel("cron:enable", "Enable cron", 0)
	describeByLevel("cron:disable", "Disable cron", 0)
	describeByLevel("db:import", "Import database", 0)
	describeByLevel("-f", "Force mode", 1)
	describeByLevel("db:export", "Export database", 0)
	describeByLevel("db:info", "Information about credentials and remote host and port", 0)
	describeByLevel("debug:enable", "Enable xdebug", 0)
	describeByLevel("debug:disable", "Disable xdebug", 0)
	describeByLevel("info", "Show information about third-parties modules (name, current version, latest version, status)", 0)
	describeByLevel("help", "Display help for commands", 0)
	describeByLevel("logs", "View logs of a container", 0)
	describeByLevel("[name of container]", "Container name. Optional. Default container: php. Example: php", 1)
	describeByLevel("magento", "Execute Magento command inside php container", 0)
	describeByLevel("node", "Execute NodeJs command inside php container", 0)
	describeByLevel("proxy", "Actions on the proxy server", 0)
	describeByLevel("start", "Start a proxy server", 1)
	describeByLevel("stop", "Stop a proxy server", 1)
	describeByLevel("restart", "Restart a proxy server", 1)
	describeByLevel("rebuild", "Rebuild a proxy server", 1)
	describeByLevel("prune", "Prune a proxy server", 1)
	describeByLevel("prune", "Stop and delete running project containers", 0)
	describeByLevel("rebuild", "Recreation of all containers in the project. All containers are re-created and the images from the Dockerfile are rebuilt", 0)
	describeByLevel("remote:sync:media", "Synchronization media files from remote host", 0)
	describeByLevel("--images-only", "Synchronization images only", 1)
	describeByLevel("--compress", "Apply lossy compression. Images will have weight equals 30% of original", 1)
	describeByLevel("remote:sync:db", "Create and download dump of DB from remote host", 0)
	describeByLevel("remote:sync:file", "Download a file from remote host", 0)
	describeByLevel("--path", "Path to file on server (from Magento root)", 1)
	describeByLevel("restart", "Restarting all containers and services. Stop all containers and start them again", 0)
	describeByLevel("service:list", "Show all services", 0)
	describeByLevel("service:enable", "Enable the service", 0)
	describeByLevel("[service name]", "Service name", 1)
	describeByLevel("service:disable", "Disable the service", 0)
	describeByLevel("[service name]", "Service name", 1)
	describeByLevel("setup", "Initial project setup", 0)
	describeByLevel("--download", "Download the specific Magento version from Composer to the container", 1)
	describeByLevel("--install", "Install Magento from the source code", 1)
	describeByLevel("ssl:rebuild", "Rebuild SSL Certificates", 0)
	describeByLevel("start", "Starting all containers and services", 0)
	describeByLevel("status", "Display the status of the project", 0)
	describeByLevel("stop", "Stopping all containers and services", 0)
	describeByLevel("uncompress", "Uncompress a project from archive", 0)

	fmt.Println("")
}

func describeByLevel(name, desc string, level int) {
	switch level {
	case 0:
		tabln()
		tab()
		fmtc.Success(name)
	case 1:
		tab()
		fmtc.Warning(name)
	case 2:
		tab()
		tab()
		fmtc.Title(name)
	case 3:
		tab()
		tab()
		tab()
		fmtc.Purple(name)
	}
	tab()
	fmt.Println(desc)
	tab()
	tab()
}

func tab() {
	fmt.Print("	")
}

func tabln() {
	fmt.Println("	")
}
