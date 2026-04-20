package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"mcp"},
		Handler:  Execute,
		Help:     "Start MCP (Model Context Protocol) server for AI assistants",
		Category: "general",
	})
}

func Execute() {
	s := server.NewMCPServer(
		"madock",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	registerTools(s)
	registerResources(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

func registerTools(s *server.MCPServer) {
	// --- Read-only / informational tools ---

	s.AddTool(mcp.NewTool("madock_status",
		mcp.WithDescription("Get the status of Docker containers for the current madock project. Returns service names, states (running/stopped), and tool statuses (cron, debugger)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleStatus)

	s.AddTool(mcp.NewTool("madock_config_list",
		mcp.WithDescription("List all configuration options for the current madock project. Returns key-value pairs including platform, PHP version, database settings, services, etc."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleConfigList)

	s.AddTool(mcp.NewTool("madock_db_info",
		mcp.WithDescription("Show database connection information (type, host, port, database name, user, password) for the current madock project."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleDbInfo)

	s.AddTool(mcp.NewTool("madock_service_list",
		mcp.WithDescription("List all available services and their enabled/disabled status for the current madock project. Services include: redis, elasticsearch, opensearch, rabbitmq, varnish, nodejs, etc."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleServiceList)

	s.AddTool(mcp.NewTool("madock_scope_list",
		mcp.WithDescription("List all configuration scopes (multi-site/multi-store setups) and show which scope is currently active."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleScopeList)

	s.AddTool(mcp.NewTool("madock_logs",
		mcp.WithDescription("View Docker container logs for a specific service. Useful for debugging issues with PHP, nginx, database, etc."),
		mcp.WithString("service",
			mcp.Description("Service name to get logs for (e.g. php, nginx, db, redis, elasticsearch). If empty, shows logs for all services."),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleLogs)

	s.AddTool(mcp.NewTool("madock_info_ports",
		mcp.WithDescription("Show exposed port mappings for all services in the current madock project."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleInfoPorts)

	s.AddTool(mcp.NewTool("madock_help",
		mcp.WithDescription("Get help information about madock commands. Without a command name, returns a list of all available commands. With a command name, returns detailed help for that specific command."),
		mcp.WithString("command",
			mcp.Description("Command name to get detailed help for (e.g. setup, start, db:import). Leave empty for general help."),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), handleHelp)

	// --- Configuration tools ---

	s.AddTool(mcp.NewTool("madock_config_set",
		mcp.WithDescription("Set a configuration option for the current madock project. Use config_list to see available keys. Example keys: php/version, db/version, search/engine, redis/enabled, nginx/hosts/{code}/name."),
		mcp.WithString("key",
			mcp.Required(),
			mcp.Description("Configuration key (e.g. php/version, db/version, search/engine)"),
		),
		mcp.WithString("value",
			mcp.Required(),
			mcp.Description("Configuration value to set"),
		),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	), handleConfigSet)

	// --- Container lifecycle tools ---

	s.AddTool(mcp.NewTool("madock_start",
		mcp.WithDescription("Start Docker containers for the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleStart)

	s.AddTool(mcp.NewTool("madock_stop",
		mcp.WithDescription("Stop Docker containers for the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleStop)

	s.AddTool(mcp.NewTool("madock_restart",
		mcp.WithDescription("Restart Docker containers for the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleRestart)

	s.AddTool(mcp.NewTool("madock_rebuild",
		mcp.WithDescription("Rebuild Docker containers for the current madock project. Regenerates docker-compose.yml and Dockerfiles from templates. Use after config changes (PHP version, services, etc.)."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleRebuild)

	// --- Service management tools ---

	s.AddTool(mcp.NewTool("madock_service_enable",
		mcp.WithDescription("Enable a service for the current madock project. After enabling, run rebuild to apply changes."),
		mcp.WithString("service",
			mcp.Required(),
			mcp.Description("Service name to enable (e.g. redis, elasticsearch, opensearch, rabbitmq, varnish, nodejs, mailpit)"),
		),
		mcp.WithDestructiveHintAnnotation(false),
	), handleServiceEnable)

	s.AddTool(mcp.NewTool("madock_service_disable",
		mcp.WithDescription("Disable a service for the current madock project. After disabling, run rebuild to apply changes."),
		mcp.WithString("service",
			mcp.Required(),
			mcp.Description("Service name to disable (e.g. redis, elasticsearch, opensearch, rabbitmq, varnish, nodejs, mailpit)"),
		),
		mcp.WithDestructiveHintAnnotation(false),
	), handleServiceDisable)

	// --- Cron & Debug tools ---

	s.AddTool(mcp.NewTool("madock_cron_enable",
		mcp.WithDescription("Enable cron jobs for the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleCronEnable)

	s.AddTool(mcp.NewTool("madock_cron_disable",
		mcp.WithDescription("Disable cron jobs for the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleCronDisable)

	s.AddTool(mcp.NewTool("madock_debug_enable",
		mcp.WithDescription("Enable Xdebug for PHP debugging in the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleDebugEnable)

	s.AddTool(mcp.NewTool("madock_debug_disable",
		mcp.WithDescription("Disable Xdebug for the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleDebugDisable)

	// --- Database tools ---

	s.AddTool(mcp.NewTool("madock_db_import",
		mcp.WithDescription("Import a database dump file into the project database. Supports .sql, .sql.gz, .zip files."),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Path to the database dump file (e.g. dump.sql, dump.sql.gz)"),
		),
		mcp.WithDestructiveHintAnnotation(true),
	), handleDbImport)

	s.AddTool(mcp.NewTool("madock_db_export",
		mcp.WithDescription("Export the project database to a dump file."),
		mcp.WithString("name",
			mcp.Description("Name for the export file (without extension). If empty, uses default naming."),
		),
		mcp.WithDestructiveHintAnnotation(false),
	), handleDbExport)

	s.AddTool(mcp.NewTool("madock_db_execute",
		mcp.WithDescription("Execute a SQL query against the project database."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("SQL query to execute"),
		),
	), handleDbExecute)

	// --- Scope tools ---

	s.AddTool(mcp.NewTool("madock_scope_add",
		mcp.WithDescription("Add a new configuration scope for multi-site/multi-store setup."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Scope name (e.g. b2b, wholesale, fr)"),
		),
		mcp.WithDestructiveHintAnnotation(false),
	), handleScopeAdd)

	s.AddTool(mcp.NewTool("madock_scope_set",
		mcp.WithDescription("Set the active configuration scope."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Scope name to activate"),
		),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	), handleScopeSet)

	// --- Development tools ---

	s.AddTool(mcp.NewTool("madock_composer",
		mcp.WithDescription("Run a Composer command inside the PHP container (e.g. install, update, require)."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Composer arguments (e.g. 'install', 'require vendor/package', 'update --no-dev')"),
		),
	), handleComposer)

	s.AddTool(mcp.NewTool("madock_magento",
		mcp.WithDescription("Run a Magento CLI command inside the PHP container. Only available for Magento 2 projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Magento CLI arguments (e.g. 'setup:upgrade', 'cache:flush', 'indexer:reindex')"),
		),
	), handleMagento)

	s.AddTool(mcp.NewTool("madock_n98",
		mcp.WithDescription("Run n98-magerun command inside the PHP container. Only available for Magento 2 projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("n98-magerun arguments (e.g. 'db:info', 'sys:info', 'dev:console')"),
		),
	), handleN98)

	s.AddTool(mcp.NewTool("madock_cloud",
		mcp.WithDescription("Run Magento Cloud CLI command inside the PHP container. Only available for Magento 2 projects. Use $project as placeholder for configured project name."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Magento Cloud CLI arguments (e.g. 'environment:list', 'ssh', 'db:dump')"),
		),
	), handleCloud)

	s.AddTool(mcp.NewTool("madock_wp",
		mcp.WithDescription("Run a WP-CLI command inside the PHP container. Only available for WooCommerce/WordPress projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("WP-CLI arguments (e.g. 'plugin list', 'cache flush', 'db check', 'option get siteurl')"),
		),
	), handleWp)

	s.AddTool(mcp.NewTool("madock_shopware",
		mcp.WithDescription("Run a Shopware CLI command (bin/console) inside the PHP container. Only available for Shopware projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Shopware CLI arguments (e.g. 'cache:clear', 'plugin:list', 'theme:compile')"),
		),
	), handleShopware)

	s.AddTool(mcp.NewTool("madock_shopware_bin",
		mcp.WithDescription("Run a Shopware bin/* command inside the PHP container. Only available for Shopware projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Shopware bin command and arguments (e.g. 'build-js.sh', 'watch-storefront.sh')"),
		),
	), handleShopwareBin)

	s.AddTool(mcp.NewTool("madock_shopify",
		mcp.WithDescription("Execute a command inside the Shopify project container (in the workdir root). Only available for Shopify projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Command to execute (e.g. 'npm install', 'node app.js', 'ls -la')"),
		),
	), handleShopify)

	s.AddTool(mcp.NewTool("madock_shopify_web",
		mcp.WithDescription("Execute a command inside the Shopify project container in the web/ directory. Only available for Shopify projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Command to execute in web/ directory (e.g. 'npm install', 'npm run dev')"),
		),
	), handleShopifyWeb)

	s.AddTool(mcp.NewTool("madock_shopify_web_frontend",
		mcp.WithDescription("Execute a command inside the Shopify project container in the web/frontend/ directory. Only available for Shopify projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Command to execute in web/frontend/ directory (e.g. 'npm install', 'npm run build')"),
		),
	), handleShopifyWebFrontend)

	s.AddTool(mcp.NewTool("madock_prestashop",
		mcp.WithDescription("Run a PrestaShop CLI command (bin/console) inside the PHP container. Only available for PrestaShop projects."),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("PrestaShop CLI arguments (e.g. 'cache:clear', 'prestashop:module list')"),
		),
	), handlePrestaShop)

	s.AddTool(mcp.NewTool("madock_flush_cache",
		mcp.WithDescription("Flush all caches for the current project (platform-specific: Magento cache, OPcache, etc.)."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleFlushCache)

	s.AddTool(mcp.NewTool("madock_ssl_rebuild",
		mcp.WithDescription("Rebuild SSL certificates for the current madock project."),
		mcp.WithDestructiveHintAnnotation(false),
	), handleSslRebuild)

	// --- Remote sync tools ---

	s.AddTool(mcp.NewTool("madock_remote_sync_db",
		mcp.WithDescription("Synchronize database from a remote server. Requires SSH configuration in project config."),
		mcp.WithString("ssh_type",
			mcp.Description("SSH connection type: dev, stage, or prod"),
		),
		mcp.WithDestructiveHintAnnotation(true),
	), handleRemoteSyncDb)

	s.AddTool(mcp.NewTool("madock_remote_sync_media",
		mcp.WithDescription("Synchronize media files from a remote server. Requires SSH configuration in project config."),
		mcp.WithString("ssh_type",
			mcp.Description("SSH connection type: dev, stage, or prod"),
		),
		mcp.WithBoolean("images_only",
			mcp.Description("Sync only image files"),
		),
		mcp.WithDestructiveHintAnnotation(false),
	), handleRemoteSyncMedia)
}

// ---------------------------------------------------------------------------
// Resources
// ---------------------------------------------------------------------------

func registerResources(s *server.MCPServer) {
	s.AddResource(
		mcp.NewResource(
			"madock://docs/llms.txt",
			"madock documentation",
			mcp.WithResourceDescription("Complete madock documentation for AI assistants: all commands, configuration options, supported platforms, and architecture overview."),
			mcp.WithMIMEType("text/plain"),
		),
		handleLlmsTxt,
	)
}

func handleLlmsTxt(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Try reading llms.txt next to the madock executable
	execDir := paths.GetExecDirPath()
	data, err := os.ReadFile(filepath.Join(execDir, "llms.txt"))
	if err != nil {
		// Fallback: try current working directory
		data, err = os.ReadFile("llms.txt")
		if err != nil {
			return nil, fmt.Errorf("llms.txt not found next to madock binary or in current directory")
		}
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/plain",
			Text:     string(data),
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const commandTimeout = 10 * time.Minute

// runResult holds separated stdout and stderr from a subprocess.
type runResult struct {
	Stdout string
	Stderr string
}

func runMadock(ctx context.Context, args ...string) (runResult, error) {
	executable, err := os.Executable()
	if err != nil {
		return runResult{}, fmt.Errorf("failed to find madock executable: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir, _ = os.Getwd()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	return runResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}, err
}

func runMadockQuiet(ctx context.Context, args ...string) (runResult, error) {
	return runMadock(ctx, append(args, "--quiet")...)
}

func toolResult(r runResult, err error, action string) (*mcp.CallToolResult, error) {
	if err != nil {
		// Include both stdout and stderr for diagnostics
		msg := fmt.Sprintf("Failed to %s: %s", action, err)
		if r.Stderr != "" {
			msg += "\n" + r.Stderr
		}
		if r.Stdout != "" {
			msg += "\n" + r.Stdout
		}
		return mcp.NewToolResultError(msg), nil
	}
	out := r.Stdout
	if out == "" {
		out = "Done"
	}
	return mcp.NewToolResultText(out), nil
}

// ---------------------------------------------------------------------------
// Read-only handlers
// ---------------------------------------------------------------------------

func handleStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "status", "--json")
	return toolResult(r, err, "get status")
}

func handleConfigList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "config:list", "--json")
	return toolResult(r, err, "list config")
}

func handleDbInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "db:info", "--json")
	return toolResult(r, err, "get db info")
}

func handleServiceList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "service:list", "--json")
	return toolResult(r, err, "list services")
}

func handleScopeList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "scope:list", "--json")
	return toolResult(r, err, "list scopes")
}

func handleLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc := request.GetString("service", "")
	args := []string{"logs"}
	if svc != "" {
		args = append(args, "-s", svc)
	}
	r, err := runMadock(ctx, args...)
	return toolResult(r, err, "get logs")
}

func handleInfoPorts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "info:ports", "--json")
	return toolResult(r, err, "get ports")
}

func handleHelp(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cmdName := request.GetString("command", "")
	args := []string{"help"}
	if cmdName != "" {
		args = append(args, cmdName)
	}
	r, err := runMadock(ctx, args...)
	return toolResult(r, err, "get help")
}

// ---------------------------------------------------------------------------
// Configuration handlers
// ---------------------------------------------------------------------------

func handleConfigSet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: key"), nil
	}
	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: value"), nil
	}

	r, runErr := runMadock(ctx, "config:set", "-n", key, "-v", value)
	if runErr != nil {
		msg := fmt.Sprintf("Failed to set config: %s", runErr)
		if r.Stderr != "" {
			msg += "\n" + r.Stderr
		}
		return mcp.NewToolResultError(msg), nil
	}

	result := fmt.Sprintf("Set %s = %s", key, value)
	if r.Stdout != "" {
		result += "\n" + r.Stdout
	}
	return mcp.NewToolResultText(result), nil
}

// ---------------------------------------------------------------------------
// Container lifecycle handlers
// ---------------------------------------------------------------------------

func handleStart(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadockQuiet(ctx, "start")
	return toolResult(r, err, "start containers")
}

func handleStop(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadockQuiet(ctx, "stop")
	return toolResult(r, err, "stop containers")
}

func handleRestart(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadockQuiet(ctx, "restart")
	return toolResult(r, err, "restart containers")
}

func handleRebuild(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadockQuiet(ctx, "rebuild")
	return toolResult(r, err, "rebuild containers")
}

// ---------------------------------------------------------------------------
// Service management handlers
// ---------------------------------------------------------------------------

func handleServiceEnable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc, err := request.RequireString("service")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: service"), nil
	}
	r, runErr := runMadock(ctx, "service:enable", svc)
	return toolResult(r, runErr, "enable service")
}

func handleServiceDisable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	svc, err := request.RequireString("service")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: service"), nil
	}
	r, runErr := runMadock(ctx, "service:disable", svc)
	return toolResult(r, runErr, "disable service")
}

// ---------------------------------------------------------------------------
// Cron & Debug handlers
// ---------------------------------------------------------------------------

func handleCronEnable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "cron:enable")
	return toolResult(r, err, "enable cron")
}

func handleCronDisable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "cron:disable")
	return toolResult(r, err, "disable cron")
}

func handleDebugEnable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "debug:enable")
	return toolResult(r, err, "enable debug")
}

func handleDebugDisable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "debug:disable")
	return toolResult(r, err, "disable debug")
}

// ---------------------------------------------------------------------------
// Database handlers
// ---------------------------------------------------------------------------

func handleDbImport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file, err := request.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: file"), nil
	}
	r, runErr := runMadock(ctx, "db:import", file)
	return toolResult(r, runErr, "import database")
}

func handleDbExport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.GetString("name", "")
	args := []string{"db:export"}
	if name != "" {
		args = append(args, "-n", name)
	}
	r, err := runMadock(ctx, args...)
	return toolResult(r, err, "export database")
}

func handleDbExecute(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: query"), nil
	}
	r, runErr := runMadock(ctx, "db:execute", query)
	return toolResult(r, runErr, "execute SQL")
}

// ---------------------------------------------------------------------------
// Scope handlers
// ---------------------------------------------------------------------------

func handleScopeAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: name"), nil
	}
	r, runErr := runMadock(ctx, "scope:add", name)
	return toolResult(r, runErr, "add scope")
}

func handleScopeSet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: name"), nil
	}
	r, runErr := runMadock(ctx, "scope:set", name)
	return toolResult(r, runErr, "set scope")
}

// ---------------------------------------------------------------------------
// Development tool handlers
// ---------------------------------------------------------------------------

func handleComposer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"composer"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run composer")
}

func handleMagento(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"magento"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run magento CLI")
}

func handleN98(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"n98"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run n98-magerun")
}

func handleCloud(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"cloud"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run magento cloud CLI")
}

func handleWp(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"wp"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run WP-CLI")
}

func handleShopware(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"shopware"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run shopware CLI")
}

func handleShopwareBin(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"shopware:bin"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run shopware bin command")
}

func handleShopify(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"shopify"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run shopify command")
}

func handleShopifyWeb(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"shopify:web"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run shopify web command")
}

func handleShopifyWebFrontend(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"shopify:web:frontend"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run shopify web frontend command")
}

func handlePrestaShop(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawArgs, err := request.RequireString("args")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: args"), nil
	}
	args := append([]string{"prestashop"}, strings.Fields(rawArgs)...)
	r, runErr := runMadock(ctx, args...)
	return toolResult(r, runErr, "run prestashop CLI")
}

func handleFlushCache(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadock(ctx, "c:f")
	return toolResult(r, err, "flush cache")
}

func handleSslRebuild(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, err := runMadockQuiet(ctx, "ssl:rebuild")
	return toolResult(r, err, "rebuild SSL")
}

// ---------------------------------------------------------------------------
// Remote sync handlers
// ---------------------------------------------------------------------------

func handleRemoteSyncDb(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := []string{"remote:sync:db"}
	if sshType := request.GetString("ssh_type", ""); sshType != "" {
		args = append(args, "-s", sshType)
	}
	r, err := runMadock(ctx, args...)
	return toolResult(r, err, "sync remote database")
}

func handleRemoteSyncMedia(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := []string{"remote:sync:media"}
	if sshType := request.GetString("ssh_type", ""); sshType != "" {
		args = append(args, "-s", sshType)
	}
	if request.GetArguments()["images_only"] == true {
		args = append(args, "-i")
	}
	r, err := runMadock(ctx, args...)
	return toolResult(r, err, "sync remote media")
}
