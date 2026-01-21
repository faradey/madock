package command

import (
	"github.com/faradey/madock/src/controller/general/bash"
	"github.com/faradey/madock/src/controller/general/claude"
	"github.com/faradey/madock/src/controller/general/clean_cache"
	"github.com/faradey/madock/src/controller/general/cli"
	"github.com/faradey/madock/src/controller/general/composer"
	"github.com/faradey/madock/src/controller/general/config"
	"github.com/faradey/madock/src/controller/general/cron"
	"github.com/faradey/madock/src/controller/general/db/export"
	_import "github.com/faradey/madock/src/controller/general/db/import"
	info2 "github.com/faradey/madock/src/controller/general/db/info"
	"github.com/faradey/madock/src/controller/general/debug"
	diffController "github.com/faradey/madock/src/controller/general/diff"
	"github.com/faradey/madock/src/controller/general/help"
	"github.com/faradey/madock/src/controller/general/info"
	infoPorts "github.com/faradey/madock/src/controller/general/info/ports"
	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/controller/general/logs"
	"github.com/faradey/madock/src/controller/general/node"
	"github.com/faradey/madock/src/controller/general/open"
	"github.com/faradey/madock/src/controller/general/patch"
	"github.com/faradey/madock/src/controller/general/project/clone"
	"github.com/faradey/madock/src/controller/general/project/remove"
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/controller/general/prune"
	"github.com/faradey/madock/src/controller/general/rebuild"
	db2 "github.com/faradey/madock/src/controller/general/remote_sync/db"
	"github.com/faradey/madock/src/controller/general/remote_sync/file"
	"github.com/faradey/madock/src/controller/general/remote_sync/media"
	"github.com/faradey/madock/src/controller/general/restart"
	"github.com/faradey/madock/src/controller/general/scope/add"
	listScope "github.com/faradey/madock/src/controller/general/scope/list"
	"github.com/faradey/madock/src/controller/general/scope/set"
	"github.com/faradey/madock/src/controller/general/service/disable"
	"github.com/faradey/madock/src/controller/general/service/enable"
	"github.com/faradey/madock/src/controller/general/service/list"
	"github.com/faradey/madock/src/controller/general/setup"
	"github.com/faradey/madock/src/controller/general/setup/env"
	"github.com/faradey/madock/src/controller/general/snapshot/create"
	"github.com/faradey/madock/src/controller/general/snapshot/restore"
	"github.com/faradey/madock/src/controller/general/ssl"
	"github.com/faradey/madock/src/controller/general/start"
	"github.com/faradey/madock/src/controller/general/status"
	"github.com/faradey/madock/src/controller/general/stop"
	"github.com/faradey/madock/src/controller/magento"
	"github.com/faradey/madock/src/controller/magento/cloud"
	"github.com/faradey/madock/src/controller/magento/mftf"
	"github.com/faradey/madock/src/controller/magento/n98"
	"github.com/faradey/madock/src/controller/prestashop"
	"github.com/faradey/madock/src/controller/pwa"
	"github.com/faradey/madock/src/controller/shopify"
	"github.com/faradey/madock/src/controller/shopify/frontend"
	"github.com/faradey/madock/src/controller/shopify/web"
	"github.com/faradey/madock/src/controller/shopware"
	"github.com/faradey/madock/src/helper/compress"
)

func init() {
	// General commands
	Register(&Definition{
		Aliases: []string{"bash"},
		Handler: bash.Execute,
		Help:    "Execute bash in container",
	})
	Register(&Definition{
		Aliases: []string{"c:f"},
		Handler: clean_cache.Execute,
		Help:    "Flush cache",
	})
	Register(&Definition{
		Aliases: []string{"magento-cloud", "cloud"},
		Handler: cloud.Execute,
		Help:    "Execute Magento Cloud CLI",
	})
	Register(&Definition{
		Aliases: []string{"claude"},
		Handler: claude.Execute,
		Help:    "Execute Claude AI assistant",
	})
	Register(&Definition{
		Aliases: []string{"cli"},
		Handler: cli.Execute,
		Help:    "Execute CLI in container",
	})
	Register(&Definition{
		Aliases: []string{"composer"},
		Handler: composer.Execute,
		Help:    "Execute composer command",
	})
	Register(&Definition{
		Aliases: []string{"compress"},
		Handler: compress.Zip,
		Help:    "Compress project files",
	})

	// Config commands
	Register(&Definition{
		Aliases: []string{"config:cache:clean", "c:c:c"},
		Handler: config.CacheClean,
		Help:    "Clean config cache",
	})
	Register(&Definition{
		Aliases: []string{"config:list"},
		Handler: config.ShowEnv,
		Help:    "List configuration. Supports --json (-j) output",
	})
	Register(&Definition{
		Aliases: []string{"config:set"},
		Handler: config.SetEnvOption,
		Help:    "Set configuration option",
	})

	// Cron commands
	Register(&Definition{
		Aliases: []string{"cron:enable"},
		Handler: cron.Enable,
		Help:    "Enable cron",
	})
	Register(&Definition{
		Aliases: []string{"cron:disable"},
		Handler: cron.Disable,
		Help:    "Disable cron",
	})

	// Database commands
	Register(&Definition{
		Aliases: []string{"db:import"},
		Handler: _import.Import,
		Help:    "Import database",
	})
	Register(&Definition{
		Aliases: []string{"db:export"},
		Handler: export.Export,
		Help:    "Export database",
	})
	Register(&Definition{
		Aliases: []string{"db:info"},
		Handler: info2.Info,
		Help:    "Show database info. Supports --json (-j) output",
	})

	// Debug commands
	Register(&Definition{
		Aliases: []string{"debug:enable"},
		Handler: debug.Enable,
		Help:    "Enable debug mode",
	})
	Register(&Definition{
		Aliases: []string{"debug:disable"},
		Handler: debug.Disable,
		Help:    "Disable debug mode",
	})
	Register(&Definition{
		Aliases: []string{"debug:profile:enable"},
		Handler: debug.ProfileEnable,
		Help:    "Enable profiler",
	})
	Register(&Definition{
		Aliases: []string{"debug:profile:disable"},
		Handler: debug.ProfileDisable,
		Help:    "Disable profiler",
	})

	// Info and help
	Register(&Definition{
		Aliases: []string{"info"},
		Handler: info.Info,
		Help:    "Show project info",
	})
	Register(&Definition{
		Aliases: []string{"info:ports"},
		Handler: infoPorts.Execute,
		Help:    "Show project ports. Supports --json (-j) output",
	})
	Register(&Definition{
		Aliases: []string{"install"},
		Handler: install.Execute,
		Help:    "Install Magento",
	})
	Register(&Definition{
		Aliases: []string{"help"},
		Handler: help.Execute,
		Help:    "Show help",
	})
	Register(&Definition{
		Aliases: []string{"logs"},
		Handler: logs.Execute,
		Help:    "Show container logs",
	})

	// Magento commands
	Register(&Definition{
		Aliases: []string{"magento", "m"},
		Handler: magento.Execute,
		Help:    "Execute Magento CLI",
	})
	Register(&Definition{
		Aliases: []string{"mftf"},
		Handler: mftf.Execute,
		Help:    "Execute MFTF",
	})
	Register(&Definition{
		Aliases: []string{"mftf:init"},
		Handler: mftf.Init,
		Help:    "Initialize MFTF",
	})
	Register(&Definition{
		Aliases: []string{"n98"},
		Handler: n98.Execute,
		Help:    "Execute n98-magerun",
	})

	// Node
	Register(&Definition{
		Aliases: []string{"node"},
		Handler: node.Execute,
		Help:    "Execute node command",
	})

	// Open and patch
	Register(&Definition{
		Aliases: []string{"open"},
		Handler: open.Execute,
		Help:    "Open project in browser",
	})
	Register(&Definition{
		Aliases: []string{"patch:create"},
		Handler: patch.Execute,
		Help:    "Create patch file",
	})
	Register(&Definition{
		Aliases: []string{"diff"},
		Handler: diffController.Execute,
		Help:    "Show diff",
	})

	// PrestaShop
	Register(&Definition{
		Aliases: []string{"prestashop", "ps"},
		Handler: prestashop.Execute,
		Help:    "Execute PrestaShop CLI",
	})

	// Project commands
	Register(&Definition{
		Aliases: []string{"project:clone"},
		Handler: clone.Execute,
		Help:    "Clone project",
	})
	Register(&Definition{
		Aliases: []string{"project:remove"},
		Handler: remove.Execute,
		Help:    "Remove project",
	})

	// Proxy commands
	Register(&Definition{
		Aliases: []string{"proxy:start"},
		Handler: func() { proxy.Execute("start") },
		Help:    "Start proxy",
	})
	Register(&Definition{
		Aliases: []string{"proxy:stop"},
		Handler: func() { proxy.Execute("stop") },
		Help:    "Stop proxy",
	})
	Register(&Definition{
		Aliases: []string{"proxy:restart"},
		Handler: func() { proxy.Execute("restart") },
		Help:    "Restart proxy",
	})
	Register(&Definition{
		Aliases: []string{"proxy:rebuild"},
		Handler: func() { proxy.Execute("rebuild") },
		Help:    "Rebuild proxy",
	})
	Register(&Definition{
		Aliases: []string{"proxy:reload"},
		Handler: func() { proxy.Execute("reload") },
		Help:    "Reload proxy config",
	})
	Register(&Definition{
		Aliases: []string{"proxy:prune"},
		Handler: func() { proxy.Execute("prune") },
		Help:    "Prune proxy",
	})

	// Prune and rebuild
	Register(&Definition{
		Aliases: []string{"prune"},
		Handler: prune.Execute,
		Help:    "Prune Docker resources",
	})
	Register(&Definition{
		Aliases: []string{"rebuild"},
		Handler: rebuild.Execute,
		Help:    "Rebuild containers",
	})

	// Remote sync commands
	Register(&Definition{
		Aliases: []string{"remote:sync:db"},
		Handler: db2.Execute,
		Help:    "Sync remote database",
	})
	Register(&Definition{
		Aliases: []string{"remote:sync:media"},
		Handler: media.Execute,
		Help:    "Sync remote media",
	})
	Register(&Definition{
		Aliases: []string{"remote:sync:file"},
		Handler: file.Execute,
		Help:    "Sync remote file",
	})

	// Restart
	Register(&Definition{
		Aliases: []string{"restart"},
		Handler: restart.Execute,
		Help:    "Restart containers",
	})

	// PWA
	Register(&Definition{
		Aliases: []string{"pwa"},
		Handler: pwa.Execute,
		Help:    "Execute PWA command",
	})

	// Scope commands
	Register(&Definition{
		Aliases: []string{"scope:add"},
		Handler: add.Execute,
		Help:    "Add scope",
	})
	Register(&Definition{
		Aliases: []string{"scope:list"},
		Handler: listScope.Execute,
		Help:    "List scopes. Supports --json (-j) output",
	})
	Register(&Definition{
		Aliases: []string{"scope:set"},
		Handler: set.Execute,
		Help:    "Set active scope",
	})

	// Service commands
	Register(&Definition{
		Aliases: []string{"service:list"},
		Handler: list.Execute,
		Help:    "List services. Supports --json (-j) output",
	})
	Register(&Definition{
		Aliases: []string{"service:enable"},
		Handler: enable.Execute,
		Help:    "Enable service",
	})
	Register(&Definition{
		Aliases: []string{"service:disable"},
		Handler: disable.Execute,
		Help:    "Disable service",
	})

	// Setup commands
	Register(&Definition{
		Aliases: []string{"setup"},
		Handler: setup.Execute,
		Help:    "Setup project",
	})
	Register(&Definition{
		Aliases: []string{"setup:env"},
		Handler: env.Execute,
		Help:    "Setup environment",
	})

	// Shopify commands
	Register(&Definition{
		Aliases: []string{"shopify", "sy"},
		Handler: shopify.Execute,
		Help:    "Execute Shopify CLI",
	})
	Register(&Definition{
		Aliases: []string{"shopify:web", "sy:w"},
		Handler: web.Execute,
		Help:    "Execute Shopify web",
	})
	Register(&Definition{
		Aliases: []string{"shopify:web:frontend", "sy:w:f"},
		Handler: frontend.Execute,
		Help:    "Execute Shopify frontend",
	})

	// Shopware commands
	Register(&Definition{
		Aliases: []string{"shopware", "sw"},
		Handler: shopware.Execute,
		Help:    "Execute Shopware CLI",
	})
	Register(&Definition{
		Aliases: []string{"shopware:bin", "sw:b"},
		Handler: shopware.ExecuteBin,
		Help:    "Execute Shopware bin/console",
	})

	// Snapshot commands
	Register(&Definition{
		Aliases: []string{"snapshot:create"},
		Handler: create.Execute,
		Help:    "Create snapshot",
	})
	Register(&Definition{
		Aliases: []string{"snapshot:restore"},
		Handler: restore.Execute,
		Help:    "Restore snapshot",
	})

	// SSL
	Register(&Definition{
		Aliases: []string{"ssl:rebuild"},
		Handler: ssl.Execute,
		Help:    "Rebuild SSL certificates",
	})

	// Start/Stop/Status
	Register(&Definition{
		Aliases: []string{"start"},
		Handler: start.Execute,
		Help:    "Start containers",
	})
	Register(&Definition{
		Aliases: []string{"status"},
		Handler: status.Execute,
		Help:    "Show container status. Supports --json (-j) output",
	})
	Register(&Definition{
		Aliases: []string{"stop"},
		Handler: stop.Execute,
		Help:    "Stop containers",
	})

	// Uncompress
	Register(&Definition{
		Aliases: []string{"uncompress"},
		Handler: compress.Unzip,
		Help:    "Uncompress project files",
	})
}
