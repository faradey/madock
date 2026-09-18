package info

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/service"
	"github.com/faradey/madock/v4/src/controller/platform"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/ports"
	inforeg "github.com/faradey/madock/v4/src/info"
)

type ArgsStruct struct {
	attr.Arguments
	ShowSecrets bool   `arg:"--show-secrets" help:"Print secret values in full where they are withheld by default"`
	Format      string `arg:"--format" help:"Output format: text (default), json, md, xml"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:    []string{"info"},
		Handler:    Info,
		Help:       "Show project info. Supports --format=text|json|md|xml and --json (-j) output",
		Category:   "general",
		ArgsType:   new(ArgsStruct),
		JSONOutput: true,
	})
}

func Info() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	format := args.Format
	if args.Json {
		if format != "" && format != "json" {
			logger.Fatal(fmt.Errorf("--json and --format=%s disagree; pass one of them", format))
		}
		format = "json"
	}
	if format == "" {
		format = "text"
	}
	// Refuse the format before any work is done: the platform half may take
	// half a minute, and an unknown format should not cost that.
	if err := Render(format, nil, io.Discard); err != nil {
		logger.Fatal(err)
	}

	projectConf := configs.GetCurrentProjectConfig()
	projectName := configs.GetProjectName()
	mainService := platform.GetMainService(projectConf)

	ctx := &inforeg.InfoContext{
		ProjectName: projectName,
		ProjectPath: paths.GetRunDirPath(),
		ProjectConf: projectConf,
		Service:     mainService,
	}

	blocks := collectGeneric(ctx, args.ShowSecrets)
	blocks = append(blocks, collectScopeBreakdown(projectName, projectConf)...)

	if handler, ok := inforeg.Get(projectConf["platform"]); ok {
		platformBlocks, err := handler.Collect(ctx)
		if err != nil {
			logger.Fatal(err)
		}
		blocks = append(blocks, platformBlocks...)
	}

	if err := Render(format, blocks, os.Stdout); err != nil {
		logger.Fatal(err)
	}
}

func collectGeneric(ctx *inforeg.InfoContext, showSecrets bool) []inforeg.Block {
	blocks := []inforeg.Block{}
	conf := ctx.ProjectConf
	platformName := conf["platform"]
	if platformName == "" {
		platformName = "unknown"
	}
	language := conf["language"]
	if language == "" {
		language = "php"
	}
	scope := conf["activeScope"]
	if scope == "" {
		scope = "default"
	}

	configPath := filepath.Join(paths.GetExecDirPath(), "aruntime", "projects", ctx.ProjectName, "config.xml")

	blocks = append(blocks, inforeg.Block{Key: "project", Title: "Project", Items: []inforeg.Item{
		{Key: "name", Value: ctx.ProjectName},
		{Key: "path", Value: ctx.ProjectPath},
		{Key: "platform", Value: platformName},
		{Key: "language", Value: language},
		{Key: "scope", Value: scope},
		{Key: "config", Value: configPath},
	}})

	hosts := configs.GetHosts(conf)
	if len(hosts) > 0 {
		items := make([]inforeg.Item, 0, len(hosts))
		seen := make(map[string]int, len(hosts))
		for _, h := range hosts {
			key := h["code"]
			seen[key]++
			if seen[key] > 1 {
				key = h["code"] + "#" + strconv.Itoa(seen[key])
			}
			items = append(items, inforeg.Item{Key: key, Value: h["name"]})
		}
		blocks = append(blocks, inforeg.Block{Key: "hosts", Title: "Hosts", Items: items})
	}

	if conf["db/database"] != "" || conf["db/user"] != "" {
		dbType := configs.GetDbType(conf)
		dbItems := []inforeg.Item{
			{Key: "type", Value: strings.ToUpper(dbType)},
			{Key: "host", Value: "db"},
		}
		if v := conf["db/database"]; v != "" {
			dbItems = append(dbItems, inforeg.Item{Key: "name", Value: v})
		}
		if v := conf["db/user"]; v != "" {
			dbItems = append(dbItems, inforeg.Item{Key: "user", Value: v})
		}
		if v := conf["db/password"]; v != "" {
			dbItems = append(dbItems, inforeg.Item{Key: "password", Value: fmtc.SecretOrValue(v, showSecrets)})
		}
		// Read-only lookup — do not allocate a port from `madock info`.
		if dbPort := ports.GetRegistry().Get(ctx.ProjectName, ports.ServiceDB); dbPort > 0 {
			dbItems = append(dbItems, inforeg.Item{Key: "remote", Value: "localhost:" + strconv.Itoa(dbPort)})
		}
		blocks = append(blocks, inforeg.Block{Key: "database", Title: "Database", Items: dbItems})
	}

	if services := collectEnabledServices(conf); len(services) > 0 {
		items := make([]inforeg.Item, 0, len(services))
		for _, svc := range services {
			items = append(items, inforeg.Item{Key: svc.name, Value: svc.version})
		}
		blocks = append(blocks, inforeg.Block{Key: "services", Title: "Services (scope: " + scope + ")", Items: items})
	}

	return blocks
}

// collectScopeBreakdown lists per-scope service overrides when more than one
// scope is defined. The active scope's merged view is already produced by
// collectGeneric; these blocks show the raw enabled set for each non-active
// scope so users can see what would change after `scope:set <name>`.
func collectScopeBreakdown(projectName string, conf map[string]string) []inforeg.Block {
	scopes := configs.GetScopes(projectName)
	if len(scopes) <= 1 {
		return nil
	}

	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
	if !paths.IsFileExist(configPath) {
		return nil
	}
	rawConf := configs.ParseXmlFile(configPath)

	active := conf["activeScope"]
	if active == "" {
		active = "default"
	}

	names := make([]string, 0, len(scopes))
	for s := range scopes {
		// "activeScope" is a config-pointer key, not a real scope.
		if s == active || s == "" || s == "activeScope" {
			continue
		}
		names = append(names, s)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil
	}

	blocks := make([]inforeg.Block, 0, len(names))
	for _, name := range names {
		services := collectScopedServices(rawConf, name)
		title := "Scope: " + name
		if len(services) == 0 {
			blocks = append(blocks, inforeg.Block{Key: "scopes", Name: name, Title: title,
				Items: []inforeg.Item{{Key: "services", Value: "(none configured)"}}})
			continue
		}
		items := make([]inforeg.Item, 0, len(services))
		for _, svc := range services {
			items = append(items, inforeg.Item{Key: svc.name, Value: svc.version})
		}
		blocks = append(blocks, inforeg.Block{Key: "scopes", Name: name, Title: title, Items: items})
	}
	return blocks
}

// collectScopedServices reads enabled services from the raw XML config
// under "scopes/<scope>/" prefix.
func collectScopedServices(rawConf map[string]string, scope string) []enabledService {
	prefix := "scopes/" + scope + "/"
	keys := make([]string, 0)
	for k, v := range rawConf {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		rel := strings.TrimPrefix(k, prefix)
		if !strings.HasSuffix(rel, "/enabled") || v != "true" {
			continue
		}
		keys = append(keys, rel)
	}
	sort.Strings(keys)

	out := make([]enabledService, 0, len(keys))
	for _, rel := range keys {
		base := strings.TrimSuffix(rel, "/enabled")
		name := service.GetByLong(base)
		if name == base {
			if idx := strings.LastIndex(base, "/"); idx >= 0 {
				name = base[idx+1:]
			}
		}
		version := rawConf[prefix+base+"/version"]
		if version == "" {
			version = "enabled"
		}
		out = append(out, enabledService{name: name, version: version})
	}
	return out
}

type enabledService struct {
	name    string
	version string
}

func collectEnabledServices(conf map[string]string) []enabledService {
	keys := make([]string, 0, len(conf))
	for k := range conf {
		if !strings.HasSuffix(k, "/enabled") || conf[k] != "true" {
			continue
		}
		// Skip scope-shadowed copies — scopes/<name>/... are overrides,
		// not first-class services.
		if strings.HasPrefix(k, "scopes/") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]enabledService, 0, len(keys))
	for _, k := range keys {
		base := strings.TrimSuffix(k, "/enabled")
		// Resolve canonical short name from the service registry
		// (e.g. "search/opensearch" → "opensearch", "db/phpmyadmin" → "phpmyadmin").
		// Falls back to the basename for keys that aren't in the registry.
		name := service.GetByLong(base)
		if name == base {
			if idx := strings.LastIndex(base, "/"); idx >= 0 {
				name = base[idx+1:]
			}
		}
		version := conf[base+"/version"]
		if version == "" {
			version = "enabled"
		}
		out = append(out, enabledService{name: name, version: version})
	}
	return out
}
