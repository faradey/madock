package info

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/service"
	"github.com/faradey/madock/v3/src/controller/platform"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/ports"
	inforeg "github.com/faradey/madock/v3/src/info"
)

type ArgsStruct struct {
	attr.Arguments
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"info"},
		Handler:  Info,
		Help:     "Show project info",
		Category: "general",
		ArgsType: new(ArgsStruct),
	})
}

func Info() {
	attr.Parse(new(ArgsStruct))

	projectConf := configs.GetCurrentProjectConfig()
	projectName := configs.GetProjectName()
	mainService := platform.GetMainService(projectConf)

	ctx := &inforeg.InfoContext{
		ProjectName: projectName,
		ProjectPath: paths.GetRunDirPath(),
		ProjectConf: projectConf,
		Service:     mainService,
	}

	printGeneric(ctx)
	printScopeBreakdown(projectName, projectConf)

	if handler, ok := inforeg.Get(projectConf["platform"]); ok {
		fmt.Println("")
		if err := handler.Print(ctx); err != nil {
			logger.Fatal(err)
		}
	}
}

func printGeneric(ctx *inforeg.InfoContext) {
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

	fmtc.Section("Project", []fmtc.SectionItem{
		{Key: "name", Value: ctx.ProjectName},
		{Key: "path", Value: ctx.ProjectPath},
		{Key: "platform", Value: platformName},
		{Key: "language", Value: language},
		{Key: "scope", Value: scope},
		{Key: "config", Value: configPath},
	})

	hosts := configs.GetHosts(conf)
	if len(hosts) > 0 {
		items := make([]fmtc.SectionItem, 0, len(hosts))
		seen := make(map[string]int, len(hosts))
		for _, h := range hosts {
			key := h["code"]
			seen[key]++
			if seen[key] > 1 {
				key = h["code"] + "#" + strconv.Itoa(seen[key])
			}
			items = append(items, fmtc.SectionItem{Key: key, Value: h["name"]})
		}
		fmtc.Section("Hosts", items)
	}

	if conf["db/database"] != "" || conf["db/user"] != "" {
		dbType := configs.GetDbType(conf)
		dbItems := []fmtc.SectionItem{
			{Key: "type", Value: strings.ToUpper(dbType)},
			{Key: "host", Value: "db"},
		}
		if v := conf["db/database"]; v != "" {
			dbItems = append(dbItems, fmtc.SectionItem{Key: "name", Value: v})
		}
		if v := conf["db/user"]; v != "" {
			dbItems = append(dbItems, fmtc.SectionItem{Key: "user", Value: v})
		}
		if v := conf["db/password"]; v != "" {
			dbItems = append(dbItems, fmtc.SectionItem{Key: "password", Value: maskPassword(v)})
		}
		// Read-only lookup — do not allocate a port from `madock info`.
		if dbPort := ports.GetRegistry().Get(ctx.ProjectName, ports.ServiceDB); dbPort > 0 {
			dbItems = append(dbItems, fmtc.SectionItem{Key: "remote", Value: "localhost:" + strconv.Itoa(dbPort)})
		}
		fmtc.Section("Database", dbItems)
	}

	if services := collectEnabledServices(conf); len(services) > 0 {
		items := make([]fmtc.SectionItem, 0, len(services))
		for _, svc := range services {
			items = append(items, fmtc.SectionItem{Key: svc.name, Value: svc.version})
		}
		fmtc.Section("Services (scope: "+scope+")", items)
	}
}

// printScopeBreakdown lists per-scope service overrides when more than one
// scope is defined. The active scope's merged view is already printed by
// printGeneric; this section shows the raw enabled set for each non-active
// scope so users can see what would change after `scope:set <name>`.
func printScopeBreakdown(projectName string, conf map[string]string) {
	scopes := configs.GetScopes(projectName)
	if len(scopes) <= 1 {
		return
	}

	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
	if !paths.IsFileExist(configPath) {
		return
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
		return
	}

	for _, name := range names {
		services := collectScopedServices(rawConf, name)
		title := "Scope: " + name
		if len(services) == 0 {
			fmtc.Section(title, []fmtc.SectionItem{{Key: "services", Value: "(none configured)"}})
			continue
		}
		items := make([]fmtc.SectionItem, 0, len(services))
		for _, svc := range services {
			items = append(items, fmtc.SectionItem{Key: svc.name, Value: svc.version})
		}
		fmtc.Section(title, items)
	}
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

func maskPassword(v string) string {
	r := []rune(v)
	n := len(r)
	if n <= 2 {
		return strings.Repeat("*", n)
	}
	if n <= 4 {
		return string(r[0]) + strings.Repeat("*", n-1)
	}
	return string(r[0]) + strings.Repeat("*", n-2) + string(r[n-1])
}
