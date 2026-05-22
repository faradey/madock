package install

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/model/versions"
)

// shellSingleQuote wraps s for safe use inside a bash single-quoted string:
// every embedded ' is replaced with '\'' (close quote, escaped quote, reopen).
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// InstallHandler is called to install a platform for a given project.
type InstallHandler func(projectName, platformVersion string, projectConf map[string]string)

var installHandlers = map[string]InstallHandler{}

// RegisterInstallHandler registers an install handler for a platform.
func RegisterInstallHandler(platform string, handler InstallHandler) {
	installHandlers[platform] = handler
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"install"},
		Handler:  Execute,
		Help:     "Install Magento",
		Category: "general",
	})

	RegisterInstallHandler("magento2", func(projectName, platformVersion string, _ map[string]string) {
		Magento(projectName, platformVersion)
	})
	RegisterInstallHandler("shopware", func(projectName, platformVersion string, _ map[string]string) {
		Shopware(projectName, platformVersion, false)
	})
	RegisterInstallHandler("prestashop", func(projectName, platformVersion string, _ map[string]string) {
		PrestaShop(projectName, platformVersion, false)
	})
	RegisterInstallHandler("woocommerce", func(projectName, platformVersion string, _ map[string]string) {
		WooCommerce(projectName, platformVersion, false)
	})
	RegisterInstallHandler("medusa", func(projectName, platformVersion string, _ map[string]string) {
		Medusa(projectName, platformVersion)
	})
	RegisterInstallHandler("saleor", func(projectName, platformVersion string, _ map[string]string) {
		Saleor(projectName, platformVersion)
	})
}

func Execute() {
	projectConf := configs.GetCurrentProjectConfig()
	platform := projectConf["platform"]
	projectName := configs.GetProjectName()

	handler, ok := installHandlers[platform]
	if !ok {
		fmtc.Warning("This command is not supported for " + platform)
		return
	}

	platformVersion := ""
	if tv, found := versions.GetVersionsForPlatform(platform, ""); found {
		platformVersion = tv.PlatformVersion
	}

	handler(projectName, platformVersion, projectConf)
}

func Magento(projectName, platformVer string) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}
	installCommand := "bin/magento setup:install " +
		"--base-url=https://" + host + " " +
		"--db-host=db " +
		"--db-name=" + projectConf["db/database"] + " " +
		"--db-user=" + projectConf["db/user"] + " " +
		"--db-password=" + projectConf["db/password"] + " " +
		"--admin-firstname=" + projectConf["magento/admin_first_name"] + " " +
		"--admin-lastname=" + projectConf["magento/admin_last_name"] + " " +
		"--admin-email=" + projectConf["magento/admin_email"] + " " +
		"--admin-user=" + projectConf["magento/admin_user"] + " " +
		"--admin-password=" + projectConf["magento/admin_password"] + " " +
		"--backend-frontname=" + projectConf["magento/admin_frontname"] + " " +
		"--language=" + projectConf["magento/locale"] + " " +
		"--currency=" + projectConf["magento/currency"] + " " +
		"--timezone=" + projectConf["magento/timezone"] + " " +
		"--use-rewrites=1 "
	if platformVer >= "2.3.7" {
		if projectConf["search/elasticsearch/enabled"] == "true" {
			installCommand += "--search-engine=elasticsearch7 " +
				"--elasticsearch-host=elasticsearch " +
				"--elasticsearch-port=9200 " +
				"--elasticsearch-index-prefix=magento2 " +
				"--elasticsearch-timeout=15 "
		} else if projectConf["search/opensearch/enabled"] == "true" {
			if platformVer >= "2.4.6" {
				installCommand += "--search-engine=opensearch " +
					"--opensearch-host=opensearch " +
					"--opensearch-port=9200 " +
					"--opensearch-index-prefix=magento2 " +
					"--opensearch-timeout=15 "
			} else {
				installCommand += "--search-engine=elasticsearch7 " +
					"--elasticsearch-host=opensearch " +
					"--elasticsearch-port=9200 " +
					"--elasticsearch-index-prefix=magento2 " +
					"--elasticsearch-timeout=15 "
			}
		}

		if platformVer >= "2.4.6" {
			installCommand += "&& bin/magento module:disable Magento_AdminAdobeImsTwoFactorAuth "
		}
		installCommand += "&& bin/magento module:disable Magento_TwoFactorAuth "
	}
	installCommand += " && bin/magento setup:upgrade && bin/magento cache:clean && bin/magento indexer:reindex | bin/magento cache:flush"
	fmt.Println(installCommand)
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Magento installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Magento Admin URI: /" + projectConf["magento/admin_frontname"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin User: " + projectConf["magento/admin_user"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin Password: " + projectConf["magento/admin_password"])
}

func Shopware(projectName, platformVer string, isSampleData bool) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}

	installCommand := "sed -i 's/APP_URL=http:\\/\\/127.0.0.1:8000/APP_URL=https:\\/\\/" + host + "/g' .env "
	installCommand += "&& sed -i 's/DATABASE_URL=mysql:\\/\\/root:root@localhost\\/shopware/DATABASE_URL=mysql:\\/\\/" + projectConf["db/user"] + ":" + projectConf["db/password"] + "@db:3306\\/" + projectConf["db/database"] + "/g' .env "
	if projectConf["search/elasticsearch/enabled"] == "true" {
		installCommand += "&& sed -i 's/SHOPWARE_ES_ENABLED=0/SHOPWARE_ES_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/OPENSEARCH_URL=http:\\/\\/localhost:9200/OPENSEARCH_URL=http:\\/\\/elasticsearch:9200/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEXING_ENABLED=0/SHOPWARE_ES_INDEXING_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEX_PREFIX=sw/SHOPWARE_ES_INDEX_PREFIX=swlocal/g' .env "
	} else if projectConf["search/opensearch/enabled"] == "true" {
		installCommand += "&& sed -i 's/SHOPWARE_ES_ENABLED=0/SHOPWARE_ES_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/OPENSEARCH_URL=http:\\/\\/localhost:9200/OPENSEARCH_URL=opensearch:9200/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEXING_ENABLED=0/SHOPWARE_ES_INDEXING_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEX_PREFIX=sw/SHOPWARE_ES_INDEX_PREFIX=swlocal/g' .env "
	}

	// replace SHOPWARE_HTTP_CACHE_ENABLED=1 to SHOPWARE_HTTP_CACHE_ENABLED=0
	installCommand += "&& sed -i 's/SHOPWARE_HTTP_CACHE_ENABLED=1/SHOPWARE_HTTP_CACHE_ENABLED=0/g' .env "
	installCommand += "&& sed -i 's/STOREFRONT_PROXY_URL=http:\\/\\/localhost/STOREFRONT_PROXY_URL=https:\\/\\/" + host + "/g' .env "
	installCommand += "&& bin/console system:setup "
	installCommand += "&& bin/console system:install " +
		"--basic-setup " +
		"--shop-name=\"Your Shop Name\" " +
		"--shop-email=\"" + projectConf["magento/admin_email"] + "\" " +
		"--shop-locale=\"en-GB\" " +
		"--shop-currency=\"USD\" " +
		"&& composer update "

	if isSampleData {
		installCommand += "&& composer require swag/demo-data shopware/dev-tools && bin/console framework:demodata "
	}
	installCommand += "&& bin/console es:index "

	fmt.Println(installCommand)
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Shopware installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Shopware Admin URI: /admin")
	fmtc.SuccessLn("[SUCCESS]: Shopware Admin User: admin")
	fmtc.SuccessLn("[SUCCESS]: Shopware Admin Password: shopware")
}

func WooCommerce(projectName, platformVer string, isSampleData bool) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}

	installCommand := "wp config create " +
		"--dbname=" + projectConf["db/database"] + " " +
		"--dbuser=" + projectConf["db/user"] + " " +
		"--dbpass=" + projectConf["db/password"] + " " +
		"--dbhost=db " +
		"--path=" + projectConf["workdir"] + " " +
		"--force " +
		"&& wp core install " +
		"--url=https://" + host + " " +
		"--title='WooCommerce Store' " +
		"--admin_user=" + projectConf["magento/admin_user"] + " " +
		"--admin_password=" + projectConf["magento/admin_password"] + " " +
		"--admin_email=" + projectConf["magento/admin_email"] + " " +
		"--path=" + projectConf["workdir"] + " " +
		"&& wp plugin install woocommerce --activate " +
		"--path=" + projectConf["workdir"] + " " +
		"&& wp rewrite structure '/%postname%/' " +
		"--path=" + projectConf["workdir"] + " "

	if isSampleData {
		installCommand += "&& wp plugin install flavor flavor-starter --activate " +
			"--path=" + projectConf["workdir"] + " "
	}

	fmt.Println(installCommand)
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: WooCommerce installation complete.")
	fmtc.SuccessLn("[SUCCESS]: WooCommerce Store URL: https://" + host)
	fmtc.SuccessLn("[SUCCESS]: WordPress Admin URI: /wp-admin")
	fmtc.SuccessLn("[SUCCESS]: WordPress Admin User: " + projectConf["magento/admin_user"])
	fmtc.SuccessLn("[SUCCESS]: WordPress Admin Password: " + projectConf["magento/admin_password"])
}

func Medusa(projectName, platformVer string) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}
	if host == "" {
		host = "loc." + projectName + ".com"
	}

	// URL-encode credentials in case they contain reserved characters
	// like `@`, `:`, `/`, `?`, `#`. Without escaping the pg client
	// misparses the URL (e.g. a `@` in the password gets treated as
	// the user/host separator).
	dbUser := url.QueryEscape(projectConf["db/user"])
	dbPassword := url.QueryEscape(projectConf["db/password"])
	dbName := projectConf["db/database"]
	if dbName == "" {
		dbName = "db"
	}
	dbURL := "postgres://" + dbUser + ":" + dbPassword + "@db:5432/" + dbName + "?sslmode=disable"
	redisURL := "redis://redisdb:6379"

	// Use printf instead of a heredoc — embedding EOF inside a Go string
	// concatenated with `&& yarn install` puts the terminator on the same
	// line as the next command and bash never closes the heredoc.
	envBody := "DATABASE_URL=" + dbURL + "\n" +
		"REDIS_URL=" + redisURL + "\n" +
		"JWT_SECRET=supersecret\n" +
		"COOKIE_SECRET=supersecret\n" +
		"STORE_CORS=https://" + host + "\n" +
		"ADMIN_CORS=https://" + host + "\n" +
		"AUTH_CORS=https://" + host + "\n"
	envWrite := "printf '%s' " + shellSingleQuote(envBody) + " > .env"

	// Patch medusa-config.ts to allow the project's nginx host in
	// Medusa Admin's bundled Vite dev server. Vite 5+ rejects requests
	// whose Host header isn't in the allowedHosts list, which means
	// the project's *.test domain returns a "Blocked request" error
	// page until the user manually edits the config. Idempotent: skip
	// when the marker `allowedHosts` is already present.
	patchConfig := `node -e "const fs=require('fs');const p='medusa-config.ts';if(!fs.existsSync(p)){process.exit(0)}let c=fs.readFileSync(p,'utf8');if(c.includes('allowedHosts'))process.exit(0);if(!/\}\)\s*$/.test(c.trimEnd())){process.exit(0)}c=c.replace(/\}\)\s*$/,'  ,\n  admin: { vite: () => ({ server: { allowedHosts: true } }) },\n})');fs.writeFileSync(p,c);console.log('[madock] medusa-config.ts: admin.vite.server.allowedHosts=true');"`

	// Medusa's develop watcher chokidar hardcodes the ignore list and
	// only ignores top-level `node_modules`, not nested ones. With the
	// Next.js storefront cloned into ./storefront/, every file written
	// by storefront's `yarn install` triggers a backend reload, which
	// keeps the backend stuck in a restart loop and starves /health.
	// Inject regex literals (the existing entries already mix regex
	// and string forms) so the matcher catches any `/storefront/` and
	// any `/node_modules/` segment regardless of nesting depth.
	// Idempotent and best-effort.
	patchWatcher := `node -e "const fs=require('fs');const p='node_modules/@medusajs/medusa/dist/commands/develop.js';if(!fs.existsSync(p))process.exit(0);let c=fs.readFileSync(p,'utf8');if(c.includes('madock-watch-patch'))process.exit(0);if(!c.includes('\"src/admin\"'))process.exit(0);c=c.replace(/\"src\/admin\",/,'\"src/admin\",new RegExp(\"storefront\"),new RegExp(\"node_modules\"),/* madock-watch-patch */ ');fs.writeFileSync(p,c);console.log('[madock] medusa develop.js: storefront and nested node_modules added to watch ignore');"`

	// `db:setup` is the umbrella command that creates the database
	// (no-op when it already exists), runs all module migrations,
	// runs the standalone migration scripts (e.g. seed roles), and
	// syncs links. `db:migrate` alone leaves the migration scripts
	// pending, so post-install boot hits "Loaders for module Tax
	// failed: relation tax_provider does not exist" until a separate
	// db:migrate:scripts run kicks them off.
	// `yarn seed` runs the starter's bundled seed.ts when present —
	// it provisions the Europe region, sales channel, shipping
	// options, and demo products that the Next.js storefront expects.
	// Without it the storefront middleware errors out with "No
	// regions found" before any page renders. Guarded by package.json
	// so non-starter projects don't fail when the script is missing.
	seedIfPresent := "if node -e \"process.exit(((require('./package.json').scripts||{}).seed)?0:1)\" 2>/dev/null; then yarn seed; fi"

	installCommand := envWrite +
		" && yarn install" +
		" && " + patchConfig +
		" && " + patchWatcher +
		" && npx medusa db:setup --db " + dbName +
		" && npx medusa user --email admin@example.com --password admin" +
		" && " + seedIfPresent

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	fmt.Println(installCommand)
	nodejsContainer := docker.GetContainerName(projectConf, projectName, "nodejs")
	err := docker.ContainerExec(nodejsContainer, "node", true, "bash", "-c", "cd "+workdir+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}

	// Restart the nodejs container so the smart entrypoint runs again
	// from scratch and starts `yarn dev` against the freshly migrated
	// database. Without the restart, PID 1 is still parked in the
	// wait-for-deps loop at the moment install completes; it does
	// fall through to yarn dev on its own, but that boot race
	// against the final migration scripts can leave Medusa with
	// cached "tax_provider does not exist" loader failures that
	// stick around until the container is recreated.
	if rerr := exec.Command("docker", "restart", nodejsContainer).Run(); rerr != nil {
		fmtc.WarningLn("Could not restart nodejs container automatically: " + rerr.Error() + ". Run `madock restart` manually if the dev server doesn't pick up.")
	}

	// Create a default publishable API key so /store/* endpoints are
	// usable out of the box. Medusa v2 rejects every storefront
	// request without an x-publishable-api-key header (HTTP 400),
	// and the key can only be created through the admin API once the
	// server is up. Wait for /health, log in as the admin we just
	// created, create the key, and write it into both the project
	// .env and the storefront/.env (when the storefront subfolder is
	// present). Best-effort: a failure here only prints a warning.
	publishableKey := createPublishableKey(nodejsContainer, host, workdir)

	// Install the Next.js storefront when it was cloned alongside the
	// backend. The storefront container shares the same project src/
	// mount, so .env.local + node_modules end up on the host
	// filesystem. After install we restart the storefront container so
	// its smart entrypoint picks up the freshly seeded env + deps.
	if projectConf["medusa/storefront/enabled"] == "true" {
		installStorefront(projectConf, projectName, host, publishableKey)
	}

	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Medusa installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Medusa Storefront URL: https://" + host)
	fmtc.SuccessLn("[SUCCESS]: Medusa Admin URI: /app")
	fmtc.SuccessLn("[SUCCESS]: Medusa Admin User: admin@example.com")
	fmtc.SuccessLn("[SUCCESS]: Medusa Admin Password: admin")
	if publishableKey != "" {
		fmtc.SuccessLn("[SUCCESS]: Publishable API key: " + publishableKey)
	}
}

// createPublishableKey waits for the Medusa dev server, logs in as the
// default admin, reuses an existing publishable API key (Medusa's
// `db:setup` seeds one bound to the default sales channel) or creates
// one and binds it, then writes the token into the project .env (and
// storefront/.env when the folder exists). Returns the key string on
// success, empty string on any failure — the caller just logs a
// warning and continues.
func createPublishableKey(nodejsContainer, host, workdir string) string {
	healthURL := "https://" + host + "/health"
	authURL := "https://" + host + "/auth/user/emailpass"
	keysURL := "https://" + host + "/admin/api-keys"
	channelsURL := "https://" + host + "/admin/sales-channels"

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: insecureTLSConfig(),
		},
	}

	// Wait up to 3 minutes for /health to return 200 — fresh Medusa
	// dev boot can take 60-90s while the admin Vite bundle compiles.
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		time.Sleep(3 * time.Second)
	}

	loginBody := strings.NewReader(`{"email":"admin@example.com","password":"admin"}`)
	loginReq, _ := http.NewRequest("POST", authURL, loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := httpClient.Do(loginReq)
	if err != nil {
		fmtc.WarningLn("Could not log in to seed the publishable API key: " + err.Error() + ". Create one manually from the admin UI.")
		return ""
	}
	defer loginResp.Body.Close()
	var loginPayload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginPayload); err != nil || loginPayload.Token == "" {
		fmtc.WarningLn("Admin login returned an unexpected payload while seeding the publishable API key.")
		return ""
	}
	bearer := "Bearer " + loginPayload.Token

	// Look for an existing publishable key already bound to a sales
	// channel. Medusa's `db:setup` seeds "Default Publishable API Key"
	// bound to the default sales channel — reuse it rather than
	// stacking a second key that storefronts would have to pick from.
	type apiKey struct {
		ID            string `json:"id"`
		Token         string `json:"token"`
		Type          string `json:"type"`
		SalesChannels []struct {
			ID string `json:"id"`
		} `json:"sales_channels"`
	}
	listReq, _ := http.NewRequest("GET", keysURL+"?type=publishable&limit=50", nil)
	listReq.Header.Set("Authorization", bearer)
	if listResp, lerr := httpClient.Do(listReq); lerr == nil {
		defer listResp.Body.Close()
		var listPayload struct {
			APIKeys []apiKey `json:"api_keys"`
		}
		if json.NewDecoder(listResp.Body).Decode(&listPayload) == nil {
			for _, k := range listPayload.APIKeys {
				if k.Token != "" && len(k.SalesChannels) > 0 {
					writeKeyToEnv(nodejsContainer, workdir, k.Token)
					return k.Token
				}
			}
		}
	}

	// No usable key — create one and bind it to the default sales
	// channel so /store/* requests with this token pass the v2
	// publishable-key gate.
	keyBody := strings.NewReader(`{"title":"madock-default","type":"publishable"}`)
	keyReq, _ := http.NewRequest("POST", keysURL, keyBody)
	keyReq.Header.Set("Content-Type", "application/json")
	keyReq.Header.Set("Authorization", bearer)
	keyResp, err := httpClient.Do(keyReq)
	if err != nil {
		fmtc.WarningLn("Could not create a publishable API key: " + err.Error())
		return ""
	}
	defer keyResp.Body.Close()
	var keyPayload struct {
		APIKey apiKey `json:"api_key"`
	}
	if err := json.NewDecoder(keyResp.Body).Decode(&keyPayload); err != nil || keyPayload.APIKey.Token == "" {
		fmtc.WarningLn("Publishable API key endpoint returned an unexpected payload.")
		return ""
	}

	chReq, _ := http.NewRequest("GET", channelsURL+"?limit=1", nil)
	chReq.Header.Set("Authorization", bearer)
	chResp, cerr := httpClient.Do(chReq)
	if cerr == nil {
		defer chResp.Body.Close()
		var chPayload struct {
			SalesChannels []struct {
				ID string `json:"id"`
			} `json:"sales_channels"`
		}
		if json.NewDecoder(chResp.Body).Decode(&chPayload) == nil && len(chPayload.SalesChannels) > 0 {
			bindBody := strings.NewReader(`{"add":["` + chPayload.SalesChannels[0].ID + `"]}`)
			bindReq, _ := http.NewRequest("POST", keysURL+"/"+keyPayload.APIKey.ID+"/sales-channels", bindBody)
			bindReq.Header.Set("Content-Type", "application/json")
			bindReq.Header.Set("Authorization", bearer)
			if bindResp, berr := httpClient.Do(bindReq); berr == nil {
				bindResp.Body.Close()
			}
		}
	}

	writeKeyToEnv(nodejsContainer, workdir, keyPayload.APIKey.Token)
	return keyPayload.APIKey.Token
}

// writeKeyToEnv appends NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY to the
// backend .env. Uses `docker exec` so we don't have to know which host
// path is bind mounted — the container always sees .env at workdir.
// installStorefront owns the storefront's .env.local separately.
//
// The leading newline guards against the case where Medusa's CLI
// leaves the file without a trailing \n (observed: `db:setup` rewrites
// .env and appends `DB_NAME=<db>` without a final newline). Without
// it, our line would glue onto the previous key.
func writeKeyToEnv(nodejsContainer, workdir, token string) {
	cmd := "cd " + workdir +
		" && (grep -q '^NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=' .env 2>/dev/null || printf '\\nNEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=%s\\n' " + token + " >> .env)"
	_ = exec.Command("docker", "exec", "-u", "node", nodejsContainer, "bash", "-c", cmd).Run()
}

// installStorefront runs `yarn install` inside the storefront container
// and writes .env.local with the backend URLs + publishable key so the
// Next.js dev server starts cleanly on port 8000. Best-effort: any
// failure prints a warning, the rest of the Medusa install still
// succeeds.
func installStorefront(projectConf map[string]string, projectName, host, publishableKey string) {
	storefrontWorkdir := projectConf["medusa/storefront/workdir"]
	if storefrontWorkdir == "" {
		storefrontWorkdir = "/var/www/html/storefront"
	}
	region := projectConf["medusa/storefront/region"]
	if region == "" {
		region = "gb"
	}
	publicBackendURL := projectConf["medusa/storefront/public_backend_url"]
	if publicBackendURL == "" {
		publicBackendURL = "https://" + host
	}

	// Browser-side calls use the public host (HTTPS through nginx);
	// server-side (SSR) calls hit the backend on the docker network.
	envBody := "NEXT_PUBLIC_MEDUSA_BACKEND_URL=" + publicBackendURL + "\n" +
		"MEDUSA_BACKEND_URL=http://nodejs:9000\n" +
		"NEXT_PUBLIC_BASE_URL=" + publicBackendURL + "\n" +
		"NEXT_PUBLIC_DEFAULT_REGION=" + region + "\n"
	if publishableKey != "" {
		envBody += "NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=" + publishableKey + "\n"
	}
	envWrite := "printf '%s' " + shellSingleQuote(envBody) + " > .env.local"
	installCommand := envWrite + " && yarn install"

	storefrontContainer := docker.GetContainerName(projectConf, projectName, "storefront")
	fmtc.InfoIconLn("Installing Medusa storefront in " + storefrontContainer)
	if err := docker.ContainerExec(storefrontContainer, "node", true, "bash", "-c", "cd "+storefrontWorkdir+" && "+installCommand); err != nil {
		fmtc.WarningLn("Storefront install failed: " + err.Error() + ". Inspect with `madock logs storefront`.")
		return
	}

	// Restart so the smart entrypoint picks up the freshly written
	// .env.local and node_modules and boots `yarn dev`.
	if rerr := exec.Command("docker", "restart", storefrontContainer).Run(); rerr != nil {
		fmtc.WarningLn("Could not restart storefront container automatically: " + rerr.Error() + ". Run `madock restart` manually if the dev server doesn't pick up.")
	}
}

// insecureTLSConfig returns a tls.Config that skips verification, used
// for the local self-signed nginx cert. madock's proxy uses a local CA
// the host shell trusts via the SSL install step, but the install
// command runs while that trust may not yet be in place.
func insecureTLSConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

func Saleor(projectName, platformVer string) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}
	if host == "" {
		host = "loc." + projectName + ".com"
	}

	// URL-encode db creds — same rationale as Medusa.
	dbUser := url.QueryEscape(projectConf["db/user"])
	dbPassword := url.QueryEscape(projectConf["db/password"])
	dbName := projectConf["db/database"]
	if dbName == "" {
		dbName = "saleor"
	}
	dbURL := "postgres://" + dbUser + ":" + dbPassword + "@db:5432/" + dbName + "?sslmode=disable"
	redisURL := "redis://redisdb:6379/0"
	celeryBroker := "redis://redisdb:6379/1"

	envBody := "SECRET_KEY=changeme-madock-dev\n" +
		"DEBUG=True\n" +
		"DATABASE_URL=" + dbURL + "\n" +
		"REDIS_URL=" + redisURL + "\n" +
		"CACHE_URL=" + redisURL + "\n" +
		"CELERY_BROKER_URL=" + celeryBroker + "\n" +
		"ALLOWED_HOSTS=*\n" +
		"ALLOWED_CLIENT_HOSTS=" + host + ",localhost\n" +
		"DEFAULT_FROM_EMAIL=noreply@" + host + "\n" +
		"EMAIL_URL=smtp://mailpit:1025\n" +
		"PUBLIC_URL=https://" + host + "\n"
	envWrite := "printf '%s' " + shellSingleQuote(envBody) + " > .env"

	// Pick `uv` when the project ships a uv.lock (Saleor 3.21+);
	// fall back to pip for older releases.
	bootstrap := "if [ -f uv.lock ] && command -v uv >/dev/null 2>&1; then" +
		" uv sync --frozen;" +
		" RUN_PY='uv run python';" +
		"else" +
		" if [ -f requirements_dev.txt ]; then pip install -r requirements_dev.txt;" +
		" elif [ -f requirements.txt ]; then pip install -r requirements.txt;" +
		" elif [ -f pyproject.toml ]; then pip install -e .;" +
		" fi;" +
		" RUN_PY='python';" +
		"fi"

	// `bootstrap` may export RUN_PY=… inside its `if` branches; ensure
	// every subsequent step also picks up the .env values that Saleor
	// expects in process env.
	loadEnv := "set -a && . ./.env && set +a"
	installCommand := envWrite +
		" && " + loadEnv +
		" && " + bootstrap +
		" && $RUN_PY manage.py migrate" +
		" && $RUN_PY manage.py populatedb --createsuperuser"

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	fmt.Println(installCommand)
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "python"), "saleor", true, "bash", "-c", "cd "+workdir+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Saleor installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Saleor API URL: https://" + host + "/graphql/")
	fmtc.SuccessLn("[SUCCESS]: Saleor Admin User: admin@example.com")
	fmtc.SuccessLn("[SUCCESS]: Saleor Admin Password: admin")
	fmtc.SuccessLn("[SUCCESS]: Dashboard: enable with `madock service:enable dashboard`")
}

func PrestaShop(projectName, platformVer string, isSampleData bool) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}

	installCommand := "php install/index_cli.php " +
		"--domain=" + host + " " +
		"--db_server=db " +
		"--db_name=" + projectConf["db/database"] + " " +
		"--db_user=" + projectConf["db/user"] + " " +
		"--db_password=" + projectConf["db/password"] + " " +
		"--firstname=" + projectConf["magento/admin_first_name"] + " " +
		"--lastname=" + projectConf["magento/admin_last_name"] + " " +
		"--email=" + projectConf["magento/admin_email"] + " " +
		"--password=" + projectConf["magento/admin_password"] + " " +
		"--timezone=" + projectConf["magento/timezone"] + " " +
		"--rewrite=1 " + " " +
		"--ssl=1 "

	if isSampleData {
		installCommand += " --fixtures=1 "
	}

	fmt.Println(installCommand)
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: PrestaShop installation complete.")
	fmtc.SuccessLn("[SUCCESS]: PrestaShop Admin URI: /admin")
	fmtc.SuccessLn("[SUCCESS]: PrestaShop Admin User: " + projectConf["magento/admin_email"])
	fmtc.SuccessLn("[SUCCESS]: PrestaShop Admin Password: " + projectConf["magento/admin_password"])
}
