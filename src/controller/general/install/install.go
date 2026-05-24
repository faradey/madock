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
	RegisterInstallHandler("spree", func(projectName, platformVersion string, _ map[string]string) {
		Spree(projectName, platformVersion)
	})
	RegisterInstallHandler("sylius", func(projectName, platformVersion string, _ map[string]string) {
		Sylius(projectName, platformVersion, false)
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

func Spree(projectName, platformVer string) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}
	if host == "" {
		host = "loc." + projectName + ".com"
	}

	// URL-encode db creds — same rationale as Medusa/Saleor: characters
	// like `@`, `:`, `/`, `?`, `#` would otherwise misparse in the
	// connection URL.
	dbUser := url.QueryEscape(projectConf["db/user"])
	dbPassword := url.QueryEscape(projectConf["db/password"])
	dbName := projectConf["db/database"]
	if dbName == "" {
		dbName = "spree"
	}
	dbURL := "postgresql://" + dbUser + ":" + dbPassword + "@db:5432/" + dbName
	redisURL := "redis://redisdb:6379/0"

	// SECRET_KEY_BASE is deterministic-but-not-secret in a local dev
	// env: Rails refuses to boot without it once RAILS_ENV is set, but
	// nothing about a madock-local project should be reachable from
	// the public internet, so the hex value being checked in is fine.
	envBody := "DATABASE_URL=" + dbURL + "\n" +
		"REDIS_URL=" + redisURL + "\n" +
		"RAILS_ENV=development\n" +
		"NODE_ENV=development\n" +
		"SECRET_KEY_BASE=changeme0000000000000000000000000000000000000000000000000000000000madocklocaldev\n" +
		"DISABLE_SPRING=1\n" +
		"BINDING=0.0.0.0\n" +
		"PORT=3000\n" +
		"ADMIN_EMAIL=admin@example.com\n" +
		"ADMIN_PASSWORD=spree123\n" +
		"SPREE_ADMIN_EMAIL=admin@example.com\n" +
		"SPREE_ADMIN_PASSWORD=spree123\n" +
		"STORE_URL=https://" + host + "\n"
	envWrite := "printf '%s' " + shellSingleQuote(envBody) + " > .env"

	// spree_starter ships a `bin/setup` script, but it tries to
	// `sudo apt-get install libpq-dev libvips-dev` and shells out to
	// mise/brew — none of which exist in our slim ruby image (sudo is
	// missing too). We bring the equivalent system packages in the
	// Dockerfile, so run the install steps directly. The
	// `spree:search:reindex` task is optional; ignore failures so a
	// search backend mismatch doesn't break the whole install.
	//
	// Pin `.ruby-version` and Gemfile.lock's RUBY VERSION line to the
	// image's actual Ruby so bundler doesn't bail with "Your Ruby
	// version is X, but your Gemfile specified Y" — Spree starter
	// pins a patch level (e.g. 4.0.1) that docker hub doesn't always
	// ship.
	pinRuby := `RUBY_VER=$(ruby -e "print RUBY_VERSION"); ` +
		`if [ -f .ruby-version ]; then echo "$RUBY_VER" > .ruby-version; fi; ` +
		`if [ -f Gemfile.lock ]; then sed -i "s/^   ruby [0-9.]*p*[0-9]*$/   ruby $RUBY_VER/" Gemfile.lock; fi`

	// nginx terminates TLS and proxies to puma over plain HTTP inside
	// the docker network. Without `config.assume_ssl = true` Rails
	// generates http:// in every redirect (login, callbacks, etc) and
	// browsers warn / lose the secure cookie flag. Also clear
	// `config.hosts` so the Next.js storefront's server-side fetches
	// to `http://ruby:3000` (inside the docker network) don't trip
	// Rails' HostAuthorization middleware — it ships a default
	// whitelist of .localhost / 127.0.0.1 / ::1 and rejects any other
	// hostname with "Blocked hosts" 403, which then surfaces in the
	// storefront as JSON parse errors and meta-refresh loops.
	// Idempotent.
	patchSsl := `if [ -f config/environments/development.rb ]; then ` +
		`grep -q 'madock-ssl-patch' config/environments/development.rb 2>/dev/null || ` +
		`sed -i '/^Rails.application.configure do$/a\  # madock-ssl-patch\n  config.assume_ssl = true\n  config.hosts.clear' config/environments/development.rb; ` +
		`fi`

	// Spree admin uses tailwindcss-rails. spree_starter's Procfile.dev
	// runs `bin/rails spree:admin:tailwindcss:watch` alongside puma;
	// we build it once during install so the admin UI loads on first
	// request. Without it `/admin_user/sign_in` 500s with
	// `Propshaft::MissingAssetError (spree/admin/application.css)`.
	bootstrap := pinRuby +
		" && " + patchSsl +
		" && bundle install" +
		" && bundle exec rails db:prepare" +
		" && (bundle exec rails spree:admin:tailwindcss:build || true)" +
		" && (bundle exec rails spree:search:reindex || true)" +
		" && (bundle exec rails spree_sample:load || true)"

	loadEnv := "set -a && . ./.env && set +a"
	installCommand := envWrite +
		" && " + loadEnv +
		" && " + bootstrap

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	fmt.Println(installCommand)
	rubyContainer := docker.GetContainerName(projectConf, projectName, "ruby")
	err := docker.ContainerExec(rubyContainer, "ruby", true, "bash", "-c", "cd "+workdir+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}

	// Restart the ruby container so the smart entrypoint runs again
	// from scratch and starts `rails server` against the freshly
	// migrated database.
	if rerr := exec.Command("docker", "restart", rubyContainer).Run(); rerr != nil {
		fmtc.WarningLn("Could not restart ruby container automatically: " + rerr.Error() + ". Run `madock restart` manually if the dev server doesn't pick up.")
	}

	// Install the Next.js storefront when it was cloned alongside the
	// backend. The storefront container shares the same project src/
	// mount, so .env.local + node_modules end up on the host
	// filesystem. After install we restart the storefront container so
	// its smart entrypoint picks up the freshly seeded env + deps.
	publishableKey := ""
	if projectConf["spree/storefront/enabled"] == "true" {
		publishableKey = installSpreeStorefront(projectConf, projectName, host, rubyContainer, workdir)
	}

	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Spree installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Spree Storefront URL: https://" + host)
	fmtc.SuccessLn("[SUCCESS]: Spree Admin URI: /admin")
	fmtc.SuccessLn("[SUCCESS]: Spree Admin User: admin@example.com")
	fmtc.SuccessLn("[SUCCESS]: Spree Admin Password: spree123")
	if publishableKey != "" {
		fmtc.SuccessLn("[SUCCESS]: Spree Publishable API key: " + publishableKey)
	}
}

// installSpreeStorefront runs `yarn install` inside the storefront
// container and writes storefront/.env.local with the backend URL +
// publishable API key + locale defaults so the Next.js dev server
// starts cleanly on port 3001. Returns the publishable key that was
// wired in (empty when extraction failed).
func installSpreeStorefront(projectConf map[string]string, projectName, host, rubyContainer, backendWorkdir string) string {
	storefrontWorkdir := projectConf["spree/storefront/workdir"]
	if storefrontWorkdir == "" {
		storefrontWorkdir = "/var/www/html/storefront"
	}
	country := projectConf["spree/storefront/country"]
	if country == "" {
		country = "us"
	}
	locale := projectConf["spree/storefront/locale"]
	if locale == "" {
		locale = "en"
	}
	storeName := projectConf["spree/storefront/store_name"]
	if storeName == "" {
		storeName = "Spree Store"
	}
	siteURL := projectConf["spree/storefront/site_url"]
	if siteURL == "" {
		siteURL = "https://" + host
	}

	// Pull the publishable API key seeded by `spree_sample:load` (or
	// the one Spree provisions during db:prepare). Fall back to an
	// empty string + warning when nothing is found — the storefront
	// will surface a clear error and the user can paste a key.
	publishableKey := extractSprePublishableKey(rubyContainer, backendWorkdir)
	if publishableKey == "" {
		fmtc.WarningLn("Could not extract a Spree publishable API key from the backend. The storefront will boot but API calls will fail until you set SPREE_PUBLISHABLE_KEY in storefront/.env.local manually.")
	}

	// Quote values so the smart entrypoint's `set -a; . ./.env.local`
	// doesn't blow up on values containing spaces (e.g. the default
	// store name "Spree Store" — POSIX sh would otherwise parse the
	// line as `NEXT_PUBLIC_STORE_NAME=Spree` followed by `Store` as
	// a command). All values are also useful as Next.js env vars
	// either way, since Next reads .env.local directly.
	envBody := "SPREE_API_URL=\"http://ruby:3000\"\n" +
		"NEXT_PUBLIC_SITE_URL=\"" + siteURL + "\"\n" +
		"NEXT_PUBLIC_DEFAULT_COUNTRY=\"" + country + "\"\n" +
		"NEXT_PUBLIC_DEFAULT_LOCALE=\"" + locale + "\"\n" +
		"NEXT_PUBLIC_STORE_NAME=\"" + storeName + "\"\n"
	if publishableKey != "" {
		envBody += "SPREE_PUBLISHABLE_KEY=\"" + publishableKey + "\"\n"
	}
	envWrite := "printf '%s' " + shellSingleQuote(envBody) + " > .env.local"

	// Patch next.config.ts (or .js/.mjs) to whitelist the project's
	// nginx host under `allowedDevOrigins`. Next.js 15+ blocks
	// cross-origin HMR WebSocket requests during dev — the storefront
	// sees its own origin as `spree.test` (the nginx host) while the
	// dev server binds to localhost, so /_next/webpack-hmr returns
	// 403 and the browser console spams `WebSocket connection failed`.
	// Idempotent via marker comment.
	patchNextConfig := `node -e "const fs=require('fs');for(const p of ['next.config.ts','next.config.js','next.config.mjs']){if(!fs.existsSync(p))continue;let c=fs.readFileSync(p,'utf8');if(c.includes('madock-allowed-host')){process.exit(0)}if(!/allowedDevOrigins\s*:\s*\[/.test(c)){process.exit(0)}c=c.replace(/allowedDevOrigins\s*:\s*\[/,'allowedDevOrigins: [\"` + host + `\", \"*.test\", /* madock-allowed-host */ ');fs.writeFileSync(p,c);console.log('[madock] '+p+': added '+'` + host + `'+' to allowedDevOrigins');break}"`

	installCommand := envWrite + " && " + patchNextConfig + " && yarn install"

	storefrontContainer := docker.GetContainerName(projectConf, projectName, "storefront")
	fmtc.InfoIconLn("Installing Spree storefront in " + storefrontContainer)
	if err := docker.ContainerExec(storefrontContainer, "node", true, "bash", "-c", "cd "+storefrontWorkdir+" && "+installCommand); err != nil {
		fmtc.WarningLn("Storefront install failed: " + err.Error() + ". Inspect with `madock logs storefront`.")
		return publishableKey
	}

	if rerr := exec.Command("docker", "restart", storefrontContainer).Run(); rerr != nil {
		fmtc.WarningLn("Could not restart storefront container automatically: " + rerr.Error() + ". Run `madock restart` manually if the dev server doesn't pick up.")
	}

	return publishableKey
}

// extractSprePublishableKey runs `rails runner` against the backend
// container to grab the first publishable Spree::ApiKey token (Spree 5
// seeds one during sample data load, bound to the default store).
// Returns empty string when nothing is found or rails errors out.
func extractSprePublishableKey(rubyContainer, workdir string) string {
	script := `set -a && [ -f .env ] && . ./.env && set +a; ` +
		`bundle exec rails runner "k = Spree::ApiKey.where(key_type: 'publishable', revoked_at: nil).order(:id).first; puts(k ? k.token : '')" 2>/dev/null`
	out, err := exec.Command("docker", "exec", "-u", "ruby", "-w", workdir, rubyContainer, "bash", "-lc", script).Output()
	if err != nil {
		return ""
	}
	// rails runner can print warnings before the actual output; take
	// the last non-empty line that starts with "pk_".
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "pk_") {
			return line
		}
	}
	return ""
}

func Sylius(projectName, platformVer string, isSampleData bool) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}
	if host == "" {
		host = "loc." + projectName + ".com"
	}

	// URL-encode db creds — passwords with `@:/?#` would otherwise
	// break the Doctrine DSN parser.
	dbUser := url.QueryEscape(projectConf["db/user"])
	dbPassword := url.QueryEscape(projectConf["db/password"])
	dbName := projectConf["db/database"]
	if dbName == "" {
		dbName = "sylius"
	}
	// Doctrine serverVersion needs a 3-segment value (`major.minor.patch`)
	// — its regex bails on shorter strings ("Invalid platform version"
	// in FileLoader.php line 190). Pad incomplete versions before
	// building the DSN.
	dbVer := projectConf["db/version"]
	if dbVer == "" {
		dbVer = "11.4.0"
	} else if strings.Count(dbVer, ".") < 2 {
		dbVer = dbVer + ".0"
	}

	// Pick the DSN scheme + version prefix from the user-selected
	// engine. Sylius supports both via Doctrine — MariaDB/MySQL is
	// the upstream default, PostgreSQL works once the DSN is right.
	var dbURL string
	switch strings.ToLower(projectConf["db/type"]) {
	case "postgresql", "postgres":
		dbURL = "postgresql://" + dbUser + ":" + dbPassword + "@db:5432/" + dbName + "?serverVersion=" + dbVer
	case "mysql":
		dbURL = "mysql://" + dbUser + ":" + dbPassword + "@db:3306/" + dbName + "?serverVersion=" + dbVer + "&charset=utf8mb4"
	default:
		// MariaDB (default for Sylius)
		dbURL = "mysql://" + dbUser + ":" + dbPassword + "@db:3306/" + dbName + "?serverVersion=mariadb-" + dbVer + "&charset=utf8mb4"
	}
	// Mailpit runs as a shared aruntime container on the host (not on
	// the per-project docker network), bound to port 1025. The PHP
	// container reaches it through host.docker.internal which compose
	// already pins via extra_hosts.
	mailerDSN := "smtp://host.docker.internal:1025"

	// Patch .env (or .env.local — Symfony's standard layering) so
	// Doctrine + Messenger + Mailer point at the docker hostnames.
	// Idempotent: we always (re)write .env.local with our values,
	// leaving .env untouched as the upstream baseline.
	envBody := "APP_ENV=dev\n" +
		"APP_DEBUG=1\n" +
		"APP_SECRET=changeme-madock-dev-secret\n" +
		"DATABASE_URL=\"" + dbURL + "\"\n" +
		"MAILER_DSN=\"" + mailerDSN + "\"\n" +
		"MAILER_URL=\"" + mailerDSN + "\"\n" +
		"MESSENGER_TRANSPORT_DSN=\"doctrine://default?auto_setup=0\"\n" +
		"SYLIUS_STORE_URL=\"https://" + host + "\"\n"
	envWrite := "printf '%s' " + shellSingleQuote(envBody) + " > .env.local"

	// Sylius bootstrap:
	//   composer install                       - PHP deps
	//   doctrine:database:create + migrate     - schema
	//   sylius:install --no-interaction        - admin user + base config
	//   sylius:fixtures:load default           - channels, taxa, products,
	//                                            promotions. Always runs;
	//                                            without it the storefront
	//                                            500s with "Channel could
	//                                            not be found!"
	//   doctrine:query:sql UPDATE channel host - point the seeded channels
	//                                            at the project's nginx
	//                                            host (Sylius resolves
	//                                            channels by hostname; the
	//                                            default fixtures use
	//                                            "localhost" / wildcards
	//                                            that don't match *.test)
	//   yarn install + yarn build              - Webpack Encore frontend
	//                                            assets — without them
	//                                            /admin/login throws
	//                                            "entrypoints.json not
	//                                            found"
	sqlEsc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	pinChannel := "php bin/console doctrine:query:sql \"UPDATE sylius_channel SET hostname='" + sqlEsc(host) + "'\""

	// Sylius-Standard ships exactly one fixture suite — `default` —
	// which seeds channels, products, customers, orders, promotions.
	// `sylius:install --no-interaction` falls back to it anyway when
	// no `--fixture-suite` is passed. The `--sample-data` flag is
	// honored for API consistency with other PHP platforms but maps
	// to the same suite either way (upstream Sylius doesn't expose a
	// bare-bones alternative — projects that want a slim demo
	// register their own suite in `config/packages/sylius_fixtures.yaml`
	// and override via `sylius/install/fixture_suite` in config.xml).
	fixtureSuite := configs.GetCurrentProjectConfig()["sylius/install/fixture_suite"]
	if fixtureSuite == "" {
		fixtureSuite = "default"
	}
	_ = isSampleData

	// Admin credentials come from the shared magento/admin_* config
	// (same defaults as Magento/Shopware/PrestaShop). Sylius's
	// fixture suite seeds a `sylius`/`sylius` admin row; we rewrite
	// its username/email/password/first_name/last_name to match the
	// project config so users have one set of creds across platforms.
	//
	// Hash via Symfony's `security:hash-password` helper (Argon2id),
	// then UPDATE the seeded row. WHERE clause covers both the
	// fixture-named admin (username='sylius') and a generic "lone
	// admin" fallback when a non-default suite renamed it.
	//
	// The `minimum` fixture suite does NOT seed an admin row, so this
	// UPDATE is a no-op there — users on that path keep the empty
	// state and create their admin manually via
	// `madock sylius sylius:admin-user:create`.
	adminUser := projectConf["magento/admin_user"]
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := projectConf["magento/admin_password"]
	if adminPass == "" {
		adminPass = "admin123"
	}
	adminEmail := projectConf["magento/admin_email"]
	if adminEmail == "" {
		adminEmail = "admin@admin.com"
	}
	adminFN := projectConf["magento/admin_first_name"]
	if adminFN == "" {
		adminFN = "admin"
	}
	adminLN := projectConf["magento/admin_last_name"]
	if adminLN == "" {
		adminLN = "admin"
	}
	adminPatch := "HASH=$(php bin/console security:hash-password " +
		shellSingleQuote(adminPass) +
		` 'Sylius\Component\Core\Model\AdminUser' --no-ansi 2>/dev/null | grep -oE '[$]argon2[^ ]+' | head -1) && ` +
		`php bin/console doctrine:query:sql "UPDATE sylius_admin_user SET username='` + sqlEsc(adminUser) +
		`', email='` + sqlEsc(adminEmail) +
		`', password='${HASH}', first_name='` + sqlEsc(adminFN) +
		`', last_name='` + sqlEsc(adminLN) +
		`' WHERE username='sylius' OR id=(SELECT MIN(id) FROM (SELECT id FROM sylius_admin_user) t)"`

	// Idempotency: the .madock-installed marker prevents
	// `sylius:install` + `sylius:fixtures:load` from re-running on
	// subsequent `madock install` invocations. Both commands create
	// new rows (channels, taxa, products, orders, admin) without
	// checking for existing data — re-running them duplicates the
	// catalog and breaks the storefront. Everything else (composer,
	// migrations, channel pin, admin patch, yarn, cache:warmup) is
	// already idempotent so it runs every time and stays in sync
	// with the latest config.
	//
	// Delete the marker to force a re-install: `rm -f .madock-installed`
	// inside the project directory.
	// `sylius:install` is a wizard that, in non-interactive mode
	// without --fixture-suite, defaults to the `default` suite (87
	// products + sample orders). Pass the selected suite explicitly
	// so the --sample-data flag actually toggles the data set.
	// `sylius:install` calls fixtures internally, so we don't run
	// `sylius:fixtures:load` separately.
	initialSetup := "if [ ! -f .madock-installed ]; then" +
		" php bin/console sylius:install --no-interaction --fixture-suite=" + fixtureSuite +
		" && touch .madock-installed;" +
		" fi"

	installCommand := envWrite +
		" && composer install --no-interaction --prefer-dist" +
		" && php bin/console doctrine:database:create --if-not-exists --no-interaction" +
		" && php bin/console doctrine:migrations:migrate --no-interaction" +
		" && " + initialSetup +
		" && " + pinChannel +
		" && " + adminPatch +
		" && (command -v yarn >/dev/null 2>&1 && yarn install || true)" +
		" && (command -v yarn >/dev/null 2>&1 && yarn build || true)" +
		" && php bin/console assets:install --symlink --no-interaction" +
		" && php bin/console cache:clear --no-warmup" +
		" && php bin/console cache:warmup"

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	fmt.Println(installCommand)
	phpContainer := docker.GetContainerName(projectConf, projectName, "php")
	err := docker.ContainerExec(phpContainer, "www-data", true, "bash", "-c", "cd "+workdir+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Sylius installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Sylius Storefront URL: https://" + host)
	fmtc.SuccessLn("[SUCCESS]: Sylius Admin URI: /admin")
	fmtc.SuccessLn("[SUCCESS]: Sylius Admin User: " + adminUser)
	fmtc.SuccessLn("[SUCCESS]: Sylius Admin Password: " + adminPass)
	fmtc.SuccessLn("[SUCCESS]: Sylius Admin Email: " + adminEmail)
}
