package install

import (
	"fmt"
	"net/url"
	"strings"

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

	installCommand := envWrite +
		" && yarn install" +
		" && " + patchConfig +
		" && npx medusa db:migrate" +
		" && npx medusa user --email admin@example.com --password admin"

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	fmt.Println(installCommand)
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "nodejs"), "node", true, "bash", "-c", "cd "+workdir+" && "+installCommand)
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Medusa installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Medusa Storefront URL: https://" + host)
	fmtc.SuccessLn("[SUCCESS]: Medusa Admin URI: /app")
	fmtc.SuccessLn("[SUCCESS]: Medusa Admin User: admin@example.com")
	fmtc.SuccessLn("[SUCCESS]: Medusa Admin Password: admin")
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
