**v3.7.14**

Fixed:
- Add PHP 8.5 to setup wizard PHP version picker (was missing despite Magento 2.4.9 defaulting to it)

**v3.7.13**

Changed:
- Setup wizard: platform picker puts Custom first, drops recommended marker — madock is multi-platform, no single choice should be highlighted
- Setup version pickers: refresh all language/runtime/service choices after platform detection
- Interactive selector: clamp box to terminal width, truncate long options so TUI doesn't wrap on narrow terminals

Fixed:
- ProcessSnippets: support nested includes, fix cron snippet
- BigCommerce Catalyst: bump default Node to 24.10.0
- BigCommerce stencil install: run global stencil-cli install as root
- Shopify laravel-shopify Download: pass --no-scripts to composer create-project
- Setup Download: run all scaffolding inside project containers (fixes code-not-mounted race)
- Shopware: init-chown, permissive umask, scheduled-task cron, messenger sidecar

**v3.7.12**

Added:
- BigCommerce platform support with 4 SDK/framework presets:
  - `--preset catalyst` (Node 22 + Catalyst monorepo) — official headless Next.js storefront. Pnpm + turbo monorepo. Install pre-installs pnpm globally in the nodejs image, clones `bigcommerce/catalyst`, runs `pnpm install` across workspaces, rewrites root `scripts.dev` to filter to `./core` with `-H 0.0.0.0` forwarded via `--` (turbo rejects bare `-H`), and parks `scripts.dev` as `scripts.dev:catalyst` so the container stays up until the user adds real store credentials to `core/.env.local` and runs `npm run dev:catalyst` (Catalyst's pre-dev `generate` step needs the store hash to fetch the GraphQL schema)
  - `--preset stencil` (Node 22 + Stencil CLI) — Cornerstone-based theme dev. Clones `bigcommerce/cornerstone`, runs `npm install`, installs `@bigcommerce/stencil-cli` globally, parks `scripts.dev` (Stencil needs interactive `stencil init` + API token entry)
  - `--preset api-php` (PHP 8.3 + MariaDB + Redis) — `bigcommerce/api` Composer SDK for backend integrations. `composer init` scaffolds pinned to `^3.3`, `composer install` (or update when no lock yet)
  - `--preset app-node` (Node 22) — `bigcommerce/sample-app-nodejs` (Express + Next.js with OAuth handshake) clone. Parks `scripts.dev` as `scripts.dev:bc` (app dev needs interactive Developer auth + ngrok tunnel)
- `madock bigcommerce <cmd>` (alias `madock bc`) — preset-aware container exec. Routes to nodejs container for catalyst/stencil/app-node, php container for api-php
- BigCommerce env writer mirrors Shopify's preset-branching: Node-only presets drop PHP/DB/Redis; PHP preset keeps full stack. Default DB name `bigcommerce` for the PHP preset, redis on by default with project-level override honored
- Auto-detection: composer.json with `bigcommerce/api` or legacy `bigcommerce/bigcommerce-api-php` → api-php preset. package.json with `@bigcommerce/catalyst-core` / `-client` / `checkout-sdk` → catalyst, with `@bigcommerce/stencil-cli` or `name=cornerstone` → stencil
- `docker/bigcommerce/nodejs/Dockerfile` pre-installs pnpm globally so the entrypoint's `pnpm dev` works without the `corepack enable` root-permission dance
- `docker/bigcommerce/nginx/conf/default.conf` swaps between FastCGI (PHP backend) and a Node-only proxy block based on `php/enabled` / `nodejs/enabled`. Node branch uses an in-block `map $http_upgrade $node_connection_upgrade` for Connection-header handling (matches the Shopify Node-preset pattern). For catalyst / stencil / app-node `main_service_port=3000`
- `MakeConfBigcommerce` materialises only the Dockerfiles the selected preset uses — Node-only presets skip PHP/DB/Redis Dockerfiles entirely. Same pattern as `MakeConfShopify`

Docs:
- `docs/bigcommerce.md` — preset matrix, install pipeline per preset, services-per-preset table, switching presets, gotchas (Missing store hash, turbo -H bug, stencil auth, app-node CLIENT_ID)
- `README.md` — BigCommerce added to supported platforms list and key features

**v3.7.11**

Shopify presets — post-tracer hardening:
- Hydrogen now renders `/` end-to-end. Two issues that fought the
  initial tracer:
  - Hydrogen's Oxygen plugin ignores user-set `server.allowedHosts`
    in vite.config.ts. Install now also patches Vite's internal
    `isHostAllowedInternal` in `node_modules/vite/dist/node/chunks/
    node.js` to short-circuit to `true`. Marker-gated so re-running
    install is idempotent. Pure dev convenience — patches
    node_modules only
  - Hydrogen's Miniflare/undici client rejects `Connection: upgrade`
    on non-WS requests ("invalid connection header"). Project nginx
    for Node-only presets now uses an in-block `map $http_upgrade
    $node_connection_upgrade { default upgrade; '' ''; }` so the
    Connection header is empty for plain HTTP and only forwarded as
    `upgrade` for genuine WS handshakes
- app-remix template clone switched from `npm init @shopify/app@latest`
  to `git clone shopify-app-template-remix` — the npm init argument
  parser changed across CLI versions in 2024 and was producing an
  empty directory. Install also parks the template's `dev` script
  (which is `shopify app dev`, an interactive Partner-CLI command)
  as `dev:shopify` and replaces `dev` with `sleep infinity` so the
  container stays up after install; users start the real dev server
  via `madock bash` + `npm run dev:shopify` (needs interactive
  Shopify Partner auth + tunnel — can't run from a non-tty container)
- laravel-shopify install now correctly rewrites Laravel 11+
  `.env` files where the DB_* lines ship commented out by default
  (Laravel switched to SQLite default in 2024). Sed patches handle
  both `^DB_*=` and `^# *DB_*=` forms
- api-php composer require pinned to `^6.0` (v7 isn't published yet
  on Packagist). Install also picks `composer install` vs `composer
  update` based on whether composer.lock exists — fresh `composer
  init` projects only have composer.json so update is correct
- `docker/shopify/php/Dockerfile` no longer hand-rolls the yarn
  install via GPG keyserver (which was failing with `gpg: keyserver
  receive failed` on every fresh build). Uses the shared
  `snippets/dockerfile/php/nodejs` snippet that installs Node + Yarn
  via npm when `php/nodejs/enabled` / `php/yarn/enabled` are set

Added:
- Shopify platform now ships with 4 SDK/framework presets so users can pick a stack at setup time instead of inheriting the legacy PHP-only default:
- Shopify platform now ships with 4 SDK/framework presets so users can pick a stack at setup time instead of inheriting the legacy PHP-only default:
  - `--preset hydrogen` (Node 22 + Remix on Vite, TypeScript) — official headless storefront, deploys to Shopify Oxygen
  - `--preset app-remix` (Node 22 + Remix + Prisma/SQLite) — official embedded Shopify App template
  - `--preset api-php` (PHP 8.3 + MariaDB + Redis) — raw `shopify/shopify-api` Composer SDK for backend integrations
  - `--preset laravel-shopify` (PHP 8.3 + Laravel + Node + MariaDB + Redis) — full Shopify App on Laravel via `Kyon147/laravel-shopify`
  Interactive preset wizard mirrors the Medusa/Saleor/Spree/Sylius flow. Aliases honored (`node` → hydrogen, `app`/`remix` → app-remix, `php`/`api` → api-php, `laravel` → laravel-shopify)
- Shopify env writer rewires the container stack per preset. Node-only presets (hydrogen, app-remix) drop PHP/MariaDB/Redis entirely — no zombie containers and no `FROM mariadb:{{{db/version}}}` build errors when the DB block is skipped. PHP presets keep the legacy full stack
- Shopify install handler dispatches per preset:
  - Hydrogen: `npm install`, patches `package.json` (adds `--host` to the `dev` script so Vite binds 0.0.0.0 instead of 127.0.0.1), patches `vite.config.ts` (adds `server.allowedHosts: true` so the project's `*.test` host doesn't trip Vite's host-header guard), then restarts the nodejs container
  - app-remix: `npm install` + `npx prisma generate && npx prisma migrate deploy` (Prisma uses SQLite by default — no DB container needed)
  - api-php: `composer install` (or `composer update` when no lock present) against a `composer init`-generated project pinned to `shopify/shopify-api:^6.0`
  - laravel-shopify: rewrites Laravel `.env` (APP_URL, DB_CONNECTION=mysql, DB_HOST=db, DB credentials from project config), `composer install`, `composer require kyon147/laravel-shopify`, `php artisan key:generate`, `migrate`, `vendor:publish --tag=shopify-config --tag=shopify-routes`
- Per-preset `DownloadShopify` scaffolders:
  - hydrogen: `npm create -y @shopify/hydrogen@latest -- --path . --quickstart --language ts --no-install-deps`
  - app-remix: `git clone --depth 1 https://github.com/Shopify/shopify-app-template-remix.git .` (the npm init argument parser changed across CLI versions in 2024 and was producing an empty directory; cloning the upstream template is the same content without the wizard step)
  - api-php: `composer init --no-interaction --require=shopify/shopify-api:^6.0`
  - laravel-shopify: `composer create-project --no-install laravel/laravel .`

Changed:
- `docker/shopify/docker-compose.yml` wraps the DB/Redis/RabbitMQ/Grafana service block in `<<<if{{{php/enabled}}}>>>` so Node-only presets don't try to build a DB image with un-substituted `{{{db/version}}}` templates
- `docker/shopify/nginx/conf/default.conf` swaps between FastCGI (PHP backend) and a Node-only proxy server block based on `php/enabled` / `nodejs/enabled`. The Node block declares an in-block `map $http_upgrade $node_connection_upgrade { default upgrade; '' ''; }` so the `Connection` header is empty on plain HTTP (Hydrogen's Miniflare/undici rejects `Connection: upgrade` on non-WS requests with "invalid connection header") and only `upgrade` for genuine WS handshakes. For hydrogen / app-remix the env writer pins `main_service_port=3000` to match the dev server upstream
- `MakeConfShopify` only materialises the Dockerfiles the selected preset actually uses (PHP, NodeJS, DB, Redis are now conditional), so Node-only presets don't ship a half-substituted db.Dockerfile that breaks `docker compose build`
- Added `nodejs.yml` snippet include to `docker/shopify/docker-compose.yml` + `docker/shopify/nodejs/Dockerfile` so the Node service has a real Dockerfile to build from

Docs:
- `docs/shopify.md` rewritten: preset matrix, install pipeline per preset, per-preset services table, switching presets, gotchas (Hydrogen Vite allowedHosts, Remix Partner auth, Laravel routes 404)

**v3.7.10**

Added:
- Sylius platform support: `madock setup --platform sylius` (PHP 8.3 / Symfony + MariaDB + Redis + Node + Yarn baked into the PHP image for Webpack Encore). `madock sylius <cmd>` runs `php bin/console <cmd>` inside the PHP container. `madock install` writes `.env.local` (DATABASE_URL with `serverVersion=mariadb-<major.minor.patch>` so Doctrine 3 doesn't reject the lockfile, MAILER_DSN, MESSENGER_TRANSPORT_DSN, SYLIUS_STORE_URL), runs `composer install`, `doctrine:database:create`, `doctrine:migrations:migrate`, `sylius:install --no-interaction`, `sylius:fixtures:load default` (channels, taxa, products, promotions, demo customers/orders/payments — always runs because the storefront 500s with "Channel could not be found!" without it), updates `sylius_channel.hostname` to the project's nginx host (Sylius resolves channels by hostname; the default fixtures use `localhost`/wildcards that don't match `*.test`), then `yarn install` + `yarn build` for the admin/shop/app Encore bundles, plus `assets:install` and cache warmup. Auto-detection via `composer.json` / `composer.lock` declaring `sylius/sylius` or `sylius/sylius-standard`. See [docs/sylius.md](docs/sylius.md)
- Sylius presets: `--preset 2` (Latest, Sylius 2.0.x / PHP 8.3 / MariaDB 11.4 / Redis 7.4 / Node 22), `--preset 1` (Stable, Sylius 1.13.x / PHP 8.2 / MariaDB 10.11 / Redis 7.2 / Node 20). Interactive preset wizard mirrors the Medusa/Saleor/Spree flow

Sylius — post-tracer hardening:
- `--sample-data` flag now toggles the fixture suite (`default` with the flag, `minimum` without). The previous build always loaded `default`, which seeded ~87 products + sample orders even when the user just wanted a bare storefront
- Admin credentials no longer hardcoded to `sylius`/`sylius` — read from the central `magento/admin_*` config (same defaults as Magento/Shopware/PrestaShop). Install hashes the password via Symfony's `security:hash-password` (Argon2id) and updates the seeded admin row in `sylius_admin_user`. Single source of admin truth across platforms
- `madock install` is now idempotent. A `.madock-installed` marker file in the project root suppresses the first-run-only `sylius:install` + `sylius:fixtures:load` steps on subsequent runs (those commands create new rows every time without checking for existing data — re-running them duplicates the catalog). Everything else (composer, migrations, channel hostname pin, admin patch, yarn, cache warmup) still runs every time so it stays in sync with the latest config. Delete the marker to force a full re-install
- `service:enable messenger` — optional Symfony Messenger consumer container (reuses the PHP image). Auto-consumes the well-known Sylius 2 transports (`main`, `payment_request`, `catalog_promotion_removal`). Override with the `SYLIUS_MESSENGER_TRANSPORTS` env var on the service
- `service:enable encore` — optional Webpack Encore watcher container running `yarn watch` against the project src. Admin/shop/app bundles rebuild on save. `WATCHPACK_POLLING=true` keeps it responsive on macOS bind mounts
- `MAILER_DSN` now points at `smtp://host.docker.internal:1025` instead of `smtp://mailpit:1025`. Mailpit runs as a shared `aruntime-mailcatcher-1` container on the host (not on per-project networks), so the in-network hostname was unreachable from the PHP container
- PostgreSQL DSN support — install handler picks the DSN scheme + serverVersion format from `db/type` config. MariaDB stays the default (`mysql://...?serverVersion=mariadb-X.Y.Z`); PostgreSQL emits `postgresql://...?serverVersion=X.Y.Z`; plain MySQL emits `mysql://...?serverVersion=X.Y.Z` without the mariadb prefix
- Elasticsearch / OpenSearch wiring honors `search/elasticsearch/enabled` / `search/opensearch/enabled` config instead of hardcoding `false`. Project-level Dockerfile generator now materialises both engine images. Enable with `madock service:enable elasticsearch` / `opensearch`
- API Platform endpoints verified: `/api/v2/shop/products`, `/api/v2/shop/channels`, `/api/v2/shop/taxons` return 200 + JSON-LD out of the box (no manual config). `/api/v2/admin/*` returns 401 without OAuth — same as upstream

Changed:
- `php/nodejs` Dockerfile snippet now installs Yarn as well when `php/yarn/enabled=true`. PHP-based platforms with Webpack/Encore pipelines (Sylius today; Shopware/PrestaShop tomorrow if they opt in) get yarn in the same image as composer instead of needing a separate container
- `service` registry expanded with `spree/sidekiq`, `spree/storefront`, `sylius/messenger`, `sylius/encore` mappings so `madock service:enable <short>` resolves correctly

**v3.7.9**

Added:
- Spree Commerce platform support: `madock setup --platform spree` (Ruby on Rails admin + auto-provisioned Next.js storefront + PostgreSQL + Redis). `madock spree <cmd>` to run `bundle exec rails <cmd>` inside the ruby container. `madock install` writes `.env` (DATABASE_URL, REDIS_URL, RAILS_ENV, SECRET_KEY_BASE, BINDING, PORT, admin credentials), pins `.ruby-version` + Gemfile.lock RUBY VERSION line to the container's actual Ruby, patches `config/environments/development.rb` with `assume_ssl = true` for nginx TLS termination, `bundle install`, `rails db:prepare`, `spree:admin:tailwindcss:build`, `spree:search:reindex`, `spree_sample:load` (211 products, 20 customers, sample orders, publishable API key). Default admin: `admin@example.com` / `spree123`. Auto-detection via `Gemfile` / `Gemfile.lock` containing `spree`. See [docs/spree.md](docs/spree.md)
- Spree presets: `--preset 5` (Spree 5.x / Ruby 4.0 / PostgreSQL 16 / Redis 7.2 / Rails 8), `--preset 4` (Spree 4.10.x / Ruby 3.2 / PostgreSQL 15 / Redis 7.0 / Rails 7.1). Interactive preset wizard mirrors the Medusa/Saleor flow
- Spree storefront auto-provisioned. `madock setup -d` clones `spree/storefront` (official Next.js 16 / TypeScript storefront) into `./storefront/` alongside the backend. `madock install` extracts the publishable key from `Spree::ApiKey` via `rails runner`, writes `storefront/.env.local` (SPREE_API_URL=http://ruby:3000, SPREE_PUBLISHABLE_KEY, NEXT_PUBLIC_SITE_URL, country/locale/store_name), and runs `yarn install`. nginx splits the public host: `/admin|/admin_user|/api|/up|/rails|/assets|/webhooks|/oauth|/cable` to `ruby:3000`, everything else to `storefront:3001`. Lazy DNS via Docker's embedded resolver (127.0.0.11) so nginx survives early boot. Storefront defaults: `spree/storefront/enabled=true`, `path=storefront`, `workdir=/var/www/html/storefront`, `country=us`, `locale=en`, `version=22.20.0` (Node), `git_url=https://github.com/spree/storefront.git`. Override any of those in `config.xml`. Set `spree/storefront/enabled=false` to fall back to backend-only nginx config (storefront URL then 301-redirects to `/admin`)
- `service:enable sidekiq` — optional Sidekiq worker container (same ruby image as backend, runs `bundle exec sidekiq` against the project's Gemfile.lock, connects to Redis at `redisdb:6379/0`)
- Smart Ruby entrypoint in the spree ruby container: waits for `Gemfile`, then for `bundle check` to pass (gem deps fully installed for current Gemfile.lock), sources `.env` right before exec, cleans up stale `tmp/pids/server.pid`, prefers `bin/rails server` and falls back to `bundle exec rails server`. Idles with a clear message when project code or deps are missing
- Smart storefront entrypoint variant for the Spree Next.js storefront: same wait-for-install-marker pattern as the Medusa storefront entrypoint, sources both `.env` and `.env.local` before exec
- `DetectFromGemfile` — Gemfile / Gemfile.lock scanner that matches `gem "spree"` declarations and `    spree (X.Y.Z)` resolved lockfile entries. Wired into the same auto-detection chain as composer / package.json / pyproject

Fixed:
- Saleor python entrypoint sourced `.env` at the wrong point. The file is written by `madock install` AFTER the container starts, so sourcing at boot found nothing — DATABASE_URL / REDIS_URL / SECRET_KEY never landed in process env and uvicorn / runserver fell back to psycopg's localhost default, serving 502. Moved `set -a; . ./.env; set +a` to right before exec, after the wait-for-deps loop has proven the install completed. Same fix pattern as the Medusa nodejs entrypoint

**v3.7.8**

Added:
- Medusa storefront is now auto-provisioned. `madock setup -d` clones `medusajs/nextjs-starter-medusa` into `storefront/` alongside the backend; `madock install` writes `storefront/.env.local` (with `MEDUSA_BACKEND_URL=http://nodejs:9000`, `NEXT_PUBLIC_MEDUSA_BACKEND_URL=https://loc.<project>.com`, default region, publishable key) and runs `yarn install` inside the storefront container, then restarts it. nginx routes `/health|/app|/store|/admin|/auth` to the backend on `nodejs:9000` and everything else to `storefront:8000` (lazy DNS via Docker's embedded resolver so nginx survives early boot). Storefront defaults: `medusa/storefront/enabled=true`, `path=storefront`, `workdir=/var/www/html/storefront`, `region=gb`, `git_url=https://github.com/medusajs/nextjs-starter-medusa.git`. Override any of those in `config.xml`. Set `medusa/storefront/enabled=false` to fall back to the single-upstream backend-only nginx config
- Medusa publishable API key seeding. `madock install` polls `/health`, reuses the publishable key bound to the default sales channel (the one `db:setup` seeds), creates and binds one if none exist, and writes `NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=…` into both the backend `.env` and `storefront/.env.local`. Eliminates the manual key-creation step that Medusa v2 otherwise requires for any `/store/*` request (would 400 with "A valid publishable key is required")
- Medusa starter seed auto-runs. When `package.json` defines a `seed` script (the default in `medusa-starter-default`), `madock install` invokes `yarn seed` after `db:setup` to populate the Europe region, sales channel, shipping options, and demo products — without it the Next.js storefront middleware errors out with "No regions found"

Fixed:
- Medusa `db:setup` left migration scripts pending. `madock install` previously called `db:migrate` and Medusa boot then hit `relation tax_provider does not exist` until a separate `db:migrate:scripts` run. Switched to `npx medusa db:setup --db <name>` (umbrella command that runs migrations, migration scripts, and link sync) and added an explicit container restart after install so PID 1 doesn't race the final migration scripts
- Medusa Admin "Blocked request" / 403 (Vite 5 host gate). `madock install` patches `medusa-config.ts` to add `admin: { vite: () => ({ server: { allowedHosts: true } }) }` if not already present, so the `*.test` project host loads the admin UI without manual config edits
- Backend chokidar reload loop triggered by storefront installs. Medusa's `develop.js` hardcodes the watcher ignore list and only ignores top-level `node_modules`. With the storefront cloned into `./storefront/`, every file written by `yarn install` in the storefront triggered a backend reload. `madock install` now patches `node_modules/@medusajs/medusa/dist/commands/develop.js` to inject regex ignores for `/storefront/` and any `/node_modules/` segment. The patch is idempotent and survives until the next backend `yarn install` (re-run `madock install` to re-apply)
- `medusa/storefront/public_backend_url` derivation. The host parser stores host strings as `domain.test:code` (where `code` namespaces nginx/hosts/<code>/name). Previously the storefront's `NEXT_PUBLIC_MEDUSA_BACKEND_URL` came out as `https://medusa.test:base` because we used the raw value. Now the trailing `:code` is stripped before building the URL
- Storefront entrypoint sources `.env` and `.env.local` right before exec so the Next.js dev server sees `NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY` etc. Same wait-for-deps marker pattern as the backend entrypoint (yarn 4 install-state.gz, yarn 1 integrity, npm package-lock, pnpm modules.yaml) so the dev server starts the moment `yarn install` completes
- Backend `.env` line gluing when `medusa db:setup` rewrites the file without a trailing newline (it appends `DB_NAME=<db>`). The publishable key write now prepends `\n` so it can't end up concatenated onto the previous line

**v3.7.7**

Fixed:
- `madock setup -d -i` for Medusa and Saleor: the Node.js / Saleor python entrypoints used to `exec sleep infinity` when `node_modules` / `.venv` was missing, then `madock install` populated those folders inside the same container via `docker exec`, but PID 1 stayed asleep. The dev server never started and nginx returned 502 Bad Gateway. The entrypoint now poll-waits for deps and exec's `yarn dev` / `uvicorn` / `manage.py runserver` the moment they appear
- Medusa and Saleor `setup` controllers now honour `-d` (download) and `-i` (install) flags. Previously only Magento setup looked at them, so `madock setup -d -i -s` on a Medusa/Saleor project rebuilt containers and exited without cloning the starter or running migrations. Medusa setup clones `medusajs/medusa-starter-default`; Saleor setup clones `saleor/saleor` at the branch derived from the selected version

**v3.7.6**

Added:
- Saleor platform support: `madock setup --platform saleor` (Python 3.12 + PostgreSQL + Redis + uvicorn/runserver). `madock saleor <cmd>` to run `manage.py` inside the python container (uses `uv run` when `uv.lock` is present). `madock install` writes `.env` (SECRET_KEY, DATABASE_URL, REDIS_URL, CELERY_BROKER_URL, ALLOWED_HOSTS, PUBLIC_URL), runs `uv sync --frozen` (or `pip install` for older releases), `manage.py migrate`, and `manage.py populatedb --createsuperuser` for the default `admin@example.com` / `admin` account. Auto-detection via `pyproject.toml` / `uv.lock` / `poetry.lock` / `requirements.txt`. See [docs/saleor.md](docs/saleor.md)
- Saleor presets: `--preset latest` (Saleor 3.23 / Python 3.12 / PostgreSQL 15 / Redis 7.2), `--preset stable` (Saleor 3.20). Interactive preset wizard mirrors the Medusa flow
- `service:enable dashboard` — optional Saleor Dashboard SPA container (`ghcr.io/saleor/saleor-dashboard:3.23`), host port auto-allocated via `{{{port/saleor_dashboard}}}`, `API_URL` wired to the project nginx host
- `service:enable worker` — optional Celery worker (with beat embedded) sharing the python image, runs `celery -A saleor --app=saleor.celeryconf:app worker --loglevel=info -B`
- Smart Python entrypoint in the saleor python container: sources `.env` (Saleor reads config from `os.environ`, does NOT auto-load `.env`), detects `manage.py` + `saleor.asgi:application` and prefers `uvicorn` for ASGI, falls back to `manage.py runserver`. Idles with a clear message when dependencies are missing
- `ProxyConfTransformer` extension point (`src/helper/configs/aruntime/proxytransform/`) — lets downstream consumers post-process the fully assembled `proxy.conf` before it lands on disk. Symmetric with the existing `DockerTransformer` hook for `docker-compose.yml`. Use case: enterprise add-ons rewriting service location prefixes (e.g. suffixing `/phpmyadmin/` with a per-project hash), adding extra server blocks for cross-domain admin tools, injecting `auth_request` directives, etc. API: `proxytransform.SetProxyConfTransformer(t ProxyConfTransformer)` where `ProxyConfTransformer.TransformProxyConf(content string) string`. Default behaviour unchanged when no transformer is registered

Changed:
- `docker.Down` / `docker.Kill`: label-based fallback (`com.docker.compose.project=madock_<name>`) so containers/volumes/networks/images get cleaned even when the compose file is missing. Previously these were silent no-ops when the project state directory had already been removed
- `GetProjectName`: resolve symlinks (`filepath.EvalSymlinks`) before comparing the stored project `path` against the current working directory. On macOS `/tmp` is a symlink to `/private/tmp`, so revisiting a project through the symlinked path no longer auto-suffixes the name to `<project>-2`. Also tightens the suffix loop to actually exit on a match

**v3.7.5**

Added:
- Medusa.js platform support: `madock setup --platform medusa` (Node.js + PostgreSQL + Redis), `madock medusa <cmd>` to run the Medusa CLI inside the nodejs container, `madock install` scaffolds `.env` + runs `db:migrate` + creates an admin user. Auto-detection via `package.json` (`@medusajs/medusa` or `@medusajs/framework`). Default versions: Node 20.18, PostgreSQL 16.4, Redis 7.2, Yarn 4.5. See [docs/medusa.md](docs/medusa.md)
- Medusa setup presets: `--preset latest` (Medusa 2.x: Node 22, Postgres 17, Redis 7.4), `--preset stable` (Medusa 2.0 baseline), `--preset legacy` (Medusa 1.x: Node 18, Postgres 14, Redis 7.0). Interactive preset wizard in `madock setup --platform medusa` mirrors the Magento flow
- `service:enable meilisearch` — Meilisearch as an opt-in search engine container across all platforms (`getmeili/meilisearch:v1.11.3`, master key `masterKey`). Wired into the Medusa compose template
- `service:enable storefront` — optional Next.js storefront container for Medusa v2. Mounts the project's `<project>/storefront/` folder into `/var/www/storefront`, internal port 8000, host port auto-allocated via `{{{port/storefront}}}`. Env vars wire `MEDUSA_BACKEND_URL` to the internal `nodejs:9000`. Configurable via `medusa/storefront/*` keys in `config.xml`

Changed:
- Port allocator now also probes the host (`net.Listen`) and consults `docker inspect HostConfig.PortBindings` for every running and stopped container before handing out a port. The registry remains the primary source of truth; the extra probes defend against ports that something outside madock claimed (other docker stacks, leftover containers, non-docker listeners)
- Default DB credentials changed from `magento`/`magento`/`magento` to DDEV-style `db`/`db`/`db` (`db/root_password` stays `password`). Affects new projects only; existing projects keep their stored values
- New V375 migration backfills `db/user`/`db/password`/`db/database` = `magento` for projects whose `config.xml` relied on the previous embedded defaults, so their Docker volumes and DB users keep working after the upgrade
- Default `timezone` switched from deprecated `Europe/Kiev` to `UTC`. IANA renamed `Europe/Kiev` to `Europe/Kyiv` in tzdata 2022b; UTC is the standard server default and avoids DST surprises in logs. Existing projects keep their stored timezone
- Shared nginx `proxy.conf` no longer hardcodes upstream port `3000`. It now uses `{{{main_service_port}}}`, resolved per platform from the project config (Medusa env writer sets it to `9000`; existing custom/nodejs projects fall back to `3000`, matching the old behaviour)

**v3.7.4**

Added:
- Magento 2.4.9 support: PHP 8.5 + Xdebug 3.5.0, MariaDB 11.8, RabbitMQ 4.2, Valkey 9.0.0. OpenSearch 3.0.0 was already wired. Composer stays on the `"2"` major (ondrej apt resolves the latest 2.9.x). Project and proxy nginx bumped to 1.28
- ActiveMQ Artemis 2 as an opt-in service. Enable with `madock service:enable artemis` — wired on all platforms (magento2, shopware, prestashop, woocommerce, shopify, custom). Defaults: `apache/activemq-artemis:2.42.0`, user/password `artemis/artemis`. Not part of the `setup` wizard
- `service:enable --version <ver>` flag. For services that have a version (currently `valkey`, `artemis`, `xdebug`), enable prompts an interactive version picker (same selector as `setup`) unless `--version` is given, then persists `<service>/version` to the project config

Changed:
- `setup` wizard no longer prompts for Valkey version. The Valkey container stays opt-in via `service:enable valkey [--version <ver>]`, matching the new pattern. Existing `<valkey>` config blocks remain unchanged
- `project:clone` now requires `--domain-suffix` / `-s`. The suffix is inserted before the TLD dot of each cloned host (e.g. `shop.test` + `-update` → `shop-update.test`), so the proxy nginx no longer aborts the clone with "Duplicate domains found" right after copying the source config
- PHP 8.5 build support: `php8.5-opcache` and `php8.5-xmlrpc` (not shipped as separate packages by ondrej PPA) are now installed in their own optional `apt-get` lines that tolerate a missing package. The pecl mcrypt skip branch in the php Dockerfile snippets now covers PHP 8.4 and any newer version (`>= 8.4`) instead of being hardcoded to 8.4 only
- `setup --preset` list: new `Magento 2.4.9 (Latest)` preset (PHP 8.5, OpenSearch 3.0, MariaDB 11.8, RabbitMQ 4.2, Valkey 9.0.0). The previous "Latest" entry for 2.4.8 is relabelled to `(Previous)`
- `patch:create` now detects `cweagans/composer-patches` major version from `composer.lock` and writes the matching format: v1 keeps the existing `"vendor/pkg": { "Title": "path" }` map, v2 writes the new `"vendor/pkg": [ { "description": "Title", "url": "path" } ]` array-of-objects shape. Applies to `extra.patches` in `composer.json` and to `patches.json`

Fixed:
- Fix `host not found in upstream "php_without_xdebug:9000"` nginx error caused by the `<<<if{{{main_service_enabled}}}>>>` block in `nginx.yml` always being stripped — `main_service` and `main_service_enabled` placeholders are now substituted before `ReplaceConfigValue` runs `processConditionals`, so the conditional sees the concrete value (`true`/`false`) instead of an unresolved placeholder. Without this fix the `depends_on: php` block in nginx was always removed, letting nginx start before `php_without_xdebug` and fail upstream DNS resolution. Affects all projects on 3.7.2/3.7.3, regardless of `php/enabled` value ([#40](https://github.com/faradey/madock/issues/40))
- Unlock the ImageMagick PDF coder in php Dockerfile snippets — default Debian/Ubuntu `/etc/ImageMagick-6/policy.xml` blocks PDF reads, which breaks Imagick-based PDF preview generation in PHP apps (e.g. Magento label rendering). The `rights="none" pattern="PDF"` policy is now switched to `rights="read|write"` during image build

**v3.7.3**

Fixed:
- Fix `host not found in upstream "php_without_xdebug:9000"` nginx error after upgrading to 3.7.2 with `php/enabled=false` and `php/xdebug/enabled=true` — nginx confs in all platform templates now gate the `fastcgi_backend_xdebug_true` upstream on the same dual condition (`php/enabled` AND `php/xdebug/enabled`) used by the `php_without_xdebug` compose snippet ([#40](https://github.com/faradey/madock/issues/40))

**v3.7.2**

Fixed:
- Fix `service "nginx" depends on undefined service "php"` and `service "php_without_xdebug" depends on undefined service "php"` errors when project config lacks `php/enabled` — nginx `depends_on` is now gated by a new `main_service_enabled` placeholder, and the `php_without_xdebug` snippet now requires both `php/enabled` and `php/xdebug/enabled` ([#40](https://github.com/faradey/madock/issues/40))
- Use full `php bin/magento cache:flush` instead of the `c:f` shorthand inside `madock c:f` to avoid Symfony console ambiguity in setup-only mode

Changed:
- V366 migration now also covers `woocommerce` and `shopify` platforms
- New V372 migration backfills `php/enabled=true` for projects upgraded from versions in the 3.6.7..3.7.1 range, which the V366 trigger (`< 3.6.7`) had missed

**v3.7.1**

Fixed:
- Fix nodejs language Dockerfile build failure: `chown 501:20 /var/www` failed because `node` base image has no `/var/www` directory — `mkdir -p /var/www` added before chown
- Suppress noisy `cron: unrecognized service` stderr from cron stop probe — `service cron status` now runs silently when used as availability probe

Added:
- Cron support in nodejs language container: `apt-get install -y cron` added to nodejs Dockerfile, enabling `cron.enabled=true` and `cron/jobs/*` for nodejs-only projects

**v3.7.0**

Added:
- `madock mcp` — built-in MCP (Model Context Protocol) server for AI assistants (Claude Code, Cursor, VS Code). Provides 30 tools: container lifecycle, configuration, database operations, service management, Composer/Magento CLI, remote sync, and more. See [docs/mcp.md](docs/mcp.md)
- WooCommerce platform support: `madock setup --platform woocommerce`, WP-CLI via `madock wp`, auto-detection by `wp-config.php`
- JetBrains IDE plugin: [Madock Integration](https://plugins.jetbrains.com/plugin/31208-madock-integration)

**v3.6.9**

Added:
- `--quiet` / `-q` flag available on all commands — suppresses Docker build/pull output (useful in JediTerm and other IDEs to avoid flood output). Affects `start`, `rebuild`, `setup`, `debug:enable`, `debug:disable` and any other command that triggers Docker operations
- `db:import` now detects MySQL `GTID_PURGED cannot be changed` errors and offers an interactive resolution: run `RESET MASTER` (or `RESET BINARY LOGS AND GTIDS` on MySQL 8.4+) and retry, or retry with GTID statements stripped from the dump on the fly
- `--reset-gtid` flag for `db:import` to perform the GTID reset automatically before import (useful for CI/scripts)
- `db:import` now detects MySQL `ERROR 1062 Duplicate entry` errors and offers to retry the import in force mode (`-f`) so duplicate-row errors are skipped instead of aborting

Fixed:
- `db:import` now restores `FOREIGN_KEY_CHECKS=1` even when the import fails
- `db:import` stderr is now captured for analysis while still being streamed to the terminal

**v3.6.8**

Changed:
- Rename `DockerSecretsInjector` interface to `DockerTransformer` — more general name for the Docker file transformation hook, backward-compatible `SetSecretsInjector` wrapper retained

**v3.6.5**

Fixed:
- Fix search engine config not applied when using presets — setup controllers now pass search engine type to project config generators
- Remove trailing slash from root path variables in nginx configs
- Use host-gateway instead of outbound IP for container host resolution — removes unreliable `GetOutboundIP()` UDP dial, uses Docker's built-in `host-gateway`
- Bump version.go to 3.6.5 (was stuck at 3.6.1 since v3.6.2–v3.6.4)

**v3.6.1**

Fixed:
- Fix `MADOCK_USER` environment variable not working with `madock bash` — the bash controller now respects env overrides via `GetEnvForUserServiceWorkdir`

**v3.6.0**

Changed:
- Move database credentials from Dockerfile ENV to docker-compose environment — passwords are no longer baked into Docker image layers (visible via `docker history`), instead passed at runtime through docker-compose environment variables. Affects mysql, postgresql, mongodb, and db2 services.

**v3.5.9**

Fixed:
- Fix db/type migration not running for users upgrading from v3.4.0+ — migration was guarded by `< "3.4.0"` and version.go was not bumped, so the migration never executed for existing users
- Bump version.go to 3.5.9

**v3.5.8**

Added:
- v3.4.0 migration: adds `db/type` field to existing project configs based on `db/repository` (mysql, postgresql, mongodb)
- Sync `config_defaults.xml` with `config.xml`: add `db/type`, `db/pgadmin`, `db/mongo_express` defaults

**v3.5.7**

Fixed:
- Fix remaining `.madock/config.xml` write paths — `SetEnvForProject` (setup) and `GetCurrentProjectConfigPath` (scope:set/add) now correctly write to `projects/<projectname>/config.xml`

**v3.5.6**

Added:
- Mailpit (mailcatcher) is now a toggleable service — disabled via `madock service:disable mailpit --global`, enabled by default for backward compatibility

Changed:
- `.madock/config.xml` is now read-only for madock — all automatic config changes (`service:enable/disable`, `config:set`, `debug:enable/disable`, `cron:enable/disable`) write to `projects/<projectname>/config.xml` instead. This allows `.madock/config.xml` to be committed to the repository without unexpected modifications on servers

**v3.5.5**

Fixed:
- Fix nodejs version in PHP container ignoring project config — `customPhpConfig` used `generalConf` directly instead of `GetOption`, always defaulting to 18.x regardless of project settings

**v3.5.4**

Added:
- `llms.txt` — structured context file for AI agents (Claude Code, Cursor, Copilot) with full command reference, config format, and architecture overview

**v3.5.3**

Fixed:
- Fix XML config parser losing data when adding keys to empty scope — `<default></default>` (empty element) blocked `SetParam` from writing nested keys. Now empty leaf nodes are promoted to branch nodes when nested keys are added.

**v3.5.2**

Added:
- Embed `docker/` and `scripts/` into the binary via `go:embed` — the binary is now self-contained
- Auto-extract embedded assets to disk on first run or version change (`.embedded_version` marker)
- `src/helper/embedded` package with `ExtractIfNeeded()` for version-aware asset extraction

**v3.5.1**

Added:
- Per-option confirmation in setup reconfigure mode: when re-running `madock setup` on a project with existing config, each option shows "Current: X — Change? [y/N]" instead of re-asking everything from scratch
- `PopulateFromConfig` helper to fill ToolsVersions from existing project config
- `SetReconfigure` flag to enable/disable reconfigure mode in setup tools
- `Language()` now accepts current value parameter for correct display in reconfigure mode

Changed:
- `SelectInteractive` shows "Change?" confirmation in reconfigure mode, skipping selector if user declines
- `hostsCustom` in custom platform converted to use `SelectInteractive` for consistent reconfigure behavior
- All platform setup handlers (Magento, Custom, Shopware, Shopify, PrestaShop) call `PopulateFromConfig` before interactive questions

**v3.4.0**

Added:
- PostgreSQL support: docker-compose snippet, Dockerfile, `db:export` via `pg_dump`, `db:import` via `psql`, `db:info` with type display
- MongoDB support: docker-compose snippet, Dockerfile, `db:export` via `mongodump`, `db:import` via `mongorestore`
- Database engine selector in `madock setup`: MariaDB, MySQL, PostgreSQL, MongoDB
- `db/type` config key for explicit database type (`mysql`, `postgresql`, `mongodb`) with auto-detection from `db/repository` for backward compatibility
- Template flags `db/type_is_mysql`, `db/type_is_postgresql`, `db/type_is_mongodb` for conditional docker-compose/Dockerfile sections
- pgAdmin service (`db/pgadmin`) for PostgreSQL admin UI
- Mongo Express service (`db/mongo_express`) for MongoDB admin UI
- `remote:sync:db` support for PostgreSQL (`pg_dump`) and MongoDB (`mongodump`)
- `DbType` field in `ToolsVersions` struct
- Version selectors: MySQL (9.2, 9.1, 8.4, 8.0), PostgreSQL (17, 16, 15, 14, 13), MongoDB (8.0, 7.0, 6.0, 5.0)

Changed:
- `db:export`, `db:import`, `db:info` commands now dispatch by database type from config
- All platform env writers set `db/type` and `db/repository` based on selected engine
- `MakeDBDockerfile` skips `my.cnf` generation for non-MySQL databases
- `db.yml` docker-compose snippet wrapped in `<<<if{{{db/type_is_mysql}}}>>>` conditional

**v3.3.0**

Added:
- Exported `GetDefaultConfigXML()` in `configs` package — returns raw embedded config defaults for enterprise config layering
- Exported `version.Version` constant in `src/version/` package so downstream consumers can read the madock version without hardcoding it
- Tests for `GetOriginalGeneralConfig()` merge behavior (embedded-only, file-over-embedded, empty-value gap-fill)

Changed:
- `main.go` uses `version.Version` instead of local `appVersion` var
- `<<<else>>>` support in template engine for conditional blocks (`<<<if>>>...<<<else>>>...<<<endif>>>`)
- Centralized service credentials in `config.xml` for RabbitMQ, Grafana, Redis, Valkey, Elasticsearch, OpenSearch
- Auth config blocks (`auth/enabled`, `auth/user`, `auth/password`) for Grafana, Redis, Valkey, Elasticsearch, OpenSearch
- Secret key registration for all new service passwords
- RabbitMQ docker snippet now uses `{{{rabbitmq/user}}}` and `{{{rabbitmq/password}}}` placeholders instead of hardcoded `guest:guest`
- Grafana docker snippet uses `<<<if>>><<<else>>>` conditional for anonymous vs credential-based auth
- Grafana RabbitMQ exporter uses config placeholders for RabbitMQ credentials
- MySQL exporter config uses `{{{db/root_password}}}` placeholder instead of hardcoded password
- Migration guide for PWA Studio projects to custom+nodejs platform
- Snippet-based Dockerfiles for all languages (Python, Go, Ruby, Node.js, None) using reusable common snippets
- Common Docker snippets: `header-ubuntu`, `cron`, `mkdir`, `chown`, `cleanup`, `footer`
- `php/enabled` conditional guard for PHP services in docker-compose
- Dynamic `depends_on` in nginx with `{{{main_service}}}` placeholder
- Interactive version selectors for Python, Go, Ruby during `madock setup`
- Nginx snippet system (`php.conf`, `proxy.conf`) for language-specific configurations
- Migration v3.3.0 for automatic config key migration
- Moved PHP Dockerfile from `docker/custom/php/` to `docker/languages/php/`
- Renamed config key `php/timezone` → `timezone` across all platforms
- Split `nodejs/enabled` into standalone (language) and `php/nodejs/enabled` (embedded in PHP container)
- All languages now use unified fallback chain through `docker/languages/<language>/`
- Moved `<timezone>` from `<php>` to top level in default config.xml

Removed:
- PWA as a standalone platform (use custom+nodejs instead)

**v3.2.0**

Added:
- Multi-language support for custom platform: PHP, Node.js, Python, Go, Ruby, and language-less (`none`) projects
- Configurable cron jobs support for all platforms
- `info:ports` command to show allocated ports for the project
- File path argument support for `db:import` command
- JSON output support for CLI commands (`--output=json`)
- MySQL 8.4+ support, removed deprecated `db/type` config
- VPS installation script
- `xdg-utils` to base PHP image for all platforms
- Hot reload ports for Shopware storefront
- Port mappings for proxy services (Grafana, Kibana, OpenSearch Dashboards, phpMyAdmin, Selenium, Varnish)
- RabbitMQ monitoring dashboard to Grafana
- Deployment guide for Magento 2 and Shopware
- Documentation for Magento, PrestaShop, Shopware, custom cron jobs, JSON output

Refactored:
- Command registry pattern replacing switch statement in `main.go`
- Platform handler interface to eliminate code duplication
- Split `docker.go` into focused modules
- `SetXmlMap` refactored to use recursion instead of hardcoded switch cases
- Path builder utility to centralize path construction
- Replaced `panic(err)` with `log.Fatal` for proper error handling
- Reuse `removeCronJobsFromConfig` in install function

Fixed:
- `patch:create` command to work without TTY
- `cron:enable/disable` not saving config status
- Duplicate project entries in domain check
- Varnish proxy configuration
- Nginx proxy configuration issues
- Network configuration for dashboard services
- Grafana configuration for dashboards and networking

Improved:
- Instant key response for setup confirmation prompt
- Increased proxy rate limit defaults
- Verbose CLI output for Shopify cron enable/disable
- Auto-detect artisan location for Shopify cron setup
- Duplicate domain error messages now show all affected projects

**v3.1.0**

Added:
- Improved documentation for media sync, cron, snapshots, isolation, environment variables, and configuration
- Interactive setup wizard with ASCII banner, progress indicators, arrow keys navigation, styled selectors, configuration summary, inline validation, help hints, and confirmation prompts
- `proxy:reload` command for graceful nginx configuration reload without downtime
- `--yes` flag to setup command for auto-confirmation (skip prompts in CI/CD)
- `--preset` flag for quick setup with preset configurations (e.g., `magento-248`, `magento-247`)
- Auto-detection of Magento version from composer.json
- Progress indicator for database import
- On-demand port allocation system for better resource management
- Configurable proxy settings
- Timestamp to debug.log entries
- Magento 2.4.9 support with OpenSearch 3.0.0
- `shopware:bin` command for Shopware CLI operations
- Unit tests for core packages
- RabbitMQ monitoring dashboard in Grafana (queues, connections, channels, message rates)
- RabbitMQ exporter for Prometheus metrics collection
- Port mappings for Grafana, Kibana, OpenSearch Dashboards, phpMyAdmin, Selenium, Varnish

Improved:
- Nginx proxy security and performance
- Updated nginx from 1.21.4 to 1.26
- Grafana stack configuration with proper datasource UIDs for dashboard compatibility

Fixed:
- Section padding panic in setup wizard
- Non-deterministic XML config output order
- Nested conditional processing in config templates
- MariaDB exec file compatibility
- Composer install command for Shopify platform
- Various potential bugs across the codebase
- Nginx http2 directive deprecation warning (nginx 1.25+)
- Duplicate upstream and global directive errors in nginx proxy
- Varnish network connectivity with backend nginx
- Grafana subpath proxy configuration

**v3.0.0**
- Introduced a generic diff command: `madock diff --platform <code> --old <ver> --new <ver> [--path <publicDirFromSiteRoot>]`
- Added store scopes documentation split into a dedicated file `docs/store_scopes.md` and linked from README
- Added Valkey key-value DB
- Minor fixes and refactors in diff scripts (path handling and directory creation)

**v2.9.1**
- Added Magento 2.4.8 support
- Fixed the restart policy for aruntime containers

**v2.9.0**
- Added the env variable MADOCK_TTY_ENABLED (0/1). MADOCK_TTY_ENABLED is enabled by default
- Fixed SSH volume
- Fixed "install" command for prestashop platform
- Fixed docs
- Added logo
- Fixed GetRunDirPath function for outside executors
- Added php8.4 support
- Fixed incorrect version comparison for MariaDB
- Fixed arguments for the Setup command
- Fixed Magento2 install subcommands
- Fixed livereload
- Fixed apt-get to apt and added --allow-releaseinfo-change
- Added php-redis library to php installation
- Fixed RabbitMQ recommended version for Magento 2.4.7-p5 and later
- Added the restart policy

**v2.8.0**
- Added **PrestaShop** as a separate service
- Fixed "composer" command for Shopify service
- Improved custom commands and documentation

**v2.7.0**
- Fixed the creation of patches
- Fixed the cron for Shopify platform
- Fixed TODO comments
- Fixed NodeJs major version for php.Docker file
- Added http2 in the nginx configuration

**v2.6.0**
- Added Grafana as a service
- Added Grafana dashboards for Loki, Mysql and Redis
- Support for snippets in configuration files has been added. This has allowed us to eliminate repetitive code and settings.
- Added the new option `--shell` for `madock bash` command. It can be used `bash` or `sh` as a shell.

**v2.5.0**
- Added supporting of Shopware
- Fixed mailcatcher configuration with MP_SMTP_AUTH_ACCEPT_ANY and MP_SMTP_AUTH_ALLOW_INSECURE
- Fixed documentation
- Fixed the media synchronization public path
- Added --db-host, --db-port, --db-name, --db-user, --db-password as options for the remote:sync:db command

**v2.4.4**
- Fixed opensearch-dashboards
- Added new command `madock project:clone` [more](docs/project_clone.md)
- Added php/nodejs service to the php container
- Fixed documentation
- Fixed bug with the `madock cli` command
- Added custom commands [more](docs/custom_commands.md)

**v2.4.3**
- Added interactive options for the `madock setup` command
- Added an isolation mode [more](docs/isolation.md)
- Added Varnish cache [more](docs/varnish.md)
- Refactoring code


**v2.4.2**
- Support Magento 2.4.7 and Adobe Commerce 2.4.7
- Updated docker-compose version to 3.8
- Fixed DB host for the service db2
- Fixed GetActiveProjects method
- Fixed start/stop project
- Fixed db:export
- Fixed node grunt exec:<theme>
- Fixed documentation
- Added "RUN npm install -g grunt-cli" to docker file
- Fixed bug with "cache" folder
- Fixed if/else in config files
- Fixed project configuration
- Fixed Snapshot container
- Added snapshots functionality for the project
- Fixed .madock/config.xml
- Update PHP mcrypt version
- Fixed OpenSearch env variables



**v2.4.1**
- Added command scope:add to add a new scope and activate it
- Added the ability to store the madock configuration within a project in the .madock folder. To do this, you need to manually create a .madock folder and transfer configuration files and database backups to it, if necessary
- Added full support for creating patches for cweagans/composer-patches
- Added full support for creating patches for vaimo/composer-patches
- Added logger with stack trace
- Fixed the config cache
- Fixed the bug with the enable/disable services
- Fixed compatible version magerun n98 and PHP
- Fixed Adobe Cloud commands
- Fixed project path
- Fixed db:import
- Fixed bug with config.xml and the setup of a new project
- Fixed missing dir aruntime/projects
- Fixed working commands Start, Stop, Restart without internet
- Fixed madock info
- Fixed xdebug profile for PHP 7.1 or less


**v2.4.0**
- Added the new option PUBLIC_DIR in the project configuration. Each platform can have a different path of public folder therefore this option will be specified as a public folder in the container.
- Fixed host for phpmyadmin2
- Fixed mcrypt extension for PHP
- Fixed mail for CLI
- Improve command "madock c:f"
- Added --force option for the command "madock rebuild". Removes running containers without waiting for them to complete correctly and creates new containers.
- Added new library for CLI commands
- Replaced Mailhog to Mailpit
- The configuration file format would be changed from .txt to .xml. The project configuration file env.txt has been renamed to config.xml. The old configuration files have been preserved so that if you have problems with the new version of Madock, you can roll back to the old version.
- Configuration scopes for the project have been added. Now switching between configurations has become convenient and there is no need to create a copy of the project in a neighboring folder. The database is also separate for each scope.
- Added the new command "madock scope:list" for listing all scopes of the project.
- Added the new command "madock scope:set" for switching between scopes of the project.
- The commands "remote:sync:media", "remote:sync:db" and "remote:sync:file" have received an additional option "--ssh-type" which specifies the prefix of the name of the ssh settings in the project configuration. This way you can specify which ssh settings to use when executing the command.
- Added aruntime configuration caching. Now Madock will parse files less when starting and rebuilding a project.
- Added the new command "madock config:cache:clean" for cleaning Madock aruntime cache.
- Added the new command "madock open" for opening the project in the browser.
- Improve documentation of Madock

**2.2.0**
- Shopify support
- Custom PHP project support
- Relocated setup option "Specify Magento version" to top
- Added CONTAINER_NAME_PREFIX option in config. This option will allow you to run a madock project independently of other docker builds in the space with the default madock_ prefix. For already configured projects, the space will have an empty prefix to prevent projects from breaking.
- Added --ignore-table for "db:export" and "remote:sync:db" commands. Ignore the table when exporting. The specified table will not be included in the backup file. To specify multiple tables, specify this option multiple times.
- Updated OS Ubuntu for containers from 20.04 to 22.04. This will only affect those projects that will be installed after updating this build.
- Improve documentation for new commands
- Fixed some problems with NodeJs
- Fixed issue #9

Thanks @artmouse @serhii-chernenko

**2.1.0**
- Support the Magento Functional Testing Framework (MFTF)
- Fixed multiline commands

**2.0.1**
- Fixed the setup with Hosts
- Fixed the setup with the version Redis and rabbitMQ
- Fixed "madock status" command
- Fixed the DB host description

**2.0.0**
- PWA Studio as a separate service.
- Backward incompatible changes were made to the code. Code changes allow new platforms to be added in the future.
- At the moment, PWA Studio has been added as a separate service.
- There are plans to add Shopify and Shopware in the future.

**1.9.1**
- Fixed command project:remove
- Removed "restart: on-failure:3" from Elasticsearch service of docker-compose
- Installed libssh2-1-dev libssh2-1 php-ssh2 for PHP
- Removed the restart_if_failure option for the DB service of docker-compose
- Improved removing project. Now deletion is more transparent. Before execution, you will see the items that will be deleted and only after your confirmation will they be deleted.
- Fixed files permission with --with-chmod

**1.9.0**
- Added
  - Support Magento 2.4.6
  - Support sample data with the setup command
  - OpenSearch
  - Support PHP 8.2 and xdebug
  - Improved patcher for creating patches from the whole folder
  - Updated phpmyadmin version from 5.2.0 to 5.2.1
  - Increased UPLOAD_LIMIT for phpmyadmin. Now it is 2GB
  - Custom DB repository in the config
  - PHP 8.2 to the setup process
  - Xdebug profile
  - Increased PHP Max Input Vars Limit by default
  - Enabled log_bin_trust_function_creators for DB
  - New option for DB commands "--service-name DB container name. Optional. Default container: db. Example: db2"
  - Support overriding /docker/nginx/conf/default-proxy.conf
  - Command "install"
  - Support n98-magerun
  - Support the second DB
  - Support proxy as a service

- Fixed
  - Default_server for the nginx proxy configuration
  - Remove --single-transaction option from the mysqldump command
  - Remove the innodb_log_file_size option for MySQL 8.x
  - Improved elasticsearch plugins installation
  - Cron
  - Bug with the start/stop command of the proxy server
  - FOREIGN_KEY_CHECKS for the import DB
  - Project setup with Redis and rabbitMQ versions
  - Bug with the media synchronization
  - Proxy port and the starting script
  - Livereload location in nginx proxy
  - DEFINER for the DB import/export
  - Issue with permissions of .ssh folder #8

**1.8.2**
- Fixed generation env.php file with rabbitmq password

**1.8.1**
- Fixed starting the Nginx proxy containers

**1.8.0**
- Added a new command "patch:create"
- Added a new param "--name" for "db:export" and "remote:sync:db" 
- Added a new command setup:env for generating env.php file
- Changed domain .loc to .test by default
- Optimization for MariaDB 10.4
- Prune the volumes with option --with-volumes. For example Madock prune --with-volumes
- Added the ability to specify a custom repository and version of docker images when you set up the project
- Added "--with-chown" option for some commands. Reset permissions for files and folders
- Improved "db:import" command. Now, the Madock can read DB files from any folder of the Magento project. The name of the DB file must contain ".sql" in any part of the name
- Fixed the problem with the same project folder names from different locations
- Added a new command project:remove
- Added stopping proxy containers if there are no active projects
- Refactoring code

**1.7.4**
- Additional changing external IPs for containers from 0.0.0.0 to 127.0.0.1

**1.7.3**
- Changed external IPs for containers from 0.0.0.0 to 127.0.0.1
- Fixed bug with CLI options and arguments

**1.7.2**
- Fixed bug with the docker compose

**1.7.1**
- The internal command "docker-compose" was replaced by "docker compose"

**1.7.0**
- All commands are brought to uniformity. Now they match the Magento approach
- Added the support of Magento cloud
- Added the support of automatically creating composer patches
- Added the new command "cli"
- Fixed some bugs
- Some code improvements

**1.6.0**
- Added the LiveReload plugin and NodeJs  
- Added automatic start of containers after project setup 
- Added the ability to download a specific file from a remote server (for example: madock remote sync file --path app/etc/config.php)    
- Now changed project configuration is applied only after setup or rebuild commands   
- Fixed some bugs and added some improvements 

**1.5.0**
- Added new options for the setup command:    
  - --download - Download the specific Magento version from Composer to the container
  - --install - Install Magento, Shopware, etc. from the source code
- Added new command madock db info. This command prints data for connecting to the database. The output contains a port (permanent) for connecting such database programs as HeidiSQL, MySQL Workbench, and others
- Support Windows OS

**1.4.0**
- Added
  - Kibana  
  - CHANGELOG.md    
  - MADOCK_VERSION in global config.txt 
  - new functionality with services. For example: madock service phpmyadmin on  
- Fixed   
  - text of warning with DB import selecting

**1.3.0**
- For media, js, css requests it was added a new container without Xdebug. This improvement decreases load when you debug your code

**v1.2.0**
- Added a new command for displaying the status of the project   
  - madock status

**v1.1.0**
- Added support for PHP 8.1
- Added support for SSL certificates. Now you can use HTTPS in local development

**v1.0.3**
- Fixed remote sync DB

**v1.0.2**
- Added  
  - Additional logging for sync
  - Validation of project folder name  
- Fixed  
  - Mapping for the general config  
  - Remove compression for an image in png format   
  - Improve sync media files    

**v1.0.1**
- Remove the unison container for macOS

**v1.0.0**
- change docs