# BigCommerce

madock can scaffold a BigCommerce project from any of four SDK/framework presets, each backed by a different container stack:

| Preset      | Language    | Use case                                                  | Stack                              |
|-------------|-------------|-----------------------------------------------------------|------------------------------------|
| `catalyst`  | Node + TS   | Official headless storefront (Next.js monorepo)           | Node 22 + pnpm                     |
| `stencil`   | Node + Handlebars | Legacy theme dev with @bigcommerce/stencil-cli      | Node 22 + Stencil CLI              |
| `api-php`   | PHP         | Backend integration via bigcommerce/api Composer SDK | PHP 8.3 + MariaDB + Redis        |
| `app-node`  | Node + TS   | Embedded App Marketplace template (Express + Next.js)     | Node 22                            |

## Quick start

```bash
# In an empty project directory
madock setup -d -i -s --platform bigcommerce --preset catalyst     # Next.js storefront
madock setup -d -i -s --platform bigcommerce --preset stencil      # Cornerstone theme dev
madock setup -d -i -s --platform bigcommerce --preset api-php      # PHP backend SDK
madock setup -d -i -s --platform bigcommerce --preset app-node     # Node embedded app
```

Without `--preset` the setup wizard pops a picker. Aliases work: `--preset next` / `--preset storefront` → catalyst; `--preset theme` → stencil; `--preset php` / `--preset api` → api-php; `--preset app` / `--preset node` → app-node.

## Preset details

### `catalyst` — Headless storefront

Clones the upstream `bigcommerce/catalyst` monorepo (pnpm workspaces + turbo). Install runs `pnpm install` for all workspaces and rewrites the root `package.json` `dev` script:

* `scripts.dev:catalyst` ← original turbo dev (with `--filter ./core -- -H 0.0.0.0` so the Next.js app binds the project's nginx upstream)
* `scripts.dev` ← `sleep infinity` (parks the container — Catalyst's pre-dev `generate` step fetches GraphQL schema from your live store, so it can't boot without real store credentials)

Wire it to your store, then run dev:

```bash
# 1. Edit core/.env.local
BIGCOMMERCE_STORE_HASH=...
BIGCOMMERCE_ACCESS_TOKEN=...
BIGCOMMERCE_STOREFRONT_TOKEN=...

# 2. Start dev
madock bash
npm run dev:catalyst
```

Open `https://loc.<project>.com` — turbo runs Next.js dev for `core/` only on port 3000, nginx proxies through.

### `stencil` — Cornerstone theme dev

Clones `bigcommerce/cornerstone` (canonical theme starter). Install runs `npm install` + `npm install -g @bigcommerce/stencil-cli` so the `stencil` command is on PATH. `scripts.dev` is parked because `stencil start` needs interactive API token entry.

```bash
madock bash
stencil init                # paste store URL + API token
stencil start --tunnel      # opens an ngrok-style tunnel against the live store
```

### `api-php` — Backend integration SDK

Scaffolds a `composer init` project pinned to `bigcommerce/api:^3.3`. No framework — just the SDK. Use case: cron jobs / ETL scripts syncing BigCommerce orders with an existing PHP backend.

```php
use Bigcommerce\Api\Client;

Client::configure([
    'storeUrl'   => 'https://your-store.mybigcommerce.com',
    'username'   => 'admin',
    'apiKey'     => '...',
]);
```

### `app-node` — Embedded Node app

Clones `bigcommerce/sample-app-nodejs` (Express + Next.js with OAuth handshake). Install runs `npm install` and parks `scripts.dev` as `scripts.dev:bc` (the real dev needs interactive Developer auth + ngrok-style tunnel).

```bash
# Wire OAuth in .env (CLIENT_ID, CLIENT_SECRET, AUTH_CALLBACK)
madock bash
npm run dev:bc
```

## Services per preset

| Service          | catalyst | stencil | api-php | app-node |
|------------------|:--------:|:-------:|:-------:|:--------:|
| nodejs (Node 22 + pnpm) | ✓ | ✓       | —       | ✓        |
| php (PHP 8.3)    | —        | —       | ✓       | —        |
| nginx            | ✓        | ✓       | ✓       | ✓        |
| MariaDB          | —        | —       | ✓       | —        |
| Redis            | —        | —       | ✓       | —        |

`madock service:enable phpmyadmin / pgadmin / rabbitmq / grafana` works on the api-php preset like any other PHP platform.

## Switching presets

The preset is stored as `bigcommerce/preset` in `config.xml`. Change it and re-run `madock rebuild` to switch stacks. The scaffolded project layout differs significantly between presets — usually you'll want a fresh directory.

## Commands

* `madock bigcommerce <command>` (alias `madock bc`) — runs `<command>` inside the preset's main container (nodejs for catalyst/stencil/app-node, php for api-php).
* `madock composer <command>` — Composer inside the PHP container (api-php only).
* `madock install` — re-run install. All preset installs are idempotent (marker comments + existence checks).
* `madock start` / `madock stop` / `madock restart` — standard.
* `madock db:export` / `madock db:import` — DB dumps (api-php only).

## Common gotchas

### Catalyst: "Missing store hash" on `npm run dev:catalyst`

Catalyst's `generate` script fetches the GraphQL schema from your live BigCommerce store. It can't run with placeholder credentials. Add real values to `core/.env.local` (BIGCOMMERCE_STORE_HASH, BIGCOMMERCE_ACCESS_TOKEN) from your Catalyst Console.

### Catalyst: "unexpected argument '-H' found" from turbo

Upstream Catalyst's `dev` script puts `-H 0.0.0.0` directly after `turbo run dev`, but current turbo needs `--` to forward args. madock auto-fixes this by rewriting the script to `turbo run dev --filter ./core -- -H 0.0.0.0` during install. If you regenerate from upstream, re-run `madock install` to re-apply.

### Stencil: "stencil init" fails with "Unable to verify auth_token"

The API token paste must match the store URL. Use `stencil init -u https://your-store.mybigcommerce.com` and paste the API account token from the BigCommerce admin (Settings → API → Store-level API accounts).

### app-node: "Missing CLIENT_ID" on dev:bc

The Node app expects BigCommerce Developer credentials. Create a draft app at https://devtools.bigcommerce.com and copy CLIENT_ID + CLIENT_SECRET into `.env`. Set AUTH_CALLBACK to your tunnel's HTTPS URL.

## Tips

* `madock bash` enters the main service container as the project user (`node` for catalyst/stencil/app-node, `www-data` for api-php).
* Catalyst monorepo workspaces require pnpm — madock pre-installs it globally in the nodejs image so the entrypoint's `pnpm dev` works after restart.
* For pure backend work without a storefront, pick `api-php` — it's the lightest preset (PHP + MariaDB + Redis, no node monorepo overhead).
