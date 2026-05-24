# Shopify

madock can scaffold a Shopify project from any of four SDK/framework presets, each backed by a different container stack:

| Preset            | Language    | Use case                                           | Stack                          |
|-------------------|-------------|----------------------------------------------------|--------------------------------|
| `hydrogen`        | Node + TS   | Headless storefront (deploys to Shopify Oxygen)    | Node 22 only                   |
| `app-remix`       | Node + TS   | Embedded Shopify App for App Store                 | Node 22 + Prisma/SQLite        |
| `api-php`         | PHP         | Backend integration via official shopify-api SDK   | PHP 8.3 + MariaDB + Redis      |
| `laravel-shopify` | PHP/Laravel | Full Shopify App on Laravel (Kyon147/laravel-shopify) | PHP 8.3 + Node + MariaDB + Redis |

## Quick start

Pick the preset and let madock scaffold + boot containers in one shot:

```bash
# In an empty project directory
madock setup -d -i -s --platform shopify --preset hydrogen          # Node storefront
madock setup -d -i -s --platform shopify --preset app-remix         # Node embedded app
madock setup -d -i -s --platform shopify --preset api-php           # PHP backend SDK
madock setup -d -i -s --platform shopify --preset laravel-shopify   # PHP/Laravel app
```

Without `--preset` the setup wizard pops a picker. Aliases work too: `--preset node`, `--preset storefront` → hydrogen; `--preset app`, `--preset remix` → app-remix; `--preset php`, `--preset api` → api-php; `--preset laravel` → laravel-shopify.

## Preset details

### `hydrogen` — Headless storefront

Scaffolds `npm create @shopify/hydrogen@latest` (Remix on Vite, TypeScript, Oxygen worker). Install runs `npm install` and:

* patches `package.json` to add `--host` to the `dev` script so the Hydrogen dev server binds `0.0.0.0` (default is 127.0.0.1)
* attempts to add `server.allowedHosts: true` to `vite.config.ts` (Hydrogen wraps Vite via the Oxygen plugin; if the dev server still returns `Blocked request. This host ... is not allowed.` add the project host manually — see gotchas below)

Wire it to a real store by setting in `.env`:

```env
PUBLIC_STORE_DOMAIN=your-store.myshopify.com
PUBLIC_STOREFRONT_API_TOKEN=...
SESSION_SECRET=...
```

Open `https://loc.<project>.com` — Hydrogen serves the storefront via the nodejs container on port 3000.

### `app-remix` — Embedded Shopify App

Scaffolds `npm init @shopify/app@latest` (Remix + Prisma + App Bridge). Install runs `npm install` and `npx prisma generate && npx prisma migrate deploy` (Prisma uses SQLite locally — no DB container needed).

To start the Partner tunnel + Admin install flow:

```bash
madock bash
npx shopify app dev
```

The Shopify CLI prompts for the Partner account + creates an ngrok-style tunnel. The local dev server itself runs on port 3000 inside the nodejs container.

### `api-php` — Backend integration SDK

Scaffolds a `composer init` project pinned to `shopify/shopify-api:^7.0`. No framework — just the SDK. Install runs `composer install`. Use case: cron jobs / ETL scripts that sync Shopify orders with an existing PHP backend.

Bootstrap your scripts with:

```php
use Shopify\Context;

Context::initialize(
    apiKey: getenv('SHOPIFY_API_KEY'),
    apiSecretKey: getenv('SHOPIFY_API_SECRET'),
    scopes: ['read_products'],
    hostName: 'loc.<project>.com',
    sessionStorage: new FileSessionStorage(),
    apiVersion: '2024-10',
);
```

### `laravel-shopify` — Full Laravel App

Scaffolds `composer create-project laravel/laravel` + adds `kyon147/laravel-shopify`. Install:

* rewrites `.env` (APP_URL, DB_CONNECTION=mysql, DB_HOST=db, DB credentials from project config)
* runs `composer install`, `composer require kyon147/laravel-shopify`
* `php artisan key:generate`, `migrate`, `vendor:publish --tag=shopify-config --tag=shopify-routes`

Edit `config/shopify-app.php` (API key/secret/scopes), then visit `/authenticate?shop=<your-store>.myshopify.com` to wire OAuth.

## Services per preset

| Service       | hydrogen | app-remix | api-php | laravel-shopify |
|---------------|:--------:|:---------:|:-------:|:---------------:|
| nodejs (Node 22) | ✓     | ✓         | —       | —               |
| php (PHP 8.3)    | —     | —         | ✓       | ✓               |
| nginx            | ✓     | ✓         | ✓       | ✓               |
| MariaDB          | —     | —         | ✓       | ✓               |
| Redis            | —     | —         | ✓       | ✓               |
| Node inside PHP  | —     | —         | —       | ✓ (asset pipeline) |

`madock service:enable phpmyadmin / pgadmin / rabbitmq / grafana` works on PHP-stack presets like for any other PHP platform.

## Switching presets

The preset is stored as `shopify/preset` in `config.xml`. Change it and re-run `madock rebuild` to switch stacks (you'll likely want to start in a fresh directory — the scaffolded project layout differs significantly between presets).

## Commands

* `madock shopify <command>` — runs the Shopify CLI inside the container. Examples:
  * Hydrogen: `madock bash` then `npx shopify hydrogen dev`
  * Remix app: `madock bash` then `npx shopify app dev`
* `madock composer <command>` — Composer inside the PHP container (api-php / laravel-shopify only).
* `madock install` — re-run install. Hydrogen/Remix install is idempotent (npm install + patches are no-op on second run). Laravel install also re-runs migrations safely.
* `madock start` / `madock stop` / `madock restart` — standard.
* `madock db:export` / `madock db:import` — DB dumps (PHP-stack presets only).

## Common gotchas

### Hydrogen returns "Blocked request. This host ... is not allowed."

Hydrogen wraps Vite via the Oxygen plugin and overrides the `server` config. madock attempts to inject `server.allowedHosts: true` into `vite.config.ts`, but if Oxygen's wrapper clobbers it, edit `vite.config.ts` manually:

```ts
export default defineConfig({
  server: {
    host: true,
    allowedHosts: ['loc.<project>.com', '.test'],
  },
  plugins: [hydrogen(), oxygen(), reactRouter()],
  // ...
});
```

Restart with `madock restart`.

### Remix app: "Partner authentication required"

`npx shopify app dev` needs a Shopify Partner account. Sign in once at https://partners.shopify.com, then `madock bash` + `npx shopify app dev` to spin up the tunnel.

### Laravel app: routes return 404

`Kyon147/laravel-shopify` ships its own routes; if `php artisan vendor:publish --tag=shopify-routes` didn't run, do it manually: `madock bash` + `php artisan vendor:publish --tag=shopify-routes`.

### Hydrogen dev server returns 502 right after install

The default `dev` script in package.json binds to 127.0.0.1, so nginx can't reach it. madock appends `--host` to the script during install (`shopify hydrogen dev --host` is a boolean flag that switches to 0.0.0.0). If you regenerate the project from upstream, re-run `madock install` to re-apply the patch.

### Legacy: previous madock Shopify defaults

Older madock versions used the `shopify/shopify-app-template-php` Laravel template as the default. That flow is preserved by the `api-php` and `laravel-shopify` presets. The old `madock shopify:web` / `madock shopify:web:frontend` shortcuts still work — they target the PHP-container subdirectories as before.

## Tips

* `madock bash` enters the main service container as the project user (`node` for hydrogen/app-remix, `www-data` for api-php/laravel-shopify).
* For Hydrogen + Remix: don't run `npm install` on the host (host Node may differ from container Node 22) — always inside `madock bash`.
