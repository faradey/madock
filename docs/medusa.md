# Medusa.js

madock runs Medusa.js commerce projects locally inside Docker: Node.js backend, PostgreSQL, Redis, optional Meilisearch, and an optional Next.js storefront container.

## Quick start

```bash
# In an empty directory or your existing Medusa project root
madock setup --platform medusa
```

The setup wizard offers presets (Latest, Stable, Legacy) and writes a `config.xml` with sane defaults. You can skip the wizard with `--preset`:

```bash
madock setup --platform medusa --preset latest   # Medusa 2.x with Node 22, PostgreSQL 17, Redis 7.4
madock setup --platform medusa --preset stable   # Medusa 2.0 baseline with Node 20, PostgreSQL 16, Redis 7.2
madock setup --platform medusa --preset legacy   # Medusa 1.x with Node 18, PostgreSQL 14, Redis 7.0
```

Auto-detection: if your project root has a `package.json` that depends on `@medusajs/medusa` or `@medusajs/framework`, `madock setup` (without `--platform`) will pick the medusa platform automatically.

## Project layout

By default madock assumes the following layout:

```
<project>/
├── package.json           # Medusa backend
├── medusa-config.ts       # backend config
└── storefront/            # optional: Next.js storefront (when service:enable storefront)
    ├── package.json
    └── ...
```

The `storefront` subfolder is mounted into the storefront container at `/var/www/storefront`. To use a different subfolder, set `medusa/storefront/path` in `config.xml`.

## Commands

* `madock medusa <command>` — runs `npx medusa <command>` inside the backend container.
* `madock install` — writes `.env`, runs `npx medusa db:migrate`, and creates a default admin user (`admin@example.com` / `admin`).
* `madock start` / `madock stop` / `madock restart` — same as for other platforms.
* `madock service:enable meilisearch` — starts the optional Meilisearch container (search backend for `@rokmohar/medusa-plugin-meilisearch`).
* `madock service:enable storefront` — starts the optional Next.js storefront container.
* `madock db:export` / `madock db:import` — PostgreSQL dumps.

## Services

| Service       | Default | Default version       | Notes                                                  |
|---------------|---------|-----------------------|--------------------------------------------------------|
| Node.js       | on      | 22.11 (Latest preset) | Backend at internal port 9000, proxied via nginx       |
| PostgreSQL    | on      | postgres:17           | Volume `dbdata`                                        |
| Redis         | on      | 7.4                   | Used by Medusa's job scheduler and cache               |
| Meilisearch   | off     | 1.11.3                | Enable with `service:enable meilisearch`               |
| Storefront    | off     | node 22.11            | Enable with `service:enable storefront`                |
| RabbitMQ      | off     | 3.12                  | Available if you use the events module backed by RMQ   |
| pgAdmin       | off     | latest                | DB browser, enable with `service:enable pgadmin`       |

## Ports

madock allocates host ports dynamically (starting from `17000`) to avoid collisions between projects. Run `madock info` or `madock info:ports` to see the current allocation.

* **Backend (`nodejs`)** — only reachable via the project's nginx host (`https://loc.<project>.com`). No direct host port to avoid conflicts. The nginx upstream is configured to hit `nodejs:9000`.
* **Storefront** — `http://localhost:<port/storefront>` on the host. The container always listens on `8000` internally; madock maps an unused host port to it.
* **Meilisearch** — `http://localhost:<port/meilisearch>` on the host. The container listens on `7700` internally.
* **PostgreSQL** — `localhost:<port/db>` for tools like psql/DBeaver.

The Medusa backend connects to Postgres and Redis using their internal docker hostnames (`db:5432`, `redis:6379`), so there's nothing to configure in `.env` beyond what `madock install` writes.

## Storefront

The storefront service runs the Medusa Next.js storefront starter (or your custom Next.js app) in development mode. It expects:

* a `storefront/` directory (override with `medusa/storefront/path` in `config.xml`)
* a `package.json` with a `dev` script (the default scaffolding from `npx create-medusa-app@latest` already provides one)

On first start the container runs `yarn install && yarn dev`. The container env vars wire it to the backend:

* `MEDUSA_BACKEND_URL=http://nodejs:9000` — server-side (SSR) calls inside the docker network
* `NEXT_PUBLIC_MEDUSA_BACKEND_URL=https://loc.<project>.com` — browser-side (CSR) calls go through the public nginx host, because the browser cannot resolve docker hostnames. Override via `medusa/storefront/public_backend_url`
* `NEXT_PUBLIC_BASE_URL=http://localhost:<host_port>`
* `NEXT_PUBLIC_DEFAULT_REGION=us` (override via `medusa/storefront/region` in config)
* `WATCHPACK_POLLING=true`, `CHOKIDAR_USEPOLLING=true` — keep HMR working on macOS bind mounts where inotify events aren't forwarded. See [HMR / file watching on macOS](#hmr--file-watching-on-macos) below

> **Note**: storefront is a Medusa-specific service. Its config keys live under the `<medusa>` section in `config.xml` (`medusa/storefront/*`), following the same convention as Magento-specific services like `magento/cloud` and `magento/mftf`. The `service:enable storefront` short name maps to the `medusa/storefront/enabled` config key and works only when the project platform is `medusa`.

If the `storefront/` folder is missing or empty, the container prints a message and stays idle so it doesn't crash-loop.

## Meilisearch

Meilisearch is a popular search backend for Medusa via [`@rokmohar/medusa-plugin-meilisearch`](https://github.com/rokmohar/medusa-plugin-meilisearch). After `service:enable meilisearch`:

* Container is reachable inside the docker network at `http://meilisearch:7700`.
* Host port: `http://localhost:<port/meilisearch>`.
* Master key: `masterKey` (override `search/meilisearch/master_key` in `config.xml` before enabling).

Add the plugin to your Medusa backend, configure it with `host: http://meilisearch:7700` and the master key, and you're set.

## HMR / file watching on macOS

The Medusa storefront container ships with `WATCHPACK_POLLING=true` and `CHOKIDAR_USEPOLLING=true` so that Next.js HMR works on macOS bind mounts (where Docker Desktop does not forward inotify events).

For the backend (`nodejs` service running `medusa develop`) and any other container that watches files, see the general guide [macos-hmr.md](macos-hmr.md). It covers Next.js, Chokidar, nodemon, ts-node-dev, tsc, vite, gulp, and grunt.

## Tips

* Run `madock medusa db:migrate` after updating dependencies — keeps the database in sync with the latest module schemas.
* Use `madock bash` to enter the backend container as the `node` user (workdir `/var/www/html`).
* The built-in Medusa admin UI is reachable at `https://loc.<project>.com/app` once the backend is running.
