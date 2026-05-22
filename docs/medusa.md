# Medusa.js

madock runs Medusa.js commerce projects locally inside Docker: Node.js backend, PostgreSQL, Redis, optional Meilisearch, plus an auto-provisioned Next.js storefront container.

## Quick start

```bash
# In an empty directory or your existing Medusa project root
madock setup -d -i -s --platform medusa
```

`-d` downloads `medusa-starter-default` + `nextjs-starter-medusa`, `-i` runs the install pipeline end-to-end, `-s` starts containers. The setup wizard offers presets (Latest, Stable, Legacy) and writes a `config.xml` with sane defaults. You can skip the wizard with `--preset`:

```bash
madock setup --platform medusa --preset latest   # Medusa 2.x with Node 22, PostgreSQL 17, Redis 7.4
madock setup --platform medusa --preset stable   # Medusa 2.0 baseline with Node 20, PostgreSQL 16, Redis 7.2
madock setup --platform medusa --preset legacy   # Medusa 1.x with Node 18, PostgreSQL 14, Redis 7.0
```

Auto-detection: if your project root has a `package.json` that depends on `@medusajs/medusa` or `@medusajs/framework`, `madock setup` (without `--platform`) will pick the medusa platform automatically.

## What `madock install` does

End-to-end pipeline inside the containers:

1. Writes backend `.env` (`DATABASE_URL`, `REDIS_URL`, JWT/cookie secrets, CORS hosts).
2. `yarn install` in the backend.
3. Patches `medusa-config.ts` to add `admin.vite.server.allowedHosts: true` (Vite 5 rejects unknown `Host` headers by default).
4. Patches `node_modules/@medusajs/medusa/dist/commands/develop.js` to inject regex ignores so the backend's chokidar watcher does not reload on `storefront/` or any nested `node_modules/` writes.
5. `npx medusa db:setup --db <name>` — creates the database, runs all module migrations + migration scripts, syncs links.
6. `npx medusa user --email admin@example.com --password admin` — admin login.
7. `yarn seed` when `package.json` defines it — provisions the Europe region (gb, de, dk, se, fr, es, it), sales channel, shipping options, demo products.
8. Restarts the backend container so the freshly migrated/seeded DB boots cleanly.
9. Polls `/health`, then reuses or creates a publishable API key (bound to the default sales channel) and appends `NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=…` to the backend `.env`.
10. Writes `storefront/.env.local` with the backend URLs + publishable key + default region.
11. `yarn install` in the storefront, then restarts the storefront container.

## Project layout

```
<project>/
├── package.json           # Medusa backend
├── medusa-config.ts       # backend config (auto-patched for Vite allowedHosts)
├── .env                   # written by madock install
└── storefront/            # Next.js storefront (auto-cloned, mounted at /var/www/html/storefront)
    ├── package.json
    └── .env.local         # written by madock install
```

The `storefront` subfolder is mounted into the storefront container at `/var/www/html/storefront`. To use a different subfolder, set `medusa/storefront/path` in `config.xml`. To use a fork of the Next.js starter, set `medusa/storefront/git_url`.

## Routing

nginx splits the public host between the backend and the storefront:

| Path prefix                        | Upstream         |
|------------------------------------|------------------|
| `/health`, `/app`, `/store`, `/admin`, `/auth` | `nodejs:9000` (backend) |
| everything else (incl. `/`)        | `storefront:8000` (Next.js) |

DNS for both upstreams is resolved lazily through Docker's embedded resolver (`127.0.0.11`), so nginx starts even before the storefront container is ready.

## Commands

* `madock medusa <command>` — runs `npx medusa <command>` inside the backend container.
* `madock install` — full pipeline described above. Re-run after a backend `yarn install` to re-apply the develop.js watcher patch.
* `madock start` / `madock stop` / `madock restart` — same as for other platforms.
* `madock service:enable meilisearch` — starts the optional Meilisearch container (search backend for `@rokmohar/medusa-plugin-meilisearch`).
* `madock db:export` / `madock db:import` — PostgreSQL dumps.

## Services

| Service       | Default | Default version       | Notes                                                  |
|---------------|---------|-----------------------|--------------------------------------------------------|
| Node.js (backend) | on  | 22.11 (Latest preset) | Internal port 9000, proxied via nginx                  |
| Storefront    | on      | matches backend node  | Next.js on internal port 8000, mapped to `port/storefront` on host |
| PostgreSQL    | on      | postgres:17           | Volume `dbdata`                                        |
| Redis         | on      | 7.4                   | Used by Medusa's job scheduler and cache               |
| Meilisearch   | off     | 1.11.3                | Enable with `service:enable meilisearch`               |
| RabbitMQ      | off     | 3.12                  | Available if you use the events module backed by RMQ   |
| pgAdmin       | off     | latest                | DB browser, enable with `service:enable pgadmin`       |

## Ports

madock allocates host ports dynamically (starting from `17000`) to avoid collisions between projects. Run `madock info` or `madock info:ports` to see the current allocation.

* **Backend (`nodejs`)** — only reachable via the project's nginx host (`https://loc.<project>.com`). No direct host port to avoid conflicts. The nginx upstream is configured to hit `nodejs:9000`.
* **Storefront** — `http://localhost:<port/storefront>` direct, or via the project nginx host at `/`. The container listens on `8000` internally.
* **Meilisearch** — `http://localhost:<port/meilisearch>` on the host. The container listens on `7700` internally.
* **PostgreSQL** — `localhost:<port/db>` for tools like psql/DBeaver.

The Medusa backend connects to Postgres and Redis using their internal docker hostnames (`db:5432`, `redisdb:6379`), so there's nothing to configure in `.env` beyond what `madock install` writes.

## Storefront

Runs the Medusa Next.js storefront starter (`medusajs/nextjs-starter-medusa`) in dev mode. Cloned automatically by `madock setup -d` into the `storefront/` subfolder, installed by `madock install`. Env vars written to `storefront/.env.local`:

* `MEDUSA_BACKEND_URL=http://nodejs:9000` — server-side (SSR) calls inside the docker network.
* `NEXT_PUBLIC_MEDUSA_BACKEND_URL=https://loc.<project>.com` — browser-side (CSR) calls go through the public nginx host. Override via `medusa/storefront/public_backend_url`.
* `NEXT_PUBLIC_BASE_URL=https://loc.<project>.com`.
* `NEXT_PUBLIC_DEFAULT_REGION=gb` — first country in the seed's Europe region. Override via `medusa/storefront/region`.
* `NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=pk_…` — the key reused or created during install.

The container also receives `WATCHPACK_POLLING=true`, `CHOKIDAR_USEPOLLING=true` from `docker-compose` to keep HMR working on macOS bind mounts where inotify events aren't forwarded. See [macos-hmr.md](macos-hmr.md).

To disable the storefront entirely, set `medusa/storefront/enabled` to `false` in `config.xml` and re-run `madock rebuild`. nginx will fall back to the standard single-upstream proxy config pointed at the backend.

If the `storefront/` folder is missing or empty when the container starts, the smart entrypoint prints a message and idles until `package.json` and the install marker appear.

## Meilisearch

Meilisearch is a popular search backend for Medusa via [`@rokmohar/medusa-plugin-meilisearch`](https://github.com/rokmohar/medusa-plugin-meilisearch). After `service:enable meilisearch`:

* Container is reachable inside the docker network at `http://meilisearch:7700`.
* Host port: `http://localhost:<port/meilisearch>`.
* Master key: `masterKey` (override `search/meilisearch/master_key` in `config.xml` before enabling).

Add the plugin to your Medusa backend, configure it with `host: http://meilisearch:7700` and the master key, and you're set.

## HMR / file watching on macOS

The Medusa storefront container ships with `WATCHPACK_POLLING=true` and `CHOKIDAR_USEPOLLING=true` so that Next.js HMR works on macOS bind mounts (where Docker Desktop does not forward inotify events).

For the backend (`nodejs` service running `medusa develop`) and any other container that watches files, see the general guide [macos-hmr.md](macos-hmr.md). It covers Next.js, Chokidar, nodemon, ts-node-dev, tsc, vite, gulp, and grunt.

## Common gotchas

### Backend reload loop after a fresh `yarn install` in the backend

`madock install` patches `node_modules/@medusajs/medusa/dist/commands/develop.js` to ignore `storefront/` and nested `node_modules/`. A plain `yarn install` (or any dependency upgrade) rewrites that file, so the loop comes back. Re-run `madock install` (or call the patch step manually) to re-apply.

### Backend logs PostgreSQL SSL errors

The bundled `postgres` image does not run with TLS. `madock install` writes `DATABASE_URL` with `?sslmode=disable` so the pg driver skips negotiation. If you wrote the `.env` file yourself, append `?sslmode=disable` to the URL.

### "redisUrl not found. A fake redis instance will be used."

Medusa v2 does not read `REDIS_URL` from the environment. Wire the running Redis container into `medusa-config.ts` explicitly:

```ts
projectConfig: {
  databaseUrl: process.env.DATABASE_URL,
  redisUrl: process.env.REDIS_URL,   // madock writes redis://redisdb:6379
  workerMode: "shared",
  http: { ... },
},
modules: [
  {
    resolve: "@medusajs/medusa/cache-redis",
    options: { redisUrl: process.env.REDIS_URL },
  },
  {
    resolve: "@medusajs/medusa/event-bus-redis",
    options: { redisUrl: process.env.REDIS_URL },
  },
  {
    resolve: "@medusajs/medusa/workflow-engine-redis",
    options: {
      redis: { url: process.env.REDIS_URL },
    },
  },
],
```

Restart the backend after editing. The `Local Event Bus installed. This is not recommended for production.` warning will go away too.

### Skipped `madock install` and the storefront crashes with "Missing required environment variables: NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY"

Medusa v2 requires every storefront request to carry a publishable API key. `madock install` seeds one automatically. If you bypassed the install step, run it once or create the key manually:

```bash
TOKEN=$(curl -sk https://loc.<project>.com/auth/user/emailpass \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin"}' | jq -r .token)

KEY=$(curl -sk -X POST https://loc.<project>.com/admin/api-keys \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"storefront","type":"publishable"}' | jq -r .api_key.token)

cat >> storefront/.env.local <<EOF
NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY=$KEY
EOF

madock restart
```

### Skipped `madock install` and the admin UI returns "Blocked request"

`madock install` patches `medusa-config.ts` to whitelist the project's nginx host in Medusa Admin's bundled Vite dev server. If you skipped the install step, edit the file manually:

```ts
admin: {
  vite: () => ({
    server: {
      allowedHosts: true,   // or ["loc.<project>.com"]
    },
  }),
},
```

Restart the backend (`madock restart`). The admin UI at `https://loc.<project>.com/app` will load.

## Tips

* Run `madock medusa db:migrate` after updating dependencies — keeps the database in sync with the latest module schemas.
* Use `madock bash` to enter the backend container as the `node` user (workdir `/var/www/html`).
* The built-in Medusa admin UI is reachable at `https://loc.<project>.com/app` once the backend is running.
* Browse the storefront at `https://loc.<project>.com/` — middleware redirects to the default region (`/gb` by default).
