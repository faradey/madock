# Spree Commerce

madock runs Spree Commerce projects locally inside Docker: Rails admin backend, PostgreSQL, Redis, optional Sidekiq worker, plus an auto-provisioned Next.js storefront container.

## Quick start

```bash
# In an empty directory or your existing Spree project root
madock setup -d -i -s --platform spree
```

`-d` downloads `spree/spree_starter` + `spree/storefront`, `-i` runs the install pipeline end-to-end, `-s` starts containers. The setup wizard offers presets (5.x Latest, 4.x Stable) and writes a `config.xml` with sane defaults. You can skip the wizard with `--preset`:

```bash
madock setup --platform spree --preset 5         # Spree 5.x with Ruby 4.0, PostgreSQL 16, Redis 7.2 (Rails 8)
madock setup --platform spree --preset 4         # Spree 4.10.x with Ruby 3.2, PostgreSQL 15, Redis 7.0 (Rails 7.1)
```

Auto-detection: if your project root has a `Gemfile` or `Gemfile.lock` that depends on `spree`, `madock setup` (without `--platform`) picks the spree platform automatically.

## What `madock install` does

End-to-end pipeline inside the containers:

1. Writes backend `.env` (`DATABASE_URL`, `REDIS_URL`, `RAILS_ENV=development`, `SECRET_KEY_BASE`, `BINDING=0.0.0.0`, `PORT=3000`, admin credentials).
2. Pins `.ruby-version` and `Gemfile.lock`'s `RUBY VERSION` line to the docker image's actual Ruby — Spree starter pins patch levels (e.g. 4.0.1) that Docker Hub doesn't always publish, so we lock to whatever's installed (e.g. 4.0.5) before bundler rejects the lockfile.
3. Patches `config/environments/development.rb` with `config.assume_ssl = true` so Rails generates `https://` redirects under the nginx TLS-terminating proxy.
4. `bundle install` (uses bundler 4 already baked into the image).
5. `bundle exec rails db:prepare` — schema + migrations + seed.
6. `bundle exec rails spree:admin:tailwindcss:build` — without this `/admin_user/sign_in` 500s with `Propshaft::MissingAssetError (spree/admin/application.css)`.
7. `bundle exec rails spree:search:reindex` (best-effort, skipped on missing search backend).
8. `bundle exec rails spree_sample:load` — Europe-wide demo data: 211 products, 20 customers, sample orders, a publishable API key.
9. Restarts the ruby container so PID 1 picks up the new `.env` and migrated DB.
10. Extracts the publishable API key seeded into `Spree::ApiKey` (via `rails runner`) and writes `storefront/.env.local` with `SPREE_API_URL=http://ruby:3000`, `SPREE_PUBLISHABLE_KEY=pk_…`, plus `NEXT_PUBLIC_*` site/locale defaults.
11. `yarn install` inside the storefront container, then restarts it so the smart entrypoint picks up the freshly seeded env + deps.

## Project layout

```
<project>/
├── Gemfile               # Spree backend (Rails)
├── Gemfile.lock          # auto-patched to match container Ruby
├── config/
│   └── environments/
│       └── development.rb # auto-patched with assume_ssl
├── bin/                  # Rails binstubs
├── .env                  # written by madock install
└── storefront/           # Next.js storefront (auto-cloned, mounted at /var/www/html/storefront)
    ├── package.json
    └── .env.local        # written by madock install
```

`storefront` subfolder is mounted into the storefront container at `/var/www/html/storefront`. Override via `spree/storefront/path` in `config.xml`. Use a fork of the Next.js starter by setting `spree/storefront/git_url`.

## Routing

nginx splits the public host between the Rails backend and the Next.js storefront:

| Path prefix                                          | Upstream                  |
|------------------------------------------------------|---------------------------|
| `/admin`, `/admin_user`, `/api`, `/up`, `/rails`, `/assets`, `/webhooks`, `/oauth`, `/cable` | `ruby:3000` (Rails)       |
| everything else (incl. `/`)                          | `storefront:3001` (Next)  |

DNS for both upstreams resolves lazily through Docker's embedded resolver (`127.0.0.11`) so nginx starts even before the storefront container is ready. When `spree/storefront/enabled=false`, nginx falls back to the standard single-upstream proxy pointed at the backend.

## Commands

* `madock spree <command>` — runs `bundle exec rails <command>` inside the backend container (e.g. `madock spree console`, `madock spree routes`, `madock spree db:migrate`).
* `madock install` — full pipeline above. Re-run after a backend `bundle install` to re-apply the `.ruby-version` / Gemfile.lock pin if Spree publishes a new patch level.
* `madock start` / `madock stop` / `madock restart` — same as for other platforms.
* `madock service:enable sidekiq` — starts the optional Sidekiq worker container (same ruby image, runs `bundle exec sidekiq`).
* `madock db:export` / `madock db:import` — PostgreSQL dumps.

## Services

| Service       | Default | Default version             | Notes                                                      |
|---------------|---------|-----------------------------|------------------------------------------------------------|
| Ruby (backend)| on      | 4.0.5 (Latest preset)       | Internal port 3000, proxied via nginx                      |
| Storefront    | on      | node 22.20.0                | Next.js on internal port 3001, mapped to `port/storefront` on host |
| PostgreSQL    | on      | postgres:16.4               | Volume `dbdata`                                            |
| Redis         | on      | 7.2.5                       | Used by Sidekiq, Rails cache, Action Cable                 |
| Sidekiq       | off     | matches ruby image          | Enable with `service:enable sidekiq`                       |
| pgAdmin       | off     | latest                      | DB browser, enable with `service:enable pgadmin`           |

## Ports

madock allocates host ports dynamically (starting from `17000`) to avoid collisions between projects. Run `madock info` or `madock info:ports` to see the current allocation.

* **Backend (`ruby`)** — only reachable via the project's nginx host (`https://loc.<project>.com`). No direct host port to avoid conflicts. The nginx upstream is configured to hit `ruby:3000`.
* **Storefront** — `http://localhost:<port/storefront>` direct, or via the project nginx host at `/`. The container listens on `3001` internally.
* **PostgreSQL** — `localhost:<port/db>` for tools like psql/DBeaver.

The Spree backend connects to Postgres and Redis using their internal docker hostnames (`db:5432`, `redisdb:6379`), so there's nothing to configure in `.env` beyond what `madock install` writes.

## Storefront

Runs `spree/storefront` (Next.js 16, TypeScript) in dev mode. Cloned automatically by `madock setup -d` into the `storefront/` subfolder, installed by `madock install`. Env vars written to `storefront/.env.local`:

* `SPREE_API_URL=http://ruby:3000` — server-side (SSR) calls inside the docker network.
* `SPREE_PUBLISHABLE_KEY=pk_…` — extracted from the `Spree::ApiKey` row seeded by `spree_sample:load`.
* `NEXT_PUBLIC_SITE_URL=https://loc.<project>.com` — used for SEO meta tags, Open Graph URLs, canonical links.
* `NEXT_PUBLIC_DEFAULT_COUNTRY=us` (override via `spree/storefront/country`).
* `NEXT_PUBLIC_DEFAULT_LOCALE=en` (override via `spree/storefront/locale`).
* `NEXT_PUBLIC_STORE_NAME="Spree Store"` (override via `spree/storefront/store_name`).

The container also receives `WATCHPACK_POLLING=true`, `CHOKIDAR_USEPOLLING=true` from `docker-compose` to keep HMR working on macOS bind mounts where inotify events aren't forwarded. See [macos-hmr.md](macos-hmr.md).

To disable the storefront entirely, set `spree/storefront/enabled` to `false` in `config.xml` and re-run `madock rebuild`. nginx falls back to the standard single-upstream proxy config pointed at the backend, and `/` routes to Spree's default `redirect(301, /admin)`.

If the `storefront/` folder is missing or empty when the container starts, the smart entrypoint prints a message and idles until `package.json` and the install marker appear.

## Sidekiq

Sidekiq is Spree's background job processor (mail, webhooks, search indexing, reports). Enable the optional worker container with `service:enable sidekiq`:

* Same ruby image as the backend, runs `bundle exec sidekiq` against the project's Gemfile.lock.
* Connects to Redis at `redisdb:6379/0` and to the same `DATABASE_URL` as the web container.
* `RAILS_MAX_THREADS=27` matches `bin/dev` defaults from spree_starter; override `SIDEKIQ_DB_POOL` in `.env` if you tune the Postgres pool.

## HMR / file watching on macOS

The Spree storefront container ships with `WATCHPACK_POLLING=true` and `CHOKIDAR_USEPOLLING=true` so Next.js HMR works on macOS bind mounts.

For the backend (`ruby` service running `rails server`), Spring is disabled (`DISABLE_SPRING=1` in `.env`) so file changes are picked up by Rails' built-in `Rails::Server` reloader without the Spring zygote dance. See the general guide [macos-hmr.md](macos-hmr.md) for other watchers.

## Common gotchas

### Backend container exits with `Your Ruby version is X, but your Gemfile specified Y`

`madock install` pins `.ruby-version` and the `RUBY VERSION` line in `Gemfile.lock` to the image's actual Ruby. After a backend `bundle install` that regenerates `Gemfile.lock`, the pin is gone and the container refuses to boot. Re-run `madock install` (or repeat the `sed` step manually) to re-apply.

### `/admin_user/sign_in` returns 500 with `Propshaft::MissingAssetError (spree/admin/application.css)`

`madock install` runs `bundle exec rails spree:admin:tailwindcss:build` once. If you skipped it (or deleted the build output), run:

```bash
madock spree spree:admin:tailwindcss:build
```

Spree's `bin/dev` Procfile.dev also runs `spree:admin:tailwindcss:watch` alongside puma to rebuild on change — start it manually if you're iterating on admin CSS.

### Admin login redirects to `http://` instead of `https://`

Without `config.assume_ssl = true`, Rails generates `http://` URLs in every redirect (Devise sign-in, admin callbacks) because puma sees the request over plain HTTP inside the docker network. `madock install` patches `config/environments/development.rb` once. If you bypassed install, add manually:

```ruby
Rails.application.configure do
  config.assume_ssl = true
  # …
end
```

### Storefront crashes with `Spree client is not configured` / API calls 401

`madock install` extracts the publishable key seeded by `spree_sample:load` and writes it to `storefront/.env.local`. If sample data hasn't been loaded yet, no key exists. Generate one via the Rails console:

```bash
madock spree runner "Spree::ApiKey.create!(store: Spree::Store.default, key_type: 'publishable', name: 'storefront')"
```

Copy the printed `token` into `storefront/.env.local` as `SPREE_PUBLISHABLE_KEY=pk_…` and `madock restart`.

### `yarn install` fails inside the storefront with `engine "node" is incompatible`

Spree storefront's `@inquirer/confirm` transitive dep needs Node 22.13+. The image is pinned to Node 22.20 by default. If you overrode `spree/storefront/version` in `config.xml` to an older Node 22.x release, raise it to 22.13+ (or any Node 23+) and re-`madock rebuild`.

## Tips

* Run `madock spree db:migrate` after pulling new Spree releases or adding gems — keeps schema in sync.
* Use `madock bash` to enter the backend container as the `ruby` user (workdir `/var/www/html`).
* Admin URL: `https://loc.<project>.com/admin` (login: `admin@example.com` / `spree123`).
* Storefront URL: `https://loc.<project>.com/` — middleware redirects to `/us/en` by default.
* Health check: `https://loc.<project>.com/up` (Rails 8 built-in probe).
