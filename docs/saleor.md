# Saleor

madock runs Saleor 3.x commerce projects locally inside Docker: Python (uvicorn/runserver), PostgreSQL, Redis, optional Celery worker and Saleor Dashboard.

## Quick start

```bash
# In an empty directory or your existing Saleor checkout
git clone --branch 3.23 https://github.com/saleor/saleor.git my-saleor
cd my-saleor
madock setup --platform saleor --preset latest
madock start
madock install
```

The setup wizard offers presets (Latest, Stable) and writes a `config.xml` with sane defaults. You can skip the prompts with `--preset`:

```bash
madock setup --platform saleor --preset latest   # Saleor 3.23 / Python 3.12 / PostgreSQL 15 / Redis 7.2
madock setup --platform saleor --preset stable   # Saleor 3.20 baseline
```

Auto-detection: if your project root has a `pyproject.toml`, `uv.lock`, `poetry.lock`, or `requirements.txt` that lists `saleor`, `madock setup` (without `--platform`) picks the saleor platform automatically and uses the version pinned in `pyproject.toml`.

## Commands

* `madock saleor <command>` — runs `python manage.py <command>` inside the python container. Prefers `uv run` when `uv.lock` is present, falls back to plain `python`.
* `madock install` — writes `.env` (SECRET_KEY, DATABASE_URL, REDIS_URL, CELERY_BROKER_URL, ALLOWED_HOSTS, PUBLIC_URL), runs `uv sync --frozen` (or `pip install -r requirements.txt` for older releases), runs `manage.py migrate`, and runs `manage.py populatedb --createsuperuser` to create the default `admin@example.com` / `admin` account plus sample data.
* `madock start` / `madock stop` / `madock restart` — same as other platforms.
* `madock service:enable dashboard` — starts the Saleor Dashboard SPA (`ghcr.io/saleor/saleor-dashboard`) on its own container.
* `madock service:enable worker` — starts a Celery worker (with beat embedded) sharing the python image.
* `madock db:export` / `madock db:import` — PostgreSQL dumps via `pg_dump` / `psql`.

## Services

| Service       | Default | Default version       | Notes                                                                |
|---------------|---------|-----------------------|----------------------------------------------------------------------|
| Python        | on      | 3.12 (Latest preset)  | uvicorn / `manage.py runserver` on internal port 8000, behind nginx  |
| PostgreSQL    | on      | postgres:15           | Volume `dbdata`                                                      |
| Redis         | on      | 7.2.5                 | Used as Django cache backend AND Celery broker (no separate broker)  |
| Dashboard     | off     | 3.23                  | Enable with `service:enable dashboard`                               |
| Worker        | off     | shares python image   | Enable with `service:enable worker` — `celery -A saleor worker -B`   |
| Mailpit       | on      | latest                | Catches outgoing SMTP, UI at `madock proxy` mailpit URL              |
| pgAdmin       | off     | latest                | DB browser, enable with `service:enable pgadmin`                     |

## Ports

madock allocates host ports dynamically (starting from `17000`) to avoid collisions between projects. Run `madock info` or `madock info:ports` to see the current allocation.

* **API** (`python` service) — reachable via the project nginx host (`https://loc.<project>.com`). No direct host port to avoid conflicts. nginx upstream is `python:8000`.
* **Dashboard** — `http://localhost:<port/saleor_dashboard>` on the host. Internally listens on `80`.
* **PostgreSQL** — `localhost:<port/db>` for tools like psql / DBeaver.

The Saleor backend reaches PostgreSQL and Redis via the docker network (`db:5432`, `redisdb:6379`), so nothing has to be configured beyond what `madock install` writes into `.env`.

## Smart Python entrypoint

The python container does not drop into a useless bash shell. The entrypoint:

1. Sources `.env` (so DATABASE_URL, REDIS_URL, SECRET_KEY end up in process env — Saleor reads them via `os.environ`).
2. Detects whether the project has `manage.py` + `saleor.asgi:application` and prefers `uvicorn saleor.asgi:application --reload` for ASGI; otherwise falls back to `python manage.py runserver`.
3. If `node_modules`-equivalent (`.venv` or installed `saleor` module) is missing, idles with `[madock] Python deps missing — run madock install first`.
4. Picks `uv run` when `uv.lock` is present (Saleor 3.21+), falls back to plain `python` for older releases.

Works for any Saleor 3.x checkout; equally usable for non-Saleor Django apps if you point madock at them with `--platform custom --language python` (though without the Saleor-specific install flow).

## HMR / file watching on macOS

Django's `runserver --reload` polls file mtimes by default, so it's mostly fine on macOS bind mounts. uvicorn `--reload` uses `watchfiles` which honors `WATCHFILES_FORCE_POLLING=true` when needed.

For deeper inotify-like reload (e.g. with `django-extensions` `runserver_plus`), see [macos-hmr.md](macos-hmr.md).

## Common gotchas

### Saleor does not auto-load .env

Saleor reads configuration from `os.environ` via `dj_database_url` and `os.environ.get(...)`. It does NOT auto-source `.env`. `madock install` and the python container entrypoint both `set -a; . ./.env; set +a` before invoking `manage.py`. If you run a one-off command via `docker exec` without madock, source the file yourself or wrap with `env $(cat .env | xargs)`.

### Database password contains URL-special characters

`madock install` URL-encodes `db/user` and `db/password` before embedding them into `DATABASE_URL` so a `@`, `:`, `/`, `?`, or `#` in the password doesn't get misparsed by the pg client.

### `1401 unapplied migrations` after install

Django sometimes lists pending migrations on the first server boot even after `manage.py migrate` has completed — the count is computed from app config metadata, not from the migration history table. Run `madock saleor migrate` once if you want the message to clear; subsequent restarts won't complain.

### Dashboard URL

The Saleor Dashboard is a static SPA that points to a GraphQL endpoint via an env var. The dashboard service in madock defaults `API_URL` to your project nginx host's `/graphql/`. Override `saleor/dashboard/api_url` in `config.xml` to change it.

### Celery worker shares the python image

`service:enable worker` brings up a second container that uses the same image as the python service but runs `celery -A saleor --app=saleor.celeryconf:app worker --loglevel=info -B` (beat embedded). It needs the deps already installed — run `madock install` first.

## Tips

* `madock saleor migrate` — apply migrations after pulling new commits or bumping dependencies.
* `madock saleor createsuperuser` — additional admin users.
* `madock saleor populatedb --createsuperuser` — re-seed sample data; safe to re-run.
* Use `madock bash` to enter the python container as the `saleor` user (workdir `/var/www/html`).
* GraphQL endpoint: `https://loc.<project>.com/graphql/`. Schema introspection works (Saleor exposes it in DEBUG mode).
