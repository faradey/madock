# Shopware

This guide covers working with Shopware projects in madock.

## Quick Start

```bash
# 1. Go to your project directory
cd your-shopware-project

# 2. Configure the project
madock setup
# Select: Shopware platform
# Choose PHP, MySQL versions matching your project
# Enter host (e.g.: shopware.local)

# 3. Start containers
madock start

# 4. Install dependencies
madock composer install

# 5. Add host to /etc/hosts
sudo echo "127.0.0.1 shopware.local" >> /etc/hosts
```

## Workflow

### Running Shopware Console Commands

Use `madock sw` (or `madock shopware`) to run Shopware console commands:

```bash
# Clear cache
madock sw cache:clear

# Compile theme
madock sw theme:compile

# Run migrations
madock sw database:migrate

# Reindex
madock sw dal:refresh:index

# List all commands
madock sw list
```

### Running Bin Scripts

Use `madock swbin` to run scripts from the `bin/` directory:

```bash
madock swbin console cache:clear
madock swbin build-js.sh
```

### Database Operations

```bash
# Import database
madock db:import

# Export database
madock db:export

# Open phpMyAdmin (if enabled)
madock service:enable phpmyadmin
# Then open: http://localhost:8080
```

### Frontend Development

```bash
# Enable Node.js service
madock service:enable php/nodejs

# Enter container and run build
madock bash -u www-data
bin/build-js.sh
```

### Hot Reload (Watch Mode)

Shopware provides a hot reload server for storefront development.

**Step 1:** Find your project's hot reload port:

```bash
madock info:ports
# Example output:
#   hot_reload                17015
#   hot_reload_2              17016
```

**Step 2:** Configure environment variables in `.env`:

```bash
# Internal port (inside container) - keep as 9998
STOREFRONT_PROXY_PORT=9998

# External port (on host) - use the port from Step 1
PROXY_URL=http://localhost:17015
```

**Step 3:** Rebuild containers (only needed once):

```bash
madock rebuild
```

**Step 4:** Run watch script:

```bash
madock bash -u www-data
./bin/watch-storefront.sh
```

**Step 5:** Open your storefront at the proxy URL:
- **http://localhost:17015** (use your port from Step 1) — auto-refreshes when you make changes

The hot reload ports are automatically exposed by madock. Each project gets unique ports to avoid conflicts.

**Troubleshooting hot reload:**

If hot reload doesn't work, check:
1. Ports are exposed: `docker ps` should show port mappings
2. `PROXY_URL` in `.env` matches the exposed port
3. Node.js is enabled: `madock service:enable php/nodejs`

## Troubleshooting

### File Permissions (files owned by root)

Shopware may create directories and files with root ownership during runtime operations. This commonly affects:

- `files/theme-config/`
- `var/cache/`
- `public/theme/`
- `public/bundles/`
- `public/sitemap/`
- `public/thumbnail/`

**Symptoms:**
- "Permission denied" errors when accessing directories from host
- Theme compilation fails
- Cache operations fail
- Cannot edit files created by Shopware

**Why this happens:**

Shopware uses background processes (scheduled tasks, message queue consumers) that may run as root inside the container. When these processes create files, they are owned by root.

**Auto-fix on container start:**

The Shopware PHP container has an entrypoint that automatically `chown`s the
following runtime directories to `www-data` on every start:

- `var/`
- `public/theme/`
- `public/bundles/`
- `public/sitemap/`
- `public/thumbnail/`
- `public/media/`
- `files/`
- `config/jwt/`
- `custom/plugins/`

So in most cases a regular `madock start` / `madock rebuild` already fixes
ownership. No manual flag needed.

**Manual fix:**

If root-owned files appear in a directory not covered by the auto-fix list (or
the container is already running), run:

```bash
madock rebuild --with-chown
```

**Prevention tips:**

1. Always run console commands with www-data user (this is the default for `madock sw`)
2. Avoid using `madock bash` without `-u www-data` flag for Shopware operations

### Scheduled Tasks and Message Queue

Shopware uses Symfony Messenger for background task processing.

#### Scheduled tasks

When the project has `cron/enabled=true`, madock automatically installs a
crontab entry for www-data:

```
* * * * * cd /var/www/html && php bin/console scheduled-task:run --time-limit=60
```

It coexists with any custom jobs you define via `cron/jobs/*` — `scheduled-task:run`
is appended idempotently. Disable it by setting `cron/enabled=false`.

#### Message queue consumer

Two ways to run the messenger consumer:

**1. Sidecar service (recommended for ongoing dev / prod-like setups):**

Enable in project config:

```xml
<shopware>
    <messenger>
        <enabled>true</enabled>
    </messenger>
</shopware>
```

Then rebuild:
```bash
madock rebuild
```

A `messenger` container starts alongside `php`, runs
`messenger:consume async low_priority --time-limit=3600 --memory-limit=512M`
as www-data, and respawns on the configured restart policy.

**2. Foreground consume (for one-off debugging):**

```bash
madock sw:consume                 # default: async --time-limit=3600 -vv
madock sw:consume failed          # drain the failed transport
madock sw:consume async -vv       # custom args (override defaults)
```

Runs as www-data inside the existing php container — Ctrl-C to stop.

#### Synchronous fallback

If you want to skip the queue entirely (older workflow), set in `.env`:

```
MESSENGER_TRANSPORT_DSN=sync://
```

Not recommended — it diverges from prod behaviour and can mask serialization
bugs. Prefer the sidecar service above.

### Elasticsearch/OpenSearch Connection

Ensure the search engine is enabled and configured:

```bash
# Enable OpenSearch
madock service:enable opensearch
madock rebuild

# Verify connection
madock sw es:status
```

In `.env`:
```
OPENSEARCH_URL=http://opensearch:9200
SHOPWARE_ES_ENABLED=1
SHOPWARE_ES_INDEXING_ENABLED=1
```

### Redis Configuration

```bash
# Enable Redis
madock service:enable redis
madock rebuild
```

In `.env`:
```
REDIS_URL=redis://redisdb:6379
```

## Useful Commands

| Command | Description |
|---------|-------------|
| `madock sw <command>` | Run Shopware console command |
| `madock sw:consume [args]` | Run messenger consumer in foreground (debug) |
| `madock swbin <script>` | Run bin script |
| `madock bash -u www-data` | Enter container as www-data |
| `madock rebuild --with-chown` | Rebuild and fix permissions |
| `madock composer <args>` | Run composer |
| `madock db:import <file>` | Import database |
| `madock db:export` | Export database |
| `madock logs php` | View PHP container logs |
| `madock logs messenger` | View messenger consumer logs (if enabled) |

## Configuration

### .env file

Key settings for Shopware in Docker:

```bash
APP_ENV=dev
APP_URL=https://shopware.local
DATABASE_URL=mysql://magento:magento@db:3306/magento

# Default async transport (Doctrine) — picked up by the madock messenger
# sidecar service when shopware/messenger/enabled=true, or by
# `madock sw:consume` for manual debugging.
# MESSENGER_TRANSPORT_DSN=doctrine://default
# Or RabbitMQ / Redis:
# MESSENGER_TRANSPORT_DSN=amqp://guest:guest@rabbitmq:5672/%2f/messages
# MESSENGER_TRANSPORT_DSN=redis://redisdb:6379/messages

# Search engine
OPENSEARCH_URL=http://opensearch:9200
SHOPWARE_ES_ENABLED=1
SHOPWARE_ES_INDEXING_ENABLED=1
```

### Services

Enable additional services as needed:

```bash
madock service:enable redis
madock service:enable opensearch
madock service:enable phpmyadmin
madock service:enable xdebug
```

### Permissive umask (dev default)

Madock builds containers with `umask 0002` so new files are group-writable
(`664` / `775` instead of `644` / `755`). This lets www-data and any other
user inside the container co-edit runtime files without permission churn.

Default: **on** for all platforms. Applies to PHP-FPM, interactive shells,
and non-interactive `bash -c` invocations (via `BASH_ENV`).

Disable (e.g. for prod-like servers via *madock pro*):

```xml
<permissions>
    <umask>
        <permissive>false</permissive>
    </umask>
</permissions>
```

Then rebuild containers.
