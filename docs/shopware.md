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
# Check allocated port in ports.conf
cat aruntime/ports.conf | grep hot_reload
# Example output: shopware/hot_reload=17015
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

**Symptoms:**
- "Permission denied" errors when accessing directories from host
- Theme compilation fails
- Cache operations fail
- Cannot edit files created by Shopware

**Why this happens:**

Shopware uses background processes (scheduled tasks, message queue consumers) that may run as root inside the container. When these processes create files, they are owned by root.

**Solution:**

Run the following command to fix file ownership:

```bash
madock rebuild --with-chown
```

If the problem reoccurs after Shopware operations (theme changes, plugin installation, cache rebuild), simply run the command again.

**Prevention tips:**

1. Always run console commands with www-data user (this is the default for `madock sw`)
2. Avoid using `madock bash` without `-u www-data` flag for Shopware operations
3. After any admin panel operation that creates files, run `madock rebuild --with-chown` if you encounter permission issues

### Scheduled Tasks and Message Queue

Shopware uses Symfony Messenger for background task processing. In development, tasks are processed via Admin Worker (JavaScript worker in browser) or synchronously.

**For local development (recommended):**

Use synchronous processing in `.env`:
```
MESSENGER_TRANSPORT_DSN=sync://
```

This ensures all tasks are processed immediately in the HTTP request context (as www-data), avoiding permission issues.

**For async processing:**

If you need async processing, run the consumer as www-data:

```bash
madock bash -u www-data
bin/console messenger:consume async --time-limit=3600
```

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
| `madock swbin <script>` | Run bin script |
| `madock bash -u www-data` | Enter container as www-data |
| `madock rebuild --with-chown` | Rebuild and fix permissions |
| `madock composer <args>` | Run composer |
| `madock db:import <file>` | Import database |
| `madock db:export` | Export database |
| `madock logs php` | View PHP container logs |

## Configuration

### .env file

Key settings for Shopware in Docker:

```bash
APP_ENV=dev
APP_URL=https://shopware.local
DATABASE_URL=mysql://magento:magento@db:3306/magento

# Sync mode for development (recommended)
MESSENGER_TRANSPORT_DSN=sync://

# Or async with Redis
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
