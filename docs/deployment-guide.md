# Deployment Guide for Existing Projects

This guide covers deploying existing Magento 2 and Shopware projects using madock on Windows.

## Prerequisites

- Docker Desktop for Windows installed and running
- WSL2 enabled (recommended for better performance)
- Git installed
- madock installed and available in PATH

## Magento 2

```bash
# 1. Clone the repository
git clone <repository-url> project-name
cd project-name

# 2. Configure the project
madock setup
# Select PHP, MySQL, Elasticsearch versions matching your project
# Enter host (e.g.: magento.local)

# 3. Start containers (with rebuild)
madock rebuild

# 4. Install dependencies
madock composer install

# 5. Import database
# Supported formats: .sql, .sql.gz, .sql.zip
madock db:import path/to/dump.sql.gz

# 6. Update Magento configuration (base URLs)
madock m setup:store-config:set --base-url="https://magento.local/"
madock m setup:store-config:set --base-url-secure="https://magento.local/"

# 7. Run migrations (if needed)
madock m setup:upgrade

# 8. Reindex and clear cache
madock m indexer:reindex
madock m cache:flush
```

## Shopware

```bash
# 1. Clone the repository
git clone <repository-url> project-name
cd project-name

# 2. Configure the project
madock setup
# Select PHP, MySQL/MariaDB versions matching your project
# Enter host (e.g.: shopware.local)

# 3. Start containers (with rebuild)
madock rebuild

# 4. Install dependencies
madock composer install

# 5. Import database
madock db:import path/to/dump.sql.gz

# 6. Update .env file
# Make sure APP_URL matches the host from setup
# Example: APP_URL=https://shopware.local

# 7. Clear cache and rebuild assets
madock bash
bin/console cache:clear
bin/console theme:compile
bin/console assets:install
exit
```

## Database Import Notes

- **Supported formats**: `.sql`, `.sql.gz`, `.sql.zip`
- madock automatically extracts archives
- The dump should contain the database structure and data
- For large databases, import may take several minutes

## Troubleshooting Permissions (Shopware)

If you encounter permission errors (especially with `var/`, `public/` folders):

### Step 1: Rebuild containers with chown

```bash
madock rebuild --with-chown
```

The `--with-chown` flag ensures proper file ownership after container starts.

### Step 2: Verify user ID inside container

```bash
madock bash
id
# Should show your UID, e.g.: uid=1000(www-data) gid=1000(www-data)
exit
```

### Step 3: Always use www-data user

When running commands inside the container, always use the default user:

```bash
# Correct - uses www-data
madock bash
bin/console cache:clear

# Incorrect - creates files as root, causes permission issues
madock bash --root
bin/console cache:clear
```

### Step 4: Fix permissions on host (Windows with WSL2)

If permissions are still broken, run from WSL2 terminal:

```bash
# Navigate to project folder
cd /mnt/c/path/to/project

# Fix ownership
sudo chown -R $(whoami):$(whoami) var public

# Fix permissions
chmod -R 775 var public
```

### Step 5: Full reset (if nothing helps)

```bash
madock stop
docker system prune -f
madock rebuild --with-chown
```

## Useful Commands

| Command | Description |
|---------|-------------|
| `madock start` | Start containers |
| `madock start --with-chown` | Start with permission fix |
| `madock stop` | Stop containers |
| `madock rebuild` | Rebuild and restart containers |
| `madock rebuild --with-chown` | Rebuild with permission fix |
| `madock bash` | Open bash in PHP container |
| `madock bash --root` | Open bash as root (use with caution) |
| `madock logs` | View container logs |
| `madock db:import <file>` | Import database |
| `madock db:export` | Export database |
| `madock composer <args>` | Run composer commands |
| `madock m <args>` | Run Magento CLI (Magento only) |

## Managing Services

madock allows you to enable/disable additional services for your project.

### Enable/Disable Services

```bash
# Enable a service (automatically rebuilds containers)
madock service:enable <service-name>

# Disable a service
madock service:disable <service-name>

# Enable multiple services at once
madock service:enable nodejs redis

# Enable globally (for all projects)
madock service:enable xdebug --global
```

### Available Services

| Service | Description |
|---------|-------------|
| `nodejs` | Separate Node.js container for frontend builds |
| `php/nodejs` | Node.js inside PHP container (for grunt, simple npm tasks) |
| `redis` | Redis cache server (container hostname: `redisdb`) |
| `rabbitmq` | RabbitMQ message broker |
| `xdebug` | PHP Xdebug extension |
| `ioncube` | IonCube loader |
| `elasticsearch` | Elasticsearch search engine |
| `opensearch` | OpenSearch search engine |
| `phpmyadmin` | phpMyAdmin database GUI |
| `ssl` | SSL/HTTPS support |
| `cron` | Cron scheduler |

**Magento-specific:**
| Service | Description |
|---------|-------------|
| `cloud` | Magento Cloud CLI |
| `n98magerun` | n98-magerun tool |
| `mftf` | Magento Functional Testing Framework |

**PWA/Shopify:**
| Service | Description |
|---------|-------------|
| `yarn` | Yarn package manager (instead of npm) |

### Examples

**Enable Node.js for frontend builds (Shopware/Magento):**
```bash
madock service:enable nodejs
```

**Enable Xdebug for debugging:**
```bash
madock service:enable xdebug
```

**Enable Redis for caching:**
```bash
madock service:enable redis
```

**Enable phpMyAdmin for database management:**
```bash
madock service:enable phpmyadmin
```

### Node.js: Container vs PHP-embedded

There are two ways to use Node.js in madock:

**1. `nodejs` — Separate container**

Best for: complex frontend builds, long-running watchers, PWA projects.

```bash
madock service:enable nodejs

# Run npm commands in nodejs container
madock node npm install
madock node npm run build
madock node npm run watch

# Enter the nodejs container
madock node bash
```

**2. `php/nodejs` — Node.js inside PHP container**

Best for: simple tasks like Magento 2 grunt compilation, quick npm scripts.
No separate container needed — runs directly in PHP container.

```bash
madock service:enable php/nodejs

# Run npm/grunt inside PHP container
madock bash
npm install
grunt exec:all
grunt watch
```

**When to use which:**
- Use `nodejs` when you need a dedicated Node.js environment or run long watchers
- Use `php/nodejs` for Magento 2 grunt tasks or when you want to keep things simple

### Using Redis

After enabling redis, configure your application to use it:

**Magento:**
Edit `app/etc/env.php`:
```php
'session' => [
    'save' => 'redis',
    'redis' => [
        'host' => 'redisdb',
        'port' => '6379',
        'database' => '0'
    ]
],
'cache' => [
    'frontend' => [
        'default' => [
            'backend' => 'Magento\\Framework\\Cache\\Backend\\Redis',
            'backend_options' => [
                'server' => 'redisdb',
                'port' => '6379',
                'database' => '1'
            ]
        ]
    ]
]
```

**Shopware:**
Add to `.env`:
```
REDIS_URL=redis://redisdb:6379
```

## Common Issues

### "Connection refused" after import

The database host in your dump may differ from madock's setup. Update the configuration:

**Magento:**
```bash
madock m setup:store-config:set --base-url="https://your-host.local/"
```

**Shopware:**
Edit `.env` file and ensure `DATABASE_URL` uses `db` as host:
```
DATABASE_URL=mysql://magento:magento@db:3306/magento
```

### Container won't start

Check if ports are already in use:
```bash
madock logs
```

### Elasticsearch/OpenSearch errors

Make sure the search engine version in `madock setup` matches your project requirements.
