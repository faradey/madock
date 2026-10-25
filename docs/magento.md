# Magento 2

This guide covers working with Magento 2 projects in madock.

## Quick Start

```bash
# 1. Go to your project directory
cd your-magento-project

# 2. Configure the project
madock setup
# Select: Magento 2 platform
# Choose PHP, MySQL, Elasticsearch versions matching your project
# Enter host (e.g.: magento.local)

# 3. The containers are already up — setup ends with a rebuild
madock status

# 4. Install dependencies
madock composer install

# 5. Add host to /etc/hosts
sudo echo "127.0.0.1 magento.local" >> /etc/hosts
```

### New Project from Scratch

```bash
# Create new Magento project with download and installation
madock setup --download --install

# Or use a preset, by its version or name; `madock matrix --platform magento2` lists them
madock setup --preset 2.4.8
```

A `--preset` that matches no preset is refused before anything is written,
with the list of presets — it never falls back to the defaults.

### Magento below 2.4.8 and Composer's security advisories

Since 2026-09-29 Composer refuses to install **every Magento release below
2.4.8**. They all require `league/flysystem` 1.x or 2.x; advisory
`PKSA-w9tt-7782-78jx` covers every flysystem up to 3.35.2, and the only fixed
release, 3.36.0, is accepted by Magento 2.4.8 and later alone. The first patch
releases of each line (2.4.4 to -p8, 2.4.5 to -p7, 2.4.6 to -p5, 2.4.7 without a
patch) are also covered by `PKSA-db8d-773v-rd1n`. Composer 2.9+ blocks such
packages on `create-project`, `install` and `update`.

An **existing** project on such a release is not affected: it starts and runs
as before; only a new download or a `composer update` stops.

To work on an older release on your own machine anyway, accept those advisories
for that project, by id:

```bash
madock setup --download --platform-version 2.4.7 \
  --allow-advisory PKSA-w9tt-7782-78jx,PKSA-db8d-773v-rd1n
```

madock downloads with `create-project --no-install`, writes the ids into the
project's `composer.json` as `audit.ignore`, and then installs, so later updates
of that project keep the decision and it stays visible in the project. Only
ids are accepted (`PKSA-…`, `CVE-…`, `GHSA-…`) — never "all advisories". Check
what was written with `madock composer config audit.ignore`.

**A project already on disk** whose `composer update` or `install` now stops:

```bash
madock composer config audit.ignore --json '["PKSA-w9tt-7782-78jx","PKSA-db8d-773v-rd1n"]'
madock composer update        # or: madock composer install
```

Commit the changed `composer.json`, so everybody working on the project has
the same decision.

**On madock older than 4.5.19**, which has no `--allow-advisory`, the same by
hand:

```bash
madock setup --platform-version 2.4.7     # without --download
madock start
madock composer create-project --no-install --repository-url=https://repo.magento.com/ \
  magento/project-community-edition:2.4.7 .
madock composer config audit.ignore --json '["PKSA-w9tt-7782-78jx","PKSA-db8d-773v-rd1n"]'
madock composer install
```

**Do not switch the blocking off as a whole** — `audit.block-insecure false`,
`COMPOSER_NO_SECURITY_BLOCKING`, `COMPOSER_NO_BLOCKING`. That lets through every
other vulnerable package as well, including ones you have never heard of. An
id accepts that one advisory and nothing else: with another id accepted the
flysystem install stays blocked (the e2e `TestAnAcceptedAdvisoryLetsComposerInstallPastIt`).

This is a decision to run a package with a known vulnerability, and it is
yours to make on your machine. **madock Pro refuses `--allow-advisory`**: it
runs servers, where that package is exactly what must stay out — upgrade to
2.4.8 or later there.

## Workflow

### Running Magento Commands

Use `madock m` (or `madock magento`) to run Magento CLI commands:

```bash
# Setup upgrade
madock m setup:upgrade

# Compile DI
madock m setup:di:compile

# Deploy static content
madock m setup:static-content:deploy -f

# Clear cache
madock m cache:flush

# Reindex
madock m indexer:reindex

# Check Magento version
madock m --version

# List all commands
madock m list
```

### Using n98-magerun

Enable and use n98-magerun for advanced operations:

```bash
# Enable n98-magerun
madock service:enable n98magerun

# Run n98 commands
madock n98 sys:info
madock n98 db:status
madock n98 cache:flush
madock n98 admin:user:list
```

### Magento Cloud

For Magento Cloud projects:

```bash
# Enable Magento Cloud CLI
madock service:enable cloud

# Run cloud commands
madock cloud project:list
madock cloud environment:list
madock cloud db:dump
```

### Running Multiple Commands

Use `madock cli` to run multiple commands at once:

```bash
madock cli "php bin/magento setup:upgrade && php bin/magento setup:di:compile && php bin/magento cache:flush"
```

### Database Operations

```bash
# Import database
madock db:import dump.sql.gz

# Export database
madock db:export

# Open phpMyAdmin (if enabled)
madock service:enable phpmyadmin
```

### Frontend Development

```bash
# Enable Node.js in PHP container
madock service:enable nodejs/embedded

# Enter container and run grunt
madock bash
npm install
grunt exec:all
grunt watch
```

See [LiveReload documentation](livereload.md) for auto-refresh setup.

## Testing

### MFTF (Magento Functional Testing Framework)

See [MFTF documentation](mftf.md) for complete setup.

```bash
# Enable MFTF
madock service:enable mftf

# Init configuration
madock mftf:init

# Generate tests
madock mftf generate:tests

# Run tests
madock mftf run:test AdminLoginSuccessfulTest -r
```

## Troubleshooting

### File Permissions

Magento generally handles permissions well since all commands run as www-data. However, if you encounter permission issues:

```bash
madock rebuild --with-chown
```

### var/cache and var/page_cache Issues

If cache directories cause problems:

```bash
madock bash
rm -rf var/cache/* var/page_cache/* generated/*
exit
madock m cache:flush
```

### Elasticsearch Connection

Ensure Elasticsearch is enabled and configured:

```bash
# Enable Elasticsearch
madock service:enable elasticsearch
madock rebuild

# Or OpenSearch
madock service:enable opensearch
madock rebuild
```

In `app/etc/env.php`:
```php
'system' => [
    'default' => [
        'catalog' => [
            'search' => [
                'engine' => 'elasticsearch7',
                'elasticsearch7_server_hostname' => 'elasticsearch',
                'elasticsearch7_server_port' => '9200'
            ]
        ]
    ]
]
```

### Redis Configuration

```bash
# Enable Redis
madock service:enable redis
madock rebuild
```

In `app/etc/env.php`:
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

### Cron Issues

```bash
# Enable cron
madock cron:enable

# Disable cron
madock cron:disable

# Check cron status
madock status
```

See [Cron documentation](cron.md) for more details.

## Useful Commands

| Command | Description |
|---------|-------------|
| `madock m <command>` | Run Magento CLI command |
| `madock n98 <command>` | Run n98-magerun command |
| `madock cloud <command>` | Run Magento Cloud CLI |
| `madock mftf <command>` | Run MFTF command |
| `madock composer <args>` | Run composer |
| `madock bash` | Enter container as www-data |
| `madock db:import <file>` | Import database |
| `madock db:export` | Export database |
| `madock logs php` | View PHP container logs |
| `madock cron:enable` | Enable cron |
| `madock cron:disable` | Disable cron |

## Services

Enable additional services as needed:

```bash
madock service:enable redis
madock service:enable elasticsearch
madock service:enable rabbitmq
madock service:enable phpmyadmin
madock service:enable xdebug
madock service:enable n98magerun
madock service:enable mftf
madock service:enable cloud
```

## Creating Patches

Create patches for composer with cweagans/composer-patches:

```bash
madock patch:create \
  --file=vendor/magento/module-analytics/Cron/CollectData.php \
  --name=collect-data-cron.patch \
  --title="Collect data cron patch"
```
