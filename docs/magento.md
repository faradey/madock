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

# 3. Start containers
madock start

# 4. Install dependencies
madock composer install

# 5. Add host to /etc/hosts
sudo echo "127.0.0.1 magento.local" >> /etc/hosts
```

### New Project from Scratch

```bash
# Create new Magento project with download and installation
madock setup --download --install

# Or use a preset
madock setup --preset magento-247
```

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
madock service:enable php/nodejs

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
