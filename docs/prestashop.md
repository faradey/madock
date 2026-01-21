# PrestaShop

This guide covers working with PrestaShop projects in madock.

## Quick Start

```bash
# 1. Go to your project directory
cd your-prestashop-project

# 2. Configure the project
madock setup
# Select: PrestaShop platform
# Choose PHP, MySQL versions matching your project
# Enter host (e.g.: prestashop.local)

# 3. Start containers
madock start

# 4. Install dependencies
madock composer install

# 5. Add host to /etc/hosts
sudo echo "127.0.0.1 prestashop.local" >> /etc/hosts
```

## Workflow

### Running PrestaShop Console Commands

Use `madock ps` (or `madock prestashop`) to run PrestaShop console commands:

```bash
# Show PrestaShop info
madock ps about

# Clear cache
madock ps cache:clear

# List modules
madock ps module:list

# Enable/disable module
madock ps module:enable module_name
madock ps module:disable module_name

# Run database migrations
madock ps doctrine:migrations:migrate

# List all commands
madock ps list
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

# Enter container and run npm
madock bash
cd themes/your-theme/_dev
npm install
npm run build
```

## Troubleshooting

### File Permissions

PrestaShop may create files with incorrect permissions during cache operations or module installations.

**Symptoms:**
- "Permission denied" errors
- Cannot clear cache from admin panel
- Module installation fails

**Solution:**

```bash
madock rebuild --with-chown
```

If the problem reoccurs, run the command again.

**Directories commonly affected:**
- `var/cache/`
- `var/logs/`
- `img/`
- `upload/`
- `download/`
- `modules/`

### Cache Issues

If cache causes problems:

```bash
madock bash
rm -rf var/cache/*
exit
madock ps cache:clear
```

Or clear cache from admin panel: Advanced Parameters > Performance > Clear cache.

### Module Installation Issues

If modules fail to install due to permissions:

```bash
# Fix permissions first
madock rebuild --with-chown

# Then install module
madock ps module:install module_name
```

### Database Connection

Ensure database settings in `app/config/parameters.php`:

```php
'database_host' => 'db',
'database_port' => '3306',
'database_name' => 'magento',
'database_user' => 'magento',
'database_password' => 'magento',
```

### Redis Configuration

```bash
# Enable Redis
madock service:enable redis
madock rebuild
```

Configure in PrestaShop admin or `app/config/parameters.php`.

## Useful Commands

| Command | Description |
|---------|-------------|
| `madock ps <command>` | Run PrestaShop console command |
| `madock composer <args>` | Run composer |
| `madock bash` | Enter container as www-data |
| `madock bash -u root` | Enter container as root |
| `madock rebuild --with-chown` | Rebuild and fix permissions |
| `madock db:import <file>` | Import database |
| `madock db:export` | Export database |
| `madock logs php` | View PHP container logs |

## Common Console Commands

```bash
# Cache
madock ps cache:clear
madock ps cache:warmup

# Modules
madock ps module:list
madock ps module:enable <name>
madock ps module:disable <name>
madock ps module:install <name>
madock ps module:uninstall <name>

# Theme
madock ps theme:list
madock ps theme:enable <name>

# Database
madock ps doctrine:migrations:migrate
madock ps doctrine:schema:update --force

# Debug
madock ps debug:router
madock ps debug:container
```

## Services

Enable additional services as needed:

```bash
madock service:enable redis
madock service:enable elasticsearch
madock service:enable phpmyadmin
madock service:enable xdebug
```
