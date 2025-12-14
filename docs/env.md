# Environment Variables

Madock supports several environment variables that allow you to customize command behavior without modifying configuration files.

## Available Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `MADOCK_SERVICE_NAME` | Override the target container for command execution | `php` | `db`, `nginx`, `node` |
| `MADOCK_USER` | Override the user inside the container | `www-data` | `root` |
| `MADOCK_WORKDIR` | Override the working directory inside the container | `/var/www/html` | `/var/www/html/app` |
| `MADOCK_TTY_ENABLED` | Enable/disable TTY mode (useful for CI/CD pipelines) | `1` | `0` or `1` |

## Usage Examples

### Disable TTY for CI/CD pipelines
```bash
MADOCK_TTY_ENABLED="0" madock cli ls
```

### Run command as root user
```bash
MADOCK_USER="root" madock cli whoami
```

### Execute command in a different container
```bash
MADOCK_SERVICE_NAME="db" madock bash
```

### Combine multiple variables
```bash
MADOCK_USER="root" MADOCK_TTY_ENABLED="0" madock cli "php bin/magento setup:upgrade"
```

## Use Cases

### CI/CD Integration
When running Madock commands in non-interactive environments (GitHub Actions, GitLab CI, Jenkins), disable TTY:
```bash
MADOCK_TTY_ENABLED="0" madock composer install --no-interaction
```

### Debugging with root access
When you need root privileges to debug permission issues:
```bash
MADOCK_USER="root" madock bash
```