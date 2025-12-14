# Cron

Madock provides built-in cron support for running scheduled tasks in your PHP projects.

## Commands

Enable cron:
```bash
madock cron:enable
```

Disable cron:
```bash
madock cron:disable
```

## How It Works

When cron is enabled:
1. A cron process starts inside the PHP container
2. For Magento projects, it executes `bin/magento cron:run` every minute
3. The setting persists across container restarts

## Viewing Cron Logs

### Magento 2
Check the Magento cron log:
```bash
madock cli "tail -f var/log/cron.log"
```

Check system cron log:
```bash
madock cli "tail -f var/log/system.log | grep -i cron"
```

### View container logs
```bash
madock logs php
```

## Verifying Cron Status

Check if cron jobs are running:
```bash
madock cli "php bin/magento cron:status"
```

List scheduled cron jobs:
```bash
madock cli "php bin/magento cron:run --group=default -vvv"
```

## Troubleshooting

### Cron not running
1. Verify cron is enabled: check your project's `config.xml` for `<cron><enabled>true</enabled></cron>`
2. Rebuild containers: `madock rebuild`
3. Check container logs: `madock logs php`

### Cron jobs stuck
Clear cron schedule:
```bash
madock cli "php bin/magento cron:remove"
madock cli "php bin/magento cron:install"
```

## Platform Support

| Platform | Cron Support |
|----------|--------------|
| Magento 2 | ✅ Full support |
| Shopware | ✅ Full support |
| PrestaShop | ✅ Full support |
| Shopify | ✅ Full support |
| Custom PHP | ✅ Configurable |
| PWA | ❌ Not applicable |