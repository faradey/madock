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
2. Custom cron jobs from configuration are installed (if defined)
3. Platform-specific cron jobs are installed automatically:
   - **Magento 2**: runs `bin/magento cron:install` (installs Magento's built-in cron)
   - **Shopify**: installs Laravel scheduler cron job automatically
4. The setting persists across container restarts

## Custom Cron Jobs

You can define custom cron jobs in your project's `config.xml`. These jobs will be installed automatically when cron is enabled and removed when disabled.

### Configuration

Add jobs to the `<cron>` section in your config:

```xml
<cron>
    <enabled>false</enabled>
    <jobs>
        <job>* * * * * cd /var/www/html &amp;&amp; php bin/console scheduled:run</job>
        <job>*/5 * * * * cd /var/www/html &amp;&amp; php artisan schedule:run</job>
        <job>0 * * * * cd /var/www/html &amp;&amp; php bin/console cache:clear</job>
    </jobs>
</cron>
```

### Important Notes

- **XML escaping**: Use `&amp;` instead of `&` in commands (e.g., `cmd1 &amp;&amp; cmd2`)
- Jobs run as the `www-data` user inside the container
- Each `<job>` element should contain a complete cron entry (schedule + command)
- Jobs are installed/removed together with `cron:enable` and `cron:disable`

### Cron Schedule Format

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sunday=0)
│ │ │ │ │
* * * * * command
```

### Example Jobs by Platform

**Shopware:**
```xml
<job>* * * * * cd /var/www/html &amp;&amp; php bin/console scheduled-task:run</job>
<job>* * * * * cd /var/www/html &amp;&amp; php bin/console messenger:consume</job>
```

**Laravel/Shopify:**
```xml
<job>* * * * * cd /var/www/html &amp;&amp; php artisan schedule:run</job>
```

**Symfony:**
```xml
<job>* * * * * cd /var/www/html &amp;&amp; php bin/console messenger:consume async</job>
```

**PrestaShop:**
```xml
<job>*/15 * * * * cd /var/www/html &amp;&amp; php bin/console prestashop:update:configuration</job>
```

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