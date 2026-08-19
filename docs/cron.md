# Cron

Madock provides built-in cron support for running scheduled tasks — commands that start, do their work and exit.

For a process that has to **stay** running — a queue worker, a message consumer — see [Workers](workers.md). Scheduling `queue:work` from cron is the usual way to end up with several of them at once, each waiting on the next.

Cron runs in the project's **main service container** — `php` for a PHP project, and `nodejs`, `python`, `golang`, `ruby` or `app` for the others, following `language`. A project with `php/enabled` off still gets cron; it goes in whichever container the project is built around.

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
1. A cron process starts inside the main service container
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
- Jobs are installed/removed together with `cron:enable` and `cron:disable`, and reinstalled on every `start`, `restart` and `rebuild`
- The configuration owns this crontab. Removing every `<job>` from it removes them from the container on the next start — a job deleted from the config does not keep running
- **Use `{{workdir}}` instead of writing the application path out.** It expands to the project's `workdir` when the crontab is installed, which is `/var/www/html` on a plain checkout and `/var/www/html/current` where deployer manages releases. A job that spells the path out is correct on one kind of machine and silently wrong on the other — cron sends its output nowhere, so the job simply stops running:
  ```xml
  <apply_due>* * * * * {{workdir}}/scripts/cron/poke.sh /api/cron/apply-due &gt;&gt; /var/www/html/logs/cron.log 2&gt;&amp;1</apply_due>
  ```
  Expansion happens at install time, so `crontab -l` shows the real path. `{{workdir}}` is the only placeholder; anything else — and any secret key — is refused, and the job is left out with a warning rather than installed with the placeholder still in it
- Jobs may also be written as named entries, which is the spelling `config:set` produces:
  ```xml
  <jobs>
      <scheduler>* * * * * cd /var/www/html &amp;&amp; php artisan schedule:run</scheduler>
  </jobs>
  ```

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

Start with `madock status`, which answers both halves — whether the daemon is up, and whether it has anything to run:

```
Tools:
 Cron is running (7 jobs)
```

The count is read from the container's crontab, not from the configuration. The three other answers each mean something different:

| Line | Meaning |
|---|---|
| `Cron is running but no jobs are installed` | The daemon is up and nothing is scheduled. Nothing will fail; nothing will happen either |
| `Cron is running (installed jobs: unknown)` | The crontab could not be read. Not the same as none |
| `Cron is enabled but not running` | The configuration asks for cron and the container has none |

`madock status --json` carries the same as `cron_running`, `cron_jobs` and `cron_jobs_known`; `cron_jobs` is `-1` when unknown.

To see the installed entries themselves:

```bash
madock cli crontab -u www-data -l
```

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