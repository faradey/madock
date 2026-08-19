# JSON Output

Some commands support JSON output format for easy integration with scripts, APIs, and external tools.

## Usage

Add `--json` or `-j` flag to supported commands:

```bash
madock status --json
madock config:list -j
```

## Response Format

All JSON responses follow a consistent structure:

**Success:**
```json
{
  "success": true,
  "data": { ... }
}
```

**Error:**
```json
{
  "success": false,
  "error": "Error message"
}
```

## Supported Commands

### status

Shows container states, proxy status, and tools status.

```bash
madock status --json
```

**Response:**
```json
{
  "success": true,
  "data": {
    "services": [
      {"name": "php-container", "service": "php", "state": "running", "running": true},
      {"name": "nginx-container", "service": "nginx", "state": "running", "running": true},
      {"name": "db-container", "service": "db", "state": "exited", "running": false}
    ],
    "proxy": [
      {"name": "proxy-container", "service": "proxy", "state": "running", "running": true}
    ],
    "tools": {
      "cron_enabled": false,
      "cron_running": false,
      "cron_jobs": -1,
      "cron_jobs_known": false,
      "debugger_enabled": true
    }
  }
}
```

`cron_enabled` is what the configuration asks for and `cron_running` is what the container has; they can disagree. `cron_jobs` counts the entries actually installed in the crontab — a running daemon with an empty one is a scheduler that schedules nothing — and is `-1` with `cron_jobs_known: false` when the container could not be asked, which is not the same as none.

### config:list

Shows all project configuration parameters.

```bash
madock config:list --json
```

**Response:**
```json
{
  "success": true,
  "data": {
    "project": "myproject",
    "config": {
      "platform": "magento2",
      "php/version": "8.2",
      "db/version": "10.6",
      "nginx/hosts/base/name": "myproject.test"
    }
  }
}
```

### scope:list

Shows all available scopes with active scope marker.

```bash
madock scope:list --json
```

**Response:**
```json
{
  "success": true,
  "data": {
    "scopes": [
      {"name": "default", "active": true},
      {"name": "staging", "active": false}
    ],
    "active": "default"
  }
}
```

### service:list

Shows all services with their enabled/disabled status.

```bash
madock service:list --json
```

**Response:**
```json
{
  "success": true,
  "data": {
    "services": [
      {"name": "elasticsearch", "enabled": true},
      {"name": "redis", "enabled": true},
      {"name": "rabbitmq", "enabled": false},
      {"name": "xdebug", "enabled": false}
    ]
  }
}
```

### db:info

Shows database connection details.

```bash
madock db:info --json
```

**Response:**
```json
{
  "success": true,
  "data": {
    "databases": [
      {
        "name": "First DB",
        "host": "db",
        "database": "magento",
        "user": "magento",
        "password": "magento",
        "root_password": "root",
        "remote_host": "localhost",
        "remote_port": 33060
      },
      {
        "name": "Second DB",
        "host": "db2",
        "database": "magento",
        "user": "magento",
        "password": "magento",
        "root_password": "root",
        "remote_host": "localhost",
        "remote_port": 33061
      }
    ]
  }
}
```

### db:export

Exports the database and returns the path of the created dump file. The dump is produced inside the database container, so no PHP or database client is required on the host.

```bash
madock db:export --json
```

**Response:**
```json
{
  "success": true,
  "data": {
    "file": "/path/to/madock/projects/myproject/backup/db/local_2026-06-12_13-38-56.sql.gz"
  }
}
```

## Exit codes

**Zero means the question was answered, not that the answer was cheerful.** `status` on a project whose containers are all stopped — or that was never started — prints what it found and exits 0: "nothing is running" is a true answer to "what is running", and a script that treats it as a failure cannot tell a stopped project from a broken one.

A non-zero exit means the question could not be answered. `status` ends non-zero when docker itself cannot be asked, and says which command failed and what it printed.

Read the state from the payload rather than from the exit code:

```bash
madock status --json | jq -e '.data.services | length > 0 and all(.[]; .running)'
```

`cron_jobs` follows the same rule in the other direction: it is `-1`, with `cron_jobs_known` false, when the crontab could not be read. Zero jobs and unknown jobs are different answers, and only one of them is a problem you can act on.

## Examples

### Get database password with jq

```bash
madock db:info --json | jq -r '.data.databases[0].password'
```

### Check if container is running

```bash
madock status --json | jq -r '.data.services[] | select(.service == "php") | .running'
```

### Get PHP version from config

```bash
madock config:list --json | jq -r '.data.config["php/version"]'
```

### List enabled services

```bash
madock service:list --json | jq -r '.data.services[] | select(.enabled == true) | .name'
```
