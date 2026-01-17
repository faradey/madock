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
      "debugger_enabled": true
    }
  }
}
```

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
