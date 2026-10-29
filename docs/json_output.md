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

### db:execute

The rows of the query, as an array of objects keyed by column, in `data`.
MySQL, MariaDB and PostgreSQL; MongoDB refuses `--json`.

```bash
madock db:execute --json "SELECT entity_id, sku FROM catalog_product_entity LIMIT 2"
```

```json
{"success": true, "data": [{"entity_id":"1","sku":"24-MB01"},{"entity_id":"2","sku":"24-MB04"}]}
```

A query that fails answers `{"success": false, "error": "…"}` with the
client's own message on stderr. Until 4.5.17 the answer was the bare array.

### system:df

What fills the Docker disk, by kind and by project. Read-only.

```bash
madock system:df --json
```

```json
{
  "success": true,
  "data": {
    "images": {"count": 581, "bytes": 50510552075, "reclaimable_bytes": 38457009953},
    "volumes": {"count": 178, "bytes": 15968258935, "reclaimable_bytes": 438701176},
    "build_cache": {"count": 2503, "bytes": 60040000000, "reclaimable_bytes": 60040000000},
    "dangling_images": {"count": 410, "bytes": 34440000000, "reclaimable_bytes": 33100000000},
    "projects": [
      {"project": "shop", "registered": true, "images": 4, "image_bytes": 1150000000, "volumes": 2, "volume_bytes": 328000000}
    ]
  }
}
```

Image sizes are unique sizes: a layer shared between images is counted once.
`reclaimable_bytes` is what no container uses — for volumes that is data, not
cache.

### system:prune

Without `--yes`, the plan: what `--images` and `--build-cache` would remove.
With it, the same plan and what was removed. Behind `allow_destructive_commands`;
volumes are never removed.

```bash
madock system:prune --images --build-cache --json
```

```json
{
  "success": true,
  "data": {
    "plan": {
      "images": [{"id": "0123456789ab", "name": "madock_shop-php:latest", "bytes": 300000000}],
      "image_bytes": 11430000000,
      "build_cache_bytes": 60040000000
    },
    "removed": false,
    "images_removed": 0,
    "build_cache_removed": false
  }
}
```

### matrix

The compatibility matrix compiled into the binary. Runs outside a project.
`--platform <name>` keeps one platform; `common` is always there. The shape and
how to evaluate it are in [version_matrix.md](version_matrix.md#reading-it-from-outside-madock).

```bash
madock matrix --json --platform magento2
```

```json
{
  "success": true,
  "data": {
    "platforms": [
      {
        "name": "magento2",
        "tools": [
          {"name": "php", "rules": [{"from": "2.4.9", "value": "8.5"}], "default": null}
        ],
        "presets": [{"version": "2.4.9", "name": "Magento 2.4.9 (Latest)", "description": "Latest stable with PHP 8.5, OpenSearch 3.0.0, …"}],
        "budgets": {}
      }
    ],
    "common": {"tools": [ … ], "offers": [{"name": "php", "versions": ["8.5", "8.4"]}]}
  }
}
```

### Commands that do work: snapshot:create, snapshot:restore, project:clone

These copy volumes, stop and start containers and untar into helpers, and
every step prints as it goes — so do the docker and tar processes they start.
With `--json` that progress goes to **stderr**, and stdout carries exactly one
document, written at the end.

If the process is ended from inside a helper before the command can answer,
stdout is **empty** and the exit status is non-zero; the reason is on stderr.
An empty stdout is never a success.

The same holds for every `--json` command when madock refuses before the
command is reached — run from inside an already registered project, or on a
project that requires a newer madock: the refusal goes to stderr, stdout stays
empty, the exit status is 1.

```bash
madock snapshot:create --name=before-upgrade --json
```

```json
{
  "success": true,
  "data": {
    "name": "snapshot-before-upgrade-2026-10-15-12-00-00",
    "path": "/path/to/madock/projects/myproject/backup/snapshot/snapshot-before-upgrade-2026-10-15-12-00-00",
    "files": ["db.tar.gz", "files.tar.gz"],
    "size_bytes": 104857600
  }
}
```

The shapes below are what `data` holds; every one of them arrives inside the
`{"success": …, "data": …}` envelope.

`snapshot:list --json` answers `{"snapshots": [ … ]}` with the same object for
each, oldest first. `snapshot:restore --json` requires `--name`, because without
it the snapshot is chosen at a prompt, and answers `{"name": "…"}`.
`project:clone --json` answers `{"name": "…", "path": "…", "hosts": ["…"]}`:

```json
{
  "success": true,
  "data": {"name": "shop-update", "path": "/path/to/shop-update", "hosts": ["shop-update.test"]}
}
```

A clone that fails answers `{"success": false, "error": "…"}` with docker's own
words in the error, and takes what it had made with it: the project directory,
its runtime, its containers, volumes and networks, and the directory beside the
source when the clone created it.

### project:list — no envelope

`project:list --json` is the one exception: it prints a bare array, one object
per project, without `success` and `data`. It predates the envelope and scripts
read it as it is.

```bash
madock project:list --json | jq -r '.[].name'
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
