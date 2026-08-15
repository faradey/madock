# Project Configuration

Madock configuration can be stored either globally or within a project folder.

## Configuration Locations

### Global configuration (default)
```
~/.madock/
├── config.xml                    # Global settings
└── projects/
    └── {project_name}/
        ├── config.xml            # Project-specific settings
        └── backup/
            └── db/               # Database backups
```

### Project-local configuration
```
{project_root}/
└── .madock/
    ├── config.xml                # Project settings (version controlled)
    ├── backup/
    │   └── db/                   # Database backups
    └── docker/                   # Custom Docker overrides
```

## Setting Up Project-Local Configuration

1. Create the `.madock` folder in your project root:
```bash
mkdir -p .madock
```

2. Create or copy `config.xml` with the settings you want:
```bash
cp ~/.madock/projects/{project_name}/config.xml .madock/config.xml
```

3. Edit `.madock/config.xml` manually as needed.

> **Important**: `.madock/config.xml` is read-only for madock — CLI commands (`service:enable/disable`, `config:set`, `debug:enable/disable`, `cron:enable/disable`) always write to `~/.madock/projects/{project_name}/config.xml`. This allows `.madock/config.xml` to be safely committed to your repository without unexpected modifications on servers or CI environments.

## Benefits of Project-Local Configuration

- **Version Control**: Track configuration changes in Git without risk of automatic overwrites
- **Team Sharing**: Share consistent environment settings with team members
- **Portability**: Move project with all settings intact
- **Server Safety**: CLI commands won't modify committed config files

## Configuration Commands

List all project settings:
```bash
madock config:list
```

Set a configuration value:
```bash
madock config:set --name=php/version --value=8.2
```

**When a change takes effect.** The compose files and Dockerfiles are rendered
from the config on every `start`, `restart` and `rebuild`. If the render differs
from what the running containers were created from, `start` says so and recreates
them — `docker compose start` only wakes existing containers and would otherwise
keep running the old definition. A change that no template reads (an SSH host, a
cron flag) renders identically and starts the containers as they are.

Derived options cannot be set: `nodejs/major_version` is computed from
`nodejs/version` on every read, so `config:set` refuses it and names the option to
set instead.

Clear configuration cache:
```bash
madock config:cache:clean
```

## Configuration Inheritance

Settings are inherited in this order (later overrides earlier):
1. `~/.madock/config.xml` (global defaults)
2. `~/.madock/projects/config.xml` (global project defaults)
3. `~/.madock/projects/{project_name}/config.xml` (project settings)
4. `{project_root}/.madock/config.xml` (local project settings)

## Key Configuration Options

| Key | Description | Default |
|-----|-------------|---------|
| `platform` | Project platform (`magento2`, `shopware`, `prestashop`, `shopify`, `custom`) | `magento2` |
| `language` | Programming language for custom platform (`php`, `nodejs`, `python`, `golang`, `ruby`, `none`) | `php` |
| `timezone` | Container timezone | `UTC` |
| `php/enabled` | Enable PHP container | `false` (set `true` by setup for PHP-based platforms) |
| `php/version` | PHP version | `8.2` |
| `php/nodejs/enabled` | Node.js inside PHP container | `false` |
| `nodejs/enabled` | Standalone Node.js container | `false` |
| `nodejs/env` | `NODE_ENV` inside the container | `development` |
| `nodejs/script` | What the container runs — see below | *(empty: pick from `package.json`)* |
| `nodejs/script_type` | How to read `nodejs/script`: `auto`, `package`, `command` | `auto` |
| `nodejs/browser_libs` | Install the shared libraries a headless browser needs | `false` |
| `php/browser_libs` | The same for the PHP image (needs `php/nodejs/enabled`) | `false` |
| `python/version` | Python version (custom platform) | `3.12` |
| `go/version` | Go version (custom platform) | `1.22` |
| `ruby/version` | Ruby version (custom platform) | `3.3` |

## What the Node container runs

Left alone, the container picks a script out of `package.json`: `dev` when
`nodejs/env` is `development`, `start` when it is `production`.

That default is why `nodejs/env` stopped being hardcoded. For a Shopify app,
`dev` is `shopify app dev` — an interactive command that prints a verification
code and waits for someone to log in. On a server nobody does, it gives up, and
the container dies with it, because that command *is* its main process. `start`
reported success minutes earlier.

`nodejs/script` says what to run instead, and `nodejs/script_type` says how to
read it:

| `script_type` | Meaning |
|---|---|
| `auto` (default) | A name `package.json` declares is run as a script; anything else is run as a command, and madock says which it chose |
| `package` | Always a `package.json` script. A name it does not declare stops the container with an explanation instead of failing at exec |
| `command` | Always a shell command |

```bash
madock config:set --name=nodejs/script --value=docker-start
madock config:set --name=nodejs/script --value="node server.js --port 3000"
madock config:set --name=nodejs/script_type --value=command
```

A script goes through the package manager the project actually uses — yarn, pnpm
or npm, detected from the lockfile. A path to a file is a command.

## Headless browsers

`nodejs/browser_libs` (and `php/browser_libs` for the PHP image) installs the
shared libraries a headless Chromium needs. Off by default: most projects have
no browser, and the libraries are weight and surface they do not need.

The list is not written down anywhere in madock — the image asks Playwright for
it at build time (`playwright install-deps`), because the package names differ
per distribution and move between releases: `libasound2` became `libasound2t64`
in Debian trixie and Ubuntu 24.04, so a list pinned by hand breaks the build the
day the base image is bumped.

Installing them from inside a running container does not work and is not worth
attempting: madock execs as a non-root user, so Playwright shells out to `sudo`
and stops at a password prompt — and anything `apt` installs into a running
container is gone at the next rebuild.

See also: [Scopes](./scopes.md) for managing multiple environments per project.