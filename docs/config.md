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

Remove a configuration value, so whatever is underneath it applies again:
```bash
madock config:unset --name=php/version
madock config:unset -n a -n b --global
```

**Why this exists.** madock keeps two copies of a project's configuration: the
one in the project (`{project}/.madock/config.xml`, committed to your
repository) and a machine-side one under `~/.madock/projects/{name}/`, seeded
from it the first time the project is seen. Reads merge both, and the project's
copy wins — so adding or changing a setting in git reaches every machine.
**Deleting one did not.** The machine-side copy kept it, `config:set` can only
assign, and clearing the cache does not touch the file, so the only way to drop
a single key was to remove the project and set it up again. A team that retires
a setting, commits and rolls out otherwise leaves every machine living by the
old value, with nothing failing and nobody told.

`config:unset` reads the value back afterwards rather than trusting the write:
if the key is still set — because the project's own config also sets it, and
that file wins — it says so and where to look, instead of reporting a removal
that did not happen.

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

## Settings that belong to the installation, not the project

One setting is deliberately outside that chain, and outside `<scopes>`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<config>
    <allow_destructive_commands>true</allow_destructive_commands>
    <scopes>
        ...
    </scopes>
</config>
```

`allow_destructive_commands` decides whether the commands that destroy a
project's data may run on this machine:

- **`project:remove`** — the registry entry, the generated runtime, the ports,
  the volumes, the images, and a recursive delete of the project directory.
- **`project:remove --registry-only`** is covered too, with one exception: an
  entry that is not a project of its own — its source directory gone, its link
  resolving to nothing, or its path inside another registered project — may be
  removed while the setting is `false`. Such an entry owns nothing that could be
  lost, and refusing kept three of them on a production host with no way to clean
  up. Containers of that name go; volumes and images are left, so the exemption
  cannot destroy data even if the entry turns out to have some.
- **`prune --with-volumes`** — `docker compose down -v --rmi all`: the data
  volumes and the images. The database is gone and nothing brings it back.

Set it to `false` and both refuse, naming the file to edit.

**Plain `prune` is not covered**, deliberately. It is `docker compose down`:
containers and the network go, the volumes and images stay, the project
directory and its registry entry are untouched, and `madock start` puts it back.
Guarding it would obstruct without protecting — and inconsistently, since `stop`
takes the same site down with nothing in front of it.

It sits at the top level because everything under `<scopes>` is reachable from a
project: the project's own `.madock/config.xml` overrides it and `config:set`
writes it. A guard a project can switch off is not a guard, so this one is
readable only from `~/.madock/config.xml` — the file belonging to whoever
administers the installation. A copy of the key inside a project's config, or
inside `<scopes>`, is ignored.

It is a guard rail, not a security control. The file sits beside the binary and
is writable by the same user, so anyone who can run the command can also edit the
file first. It stops a mistake, not a person.

The default is `true`: on a laptop, projects are created and removed all day, and
a guard that gets in the way there is a guard people learn to switch off.
madock-pro ships it as `false`, because that edition is the one that runs on
servers — and the file still has the last word there, so a machine that needs it
does not need a different binary.

## Key Configuration Options

| Key | Description | Default |
|-----|-------------|---------|
| `platform` | Project platform (`magento2`, `shopware`, `prestashop`, `shopify`, `custom`) | `magento2` |
| `nginx/enabled` | Give the project a web server — see below | `true` |
| `language` | Programming language for custom platform (`php`, `nodejs`, `python`, `golang`, `ruby`, `none`) | `php` |
| `timezone` | Container timezone | `UTC` |
| `php/enabled` | Enable PHP container | `false` (set `true` by setup for PHP-based platforms) |
| `php/version` | PHP version | `8.2` |
| `nodejs/enabled` | Standalone Node.js container | `false` |
| `nodejs/embedded/enabled` | Node.js inside the application container — see below | `false` |
| `nodejs/env` | `NODE_ENV` inside the container | `development` |
| `nodejs/script` | What the container runs — see below | *(empty: pick from `package.json`)* |
| `nodejs/script_type` | How to read `nodejs/script`: `auto`, `package`, `command` | `auto` |
| `nodejs/browser_libs` | Install the shared libraries a headless browser needs | `false` |
| `php/browser_libs` | The same for the PHP image (needs `nodejs/embedded/enabled`) | `false` |
| `python/version` | Python version (custom platform) | `3.12` |
| `go/version` | Go version (custom platform) | `1.22` |
| `ruby/version` | Ruby version (custom platform) | `3.3` |

## Node.js in two places, and they are different questions

```bash
madock config:set --name=nodejs/enabled  --value=true   # a container of its own
madock config:set --name=nodejs/embedded/enabled --value=true   # inside the app container
madock service:enable nodejs/embedded                   # the same thing, shorter
```

- **`nodejs/enabled`** gives the project a Node container running its own
  process — a Node backend, a dev server, a worker.
- **`nodejs/embedded/enabled`** puts the Node binaries inside the *application*
  container, next to whatever language it runs. That is what a build step needs:
  grunt, webpack, vite, a `npm run build` during deployment.

Both can be true at once, and neither implies the other.

Until 3.9.8 the second was spelled `php/nodejs/enabled`, which said node was
PHP's business. It is not: a Python service with a JavaScript front end, or a Go
one with an admin panel, needs exactly the same thing, and the version was
already shared at `nodejs/version`. The rename carries itself — a migration
moves the key in the installation config, in every project's registry config, in
each project's own `.madock/config.xml`, and in any template a project copied
under `.madock/docker/`.

`php/yarn/enabled` went the same way, to `nodejs/yarn/enabled`. That one was
never declared at all: no default carried it, three platform configurators set
it, and one template read it — while a real `nodejs/yarn/enabled` sat unused in
the defaults.

**Which images offer it:** all of them — php, python, ruby and golang. Every one
of those base images is Debian or Ubuntu, so the same nodesource install works
in each.

**The old names still work.** `madock service:enable php/nodejs` is accepted,
resolves to `nodejs/embedded`, and prints the new name once so it is learned
from the command rather than from a changelog. `php/yarn` is aliased the same
way, though it was never a registered service — it worked only on the three
platforms whose configurator happened to write `php/yarn/enabled`. Both are
aliases rather than a second spelling, and both will be removed once the new
names are the ones in people's fingers.

## A project with no web server

Some projects answer no request and never will: a queue worker, a bus consumer,
the service that owns a shared database schema while its neighbours take their
own webhooks. `nginx/enabled=false` leaves the web server out completely — no
container, no vhost, no block in the shared proxy, no name in the shared
certificate, and no ports reserved.

```bash
madock config:set --name=nginx/enabled --value=false
madock rebuild
```

**Removing the project's `<hosts>` does not do this**, which is the trap worth
knowing about: the container still starts, and a project with no hosts gets
`loc.<project>.com` invented for it, so its block in the shared proxy is renamed
rather than removed and the two ports stay reserved.

To check:

```bash
madock status                                  # no nginx line
madock info:ports                              # no port for nginx
grep -c "<project-host>" <MADOCK_ROOT>/aruntime/ctx/proxy.conf   # 0
```

Varnish sits in front of nginx, so it has nothing to do on a project without
one — the `depends_on` is dropped so compose still reads the file, but enabling
both is a configuration nobody needs.

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