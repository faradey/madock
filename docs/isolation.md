# Isolation Mode

Isolation mode prevents your project from making external network requests, keeping all traffic local.

## Use Cases

- **Security testing**: Test how your application behaves without external dependencies
- **Offline development**: Work without internet connection
- **Performance testing**: Eliminate network latency from tests
- **Payment testing**: Prevent accidental charges to real payment gateways
- **API isolation**: Ensure no external API calls during development

## Commands

Enable isolation mode:
```bash
madock service:enable isolation
```

Disable isolation mode:
```bash
madock service:disable isolation
```

After enabling/disabling, rebuild your containers:
```bash
madock rebuild
```

## What Gets Blocked

When isolation is enabled:

| Traffic Type | Status |
|--------------|--------|
| External HTTP/HTTPS requests | ❌ Blocked |
| External API calls | ❌ Blocked |
| Package downloads (composer, npm) | ❌ Blocked |
| Internal container communication | ✅ Allowed |
| Local database connections | ✅ Allowed |
| Local Redis/Elasticsearch | ✅ Allowed |

## Important Notes

### Before enabling isolation
1. Run `madock composer install` to ensure all dependencies are downloaded
2. Download any external resources your project needs

### Troubleshooting
If your application shows errors after enabling isolation, it likely depends on external services. Check:
- Third-party API integrations
- External image/CDN URLs
- Payment gateway connections
- Analytics services

### Temporary disable
If you need to install new packages:
```bash
madock service:disable isolation
madock rebuild
madock composer require vendor/package
madock service:enable isolation
madock rebuild
```

## Working with untrusted code

A shop to be cleaned of malware, a copy received from a stranger, an extension
nobody has read: code that may act against the machine it runs on. Isolation
alone covers the network. Four more things reach a project's containers by
default, and each is a way out for such code.

### What a container gets from the host by default

| What | Mounted as | Why it matters with hostile code |
|------|------------|----------------------------------|
| The project directory | read-write | Expected — it is the code being worked on |
| The host user's `~/.ssh` | read-only | Read-only still lets the code **read the private keys** |
| The host user's `~/.composer` (shared home) | read-write | `auth.json` holds repository **tokens**; the package cache is shared, so a planted package reaches every other project |
| `host.docker.internal` | name of the host | Services on the machine — including other projects' published ports — are reachable |
| Published ports (web, database, tools) | on every interface | The project is reachable from the local network |

### The settings that close them

Set them before the code is first started, then rebuild:

```bash
madock service:enable isolation                                    # internal network: no internet, no host
madock config:set --name ports/publish --value false               # no port on the host; only the madock proxy reaches the web server
madock config:set --name ssh/mount --value false                   # the host's ~/.ssh is not mounted
madock config:set --name php/composer/shared_home --value false    # a composer home of the project's own, without the shared auth.json
madock config:set --name php/composer/shared_cache --value false   # and without the shared package cache
madock rebuild
```

| Setting | Default | With the value above |
|---------|---------|----------------------|
| `isolation/enabled` | `false` | Every service sits on an internal network with no route out; the host is not reachable either |
| `ports/publish` | `true` | Nothing is published; the shared proxy reaches the project's web server by name on the `madock-proxy` network |
| `ssh/mount` | `true` | No `~/.ssh` in any application container. Git over ssh inside then needs a key of its own |
| `php/composer/shared_home` | `true` | The project gets its own composer home; the host's `auth.json` is not visible |
| `php/composer/shared_cache` | `true` | The download cache is the project's own |

Check what was rendered before running anything: the generated
`<madock installation>/aruntime/projects/<project>/docker-compose.yml` should
carry no `./ssh` line and no `ports:` block.

### What madock does not promise

- **A container is not a security boundary against code written to escape it.**
  An escape is a kernel bug, and on Linux the kernel is the machine's own. On
  macOS the containers run inside the Linux VM of Docker Desktop or OrbStack,
  which is one more wall but not one designed for this.
- **The proxy is shared.** Every project's web server sits on the `madock-proxy`
  network beside the shared proxy, so a compromised web container is on a
  network with the proxy that serves the other projects.
- **The project directory is writable**, by design: whatever the code writes
  there lands on the host disk.

For code that is really hostile, run madock inside a disposable virtual machine
(Lima, Multipass, UTM) that has no access to your home directory, apply the
settings above there too, and delete the VM when the work is done. Nothing on
your own machine — keys, tokens, other projects — is then within reach at all.
