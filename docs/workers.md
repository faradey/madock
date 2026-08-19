# Workers

A long-running process — a queue worker, a message consumer, a job runner — as a service of the environment, on any platform.

## Worker or cron

Both run a command you configure, and there the resemblance ends. They are different mechanisms, not two spellings of one.

| | [`cron`](cron.md) | `worker` |
|---|---|---|
| The command | starts, does its work, exits | starts once and stays up |
| Where it runs | **inside the existing container**, as a crontab line | **in a container of its own**, added to the compose file |
| What runs it | the `cron` daemon in the main service | docker, from a compose service |
| If it exits | nothing happens; cron runs it again at the next tick | docker restarts it, per the project's restart policy |
| Takes effect after | `start` or `restart` — the crontab is rewritten each time | `rebuild` — the compose file is regenerated |
| Visible in `madock status` | as `Cron is running (N jobs)` | as its own service, by name |
| Its output | goes wherever the crontab line sends it, `/dev/null` by default | `madock logs -s worker-<name>` |

Use `cron` for "every five minutes, do this". Use `worker` for "keep consuming this queue". Running `queue:work` from cron is a common way to get two of them at once, both half-dead; running a nightly cleanup as a worker means writing your own sleep loop.

The two are independent — a project can have both, and most that need a worker also need cron.

## Configuration

```xml
<worker>
    <enabled>true</enabled>
    <programs>
        <queue>php artisan queue:work redis --sleep=3 --tries=3 --max-time=3600</queue>
        <reindex>node build/reindex.js</reindex>
    </programs>
</worker>
```

Each named entry becomes its own compose service, called `worker-<name>`. Start it the usual way:

```bash
madock rebuild     # the service is created with the environment
madock status      # it appears among the services
madock logs -s worker-queue
```

| Element | Required | Default | Meaning |
|---|---|---|---|
| `enabled` | yes | `false` | Turns the whole block on |
| `programs/<name>` | yes | — | The command, run through `sh -c`. The element name becomes the service name |
| `user` | no | image default | Applies to every program. Usually `www-data` on PHP images |

**XML-escape `&` as `&amp;`** inside a command, as everywhere else in this file.

## What it renders

The two programs above become two compose services. This is the real output, from the fixture that tests it:

```yaml
  worker-queue:
    init: true
    build:
      context: ctx
      dockerfile: nodejs.Dockerfile
    working_dir: /var/www/html
    entrypoint: []
    command: ["sh", "-c", "exec node build/worker.js"]
    volumes:
      - ./src:/var/www/html:cached
    extra_hosts:
      - "host.docker.internal:host-gateway"
      - "golden.test:host-gateway"
    depends_on:
      - nodejs
    restart: no
```

`dockerfile` is the main service's — `nodejs.Dockerfile` here because the project's language is Node, `php.Dockerfile` on a PHP project. `exec` in the command means the process replaces the shell, so docker signals reach it and a stop is a stop rather than a ten-second wait.

## What a worker gets

- **The main service's image.** `php` for a PHP project, `nodejs`, `python`, `golang`, `ruby`, or `app` for a language-less one — whatever the project is built around. So the worker has exactly the runtime the application has, and there is no second Dockerfile to drift out of step.
- **The project's `workdir`** as its working directory, so the command needs no absolute path. This matters on a deployer host, where the application lives under `current/` and that path changes with every release.
- **`entrypoint: []`.** The language images start the application; a worker replaces that rather than running after it.
- `depends_on` the main service, the same `extra_hosts`, the same restart policy, and the isolated network when isolation is on.

## Why a service and not a systemd unit on the host

Because a unit wrapping `madock cli <command>` looks like it works, and the ways it fails are all quiet. **Measured on a production machine on 2026-08-19:**

- the unit knew the path `/var/www/html/current/artisan`, and after the move to a deployer layout it pointed at a directory where that file does not exist;
- every rebuild kills the `docker exec` underneath it, so systemd restarts the whole chain — the restart counter had reached 74;
- `systemctl is-active` answers `active` whether or not the container is there, so the check reads the wrapper rather than the work.

A service has none of those. It is described where the rest of the environment is described, it dies and comes back with the container, and `madock status` counts it.

It also removes a subtler cost: a host-side unit keeps the madock binary open, and replacing an open binary with `cp` fails with `Text file busy`. That is a strange thing to discover while updating a server.

## When this is not enough

`worker` gives you restart-on-exit and nothing more, which is what most projects need. For several copies of one program, per-program logs, a hot reload without a rebuild, or a status command that asks the supervisor rather than docker, madock-pro has [`supervisor`](https://github.com/faradey/madock-pro) — the same idea with a process manager inside the container.

The platform-specific worker services that predate this — Shopware's `messenger`, Sylius's `messenger`, Saleor's `worker` — are unchanged and keep their own keys.
