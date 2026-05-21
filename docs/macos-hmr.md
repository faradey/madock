# HMR / file watching on macOS

This page applies to **any** madock project — Magento, Medusa, Shopware, custom Node/Python/Go/Ruby — running on macOS hosts. On Linux the issue does not exist; the env vars below are no-ops there.

## The problem

Docker Desktop on macOS bind-mounts host directories into Linux containers through a virtualisation layer (gRPC FUSE / VirtioFS). The layer **does not forward filesystem events** from the host's FSEvents API into the container's `inotify` subsystem. Files appear in the container with up-to-date contents, but tools subscribed to `inotify` watch handles never see change notifications.

Anything relying on filesystem events to detect edits is affected:

* `next dev`, `webpack --watch`, `vite`, `astro dev` — frontend HMR
* `nodemon`, `ts-node-dev`, `nest start --watch` — backend auto-restart
* `tsc --watch`, `tsup --watch` — incremental builds
* `gulp watch`, `grunt watch` — task runners
* Symfony / Laravel queue workers running with file-watch based reload

You edit a file, save, and **nothing rebuilds**. Hard refresh in the browser shows the old code.

## The fix: polling

Polling tells the watcher to `stat()` the watched files every N milliseconds instead of relying on kernel events. CPU cost is low (a few percent for small projects, more for monorepos with 10k+ files). It works on every OS and is harmless on Linux.

Each tool reads its own env var or config key:

| Tool                              | Env / config to enable polling                                                  |
|-----------------------------------|---------------------------------------------------------------------------------|
| Next.js / webpack 5               | `WATCHPACK_POLLING=true` (`WATCHPACK_POLLING_INTERVAL=1000` to tune)             |
| Chokidar (vite, nuxt, …)          | `CHOKIDAR_USEPOLLING=true` (`CHOKIDAR_INTERVAL=1000` to tune)                    |
| Nodemon                           | `--legacy-watch` flag, or `"legacyWatch": true` in `nodemon.json`                |
| ts-node-dev                       | `--poll` flag                                                                   |
| TypeScript `tsc --watch`          | `TSC_WATCHFILE=PriorityPollingInterval` env var                                  |
| Vite (without Chokidar wrapper)   | `server.watch.usePolling = true` in `vite.config.ts`                            |
| esbuild watch                     | No polling option — fall back to a wrapping process manager (nodemon, etc.)     |
| Gulp / Grunt                      | `gulp.watch(..., { usePolling: true })` / `options: { interval: 1000 }`         |

## Setting env vars in madock

madock containers read environment from the `environment:` section of the relevant compose snippet. Three places to set them, in order of preference:

1. **Project-local `docker-compose.override.yml`** — best for one-off project tweaks. Create `<project>/.madock/docker/snippets/docker-compose/<service>.yml` or add an override compose file. See [customizations.md](customizations.md) for path resolution.
2. **Project config (`config.xml`)** — for env values consumed by snippets via `{{{...}}}` placeholders. Useful when you maintain a fork of the snippet.
3. **`package.json` scripts** — works without touching docker. Example: `"dev": "WATCHPACK_POLLING=true next dev"`.

The bundled Medusa storefront container already sets `WATCHPACK_POLLING` and `CHOKIDAR_USEPOLLING` in its snippet — out of the box HMR works on macOS for the storefront.

For the backend (`nodejs` service running `medusa develop`, or a custom Node project), set the var in the start script:

```json
{
  "scripts": {
    "dev": "CHOKIDAR_USEPOLLING=true medusa develop"
  }
}
```

Or extend the snippet through an override.

## Verifying

Run inside the container:

```bash
docker exec -it <container> sh -c 'env | grep -E "POLLING|USEPOLLING"'
```

Then edit a watched source file on the host and watch the container logs — the build should re-run within ~1s.

If polling is on and HMR still doesn't fire, check that the file you edit is actually inside the bind mount (`docker exec <container> ls -la /path/to/file` should show the new mtime).

## Why not just use `:delegated`?

The `consistent` / `cached` / `delegated` flags control **consistency direction** (host vs. container as authority for write conflicts), not event forwarding. None of them turns inotify back on. The bundled snippets use `:cached` because source code is host-authoritative and Linux read-path is faster that way, but `:delegated` would not improve HMR.
