# Docker Compose Override

Madock lets you inject extra `docker-compose` configuration without touching the generated files. The override is **optional** — if you don't create it, nothing happens; if you do, it is merged on top of the generated stack.

## How it works

On every `madock start` / `madock rebuild`, Madock looks for a platform-specific override file:

```
docker-compose.<GOOS>.yml
```

where `<GOOS>` is the operating system of the host that runs Madock:

| Host OS | File name |
|---------|-----------|
| macOS   | `docker-compose.darwin.yml`  |
| Linux   | `docker-compose.linux.yml`   |
| Windows | `docker-compose.windows.yml` |

If the file exists, its contents become `docker-compose.override.yml` in the project runtime and are passed to `docker compose` via `-f`. If it does **not** exist, an empty override is generated — no error, no extra services.

## Where to put it

The file is resolved through the standard fallback chain (first found wins). The most common place is an in-project override:

```
<PROJECT_ROOT>/.madock/docker/docker-compose.darwin.yml
```

Full resolution order:

1. `<PROJECT_ROOT>/.madock/docker/docker-compose.<GOOS>.yml` — in-project override (highest priority)
2. `<MADOCK_ROOT>/projects/<PROJECT_NAME>/docker/docker-compose.<GOOS>.yml` — per-project override
3. `<MADOCK_ROOT>/docker/<PLATFORM>/docker-compose.<GOOS>.yml` — platform default
4. `<MADOCK_ROOT>/docker/languages/<LANGUAGE>/docker-compose.<GOOS>.yml` — language default
5. `<MADOCK_ROOT>/docker/general/service/docker-compose.<GOOS>.yml` — general default

See [Customizations](customizations.md) for the same fallback chain applied to all Docker config files.

## Example

`<PROJECT_ROOT>/.madock/docker/docker-compose.darwin.yml`:

```yaml
services:
  phpfpm:
    environment:
      MY_CUSTOM_VAR: "value"
    volumes:
      - ./extra:/var/www/extra
```

Apply with:

```bash
madock rebuild
```

> The standard Compose merge rules apply: maps are merged, scalars are replaced, and lists are replaced (not appended).
