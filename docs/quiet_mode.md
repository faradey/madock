# Quiet Mode

The `--quiet` / `-q` flag suppresses Docker build and pull output. Useful in IDEs (JediTerm, VS Code terminal) where streaming Docker logs floods the output panel.

## Usage

Add `--quiet` or `-q` to any command that triggers Docker operations:

```bash
madock start -q
madock rebuild --quiet
madock setup -q
madock debug:enable -q
madock debug:disable --quiet
```

## What is suppressed

- `docker compose pull` — image pull progress
- `docker compose up --build` — build output for all services
- `docker compose down` / `kill` / `stop` / `start` — container lifecycle output
- Proxy container operations (nginx)

## What is NOT suppressed

- Madock's own status messages and spinners
- Errors — if a Docker command fails, the error is still surfaced via the logger
- `madock logs` — intentionally always shows output
- Commands that run inside containers (`bash`, `composer`, `magento`, etc.)

## Examples

```bash
# Rebuild without Docker flood
madock rebuild -q

# Toggle xdebug silently
madock debug:enable -q
madock debug:disable -q

# Full setup without pull/build output
madock setup --quiet
```
