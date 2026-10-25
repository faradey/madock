# Snapshots

Snapshots allow you to quickly save and restore the complete state of your project, including database and files.

## Use Cases

- **Before major updates**: Create a snapshot before upgrading Magento, Shopware, or other platforms
- **Testing**: Save state before testing destructive operations
- **Quick rollback**: Restore to a known working state after failures
- **Environment cloning**: Duplicate project state for testing

## Commands

### Create a Snapshot

Create a snapshot with auto-generated name:
```bash
madock snapshot:create
```

Create a snapshot with a custom name:
```bash
madock snapshot:create --name=before-upgrade-247
```

### Restore from Snapshot

Restore the project from a snapshot:
```bash
madock snapshot:restore
```

You will be prompted to select which snapshot to restore if multiple exist.
`--name` skips the prompt: an exact snapshot directory name, or the name given
at creation, in which case the newest snapshot carrying it is restored.

`snapshot:restore` is refused on a project managed by deployer, and exits 1.

### List snapshots

```bash
madock snapshot:list
```

Oldest first, by the timestamp at the end of each name.

### Remove a snapshot

```bash
madock snapshot:remove --name=before-upgrade-247
```

The name is matched as `snapshot:restore --name` matches it. With a snapshot
engine, the engine's snapshot is dropped first and the directory after; if the
engine refuses (a clone still lives on it), both stay. Deleting the directory by
hand would leave the engine's snapshot holding its space.

`snapshot/keep` does the same automatically: after each `snapshot:create`, the
oldest snapshots are removed until that many remain (`0`, the default, keeps
all).

### Clone from a snapshot

```bash
madock project:clone --name=shop-demo --domain-suffix=-demo --from=golden
```

See [project_clone.md](project_clone.md).

### JSON

`snapshot:create`, `snapshot:list`, `snapshot:remove` and `snapshot:restore` answer `--json` (`-j`)
with one document on stdout — `{"success": true, "data": …}` or
`{"success": false, "error": "…"}` — and the exit status agrees with it. The
progress the commands print goes to stderr while they run. `snapshot:restore
--json` requires `--name`, because without it the snapshot is chosen at a
prompt.

## The project stands still while the snapshot is taken

`snapshot:create` **stops the project's containers**, copies from a helper
container that mounts the same volumes, and starts them again. If the project
was already stopped it is left stopped.

This is not an optimisation detail — it is what makes the snapshot restorable.
A database's data directory is a set of pages that only mean anything together.
Copied out of a running server it comes out torn: an archive that looks like a
backup and may refuse to start. There is no way to avoid that from inside the
running container, so the copy is taken with the server quiet.

The containers are **stopped, not removed**, so coming back is a start and not
a rebuild. On a small project the whole pause is a few seconds; on a large one
it is however long the data directory takes to read.

If you need a database copy *without* pausing anything, use `madock db:export`
— a logical dump taken by the server itself, which is consistent by other
means. It is not interchangeable with a snapshot: it holds one schema, not the
files.

`project:clone` takes its copy the same way and stops the source for it. It used
to copy a running source on the grounds that cloning should not interrupt it,
and the archive came out torn for exactly the reason above.

## What's Included in a Snapshot

| Component | Included |
|-----------|----------|
| Database data directory (`db`, and `db2` when enabled) | ✅ |
| Everything under the project directory | ✅ |
| Media files | ✅ |
| Vendor folder | ✅ — it is inside the project directory |
| Generated files | ✅ — same |

The files half is the project directory as the containers see it, whole. Nothing
is filtered out of it, so a snapshot of a project with a large `vendor/` or
`generated/` is correspondingly large.

A project that consumes another project's shared database has **no data
directory of its own**. Its snapshot holds the files only, and the command says
so and names the provider. Snapshot the provider from the provider: its data
directory holds every consumer, and restoring a copy taken from one would
overwrite all of them.

### The passwords the data holds

A database or broker takes its password from the configuration once, when its
volume is first created; from then on the volume holds it. A snapshot records
the passwords current when it was taken (`db/root_password`, `db/password`,
`db2/root_password`, `db2/password`, `rabbitmq/password`) in `credentials.xml`
beside its data. `snapshot:restore` and `project:clone --from` put them back
into the project's configuration, so a project restored or cloned from a
snapshot older than a password change can still log in to its own database.
Your application's own settings (`app/etc/env.php` and the like) come back with
the files of the snapshot. A snapshot taken before madock 4.5.22 has no record;
restoring it leaves the configuration as it is.

## Storage Location

Snapshots are stored next to the madock installation, under the project's
backup directory:
```
{madock_dir}/projects/{project_name}/backup/snapshot/{snapshot_name}/
```

Each snapshot is a directory holding `db.tar.gz`, `files.tar.gz` and, when the
project has a second database, `db2.tar.gz`. Every other named volume of the
project has its own `vol-<name>.tar.gz`:

| Volume | When |
|---|---|
| `elasticsearch_vlm_<version>`, `opensearch_vlm_<version>`, `meilisearch_vlm_<version>` | the search engine is enabled |
| `grafanadata`, `lokidata` | `grafana/enabled` |
| `redisdata`, `valkeydata`, `rabbitmqdata` | the service is enabled **and** `<service>/persist` is `true` |

Redis, Valkey and RabbitMQ have no named volume by default (`persist` is
`false`): their data lives in the image's anonymous volume, survives
`stop`/`start`, and is in no snapshot or clone. Set `<service>/persist` to `true`
for it to be carried.

A snapshot taken before a volume was carried has no archive for it, and a restore
leaves that volume as it is — saying so. With a snapshot engine
(`snapshot/engine`), the engine takes the databases; the other volumes still go
through tar unless the engine takes them too, and the project is then stopped for
the copy, since an index copied out of a running server is torn.

## Tips

### Check available snapshots
```bash
madock snapshot:list
```

### Disk space
Snapshots can be large — they contain the whole project directory and the whole
data directory, uncompressed on the way in. Monitor disk usage and delete old
snapshots when no longer needed.

### After restore
`snapshot:restore` rebuilds the containers itself. `vendor/` came back with the
files, so no `composer install` is needed unless the restored state predates a
dependency change. Clearing the platform cache (`madock m cache:clean`) is worth
doing when the restored state is much older than the code.