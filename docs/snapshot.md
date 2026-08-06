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

`project:clone` deliberately does **not** stop the source project — cloning must
not interrupt whatever the source is doing — so a clone accepts the risk that
`snapshot:create` refuses.

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

## Storage Location

Snapshots are stored next to the madock installation, under the project's
backup directory:
```
{madock_dir}/projects/{project_name}/backup/snapshot/{snapshot_name}/
```

Each snapshot is a directory holding `db.tar.gz`, `files.tar.gz` and, when the
project has a second database, `db2.tar.gz`.

## Tips

### Check available snapshots
```bash
ls {madock_dir}/projects/{project_name}/backup/snapshot/
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