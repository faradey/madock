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

## What's Included in a Snapshot

| Component | Included |
|-----------|----------|
| Database dump | ✅ |
| Project files | ✅ |
| Media files | ✅ |
| Vendor folder | ❌ (run `composer install` after restore) |
| Generated files | ❌ |

## Storage Location

Snapshots are stored in:
```
~/.madock/projects/{project_name}/snapshots/
```

Or if using project-local configuration:
```
{project_root}/.madock/snapshots/
```

## Tips

### Check available snapshots
```bash
ls ~/.madock/projects/{project_name}/snapshots/
```

### Disk space
Snapshots can be large. Monitor disk usage and delete old snapshots when no longer needed.

### After restore
After restoring a snapshot, you may need to:
1. Run `madock composer install` to restore vendor dependencies
2. Run `madock rebuild` if configuration changed
3. Clear caches: `madock m cache:clean`