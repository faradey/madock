A Snapshot is a feature that allows for quickly creating snapshots of a project and saving them for future restoration purposes. Snapshots might be needed after unsuccessful Magento updates or other failures. A snapshot contains a copy of the database and project files.

To create a snapshot, execute the command:
```
madock snapshot:create
```

To restore a project from a snapshot, execute the command:
```
madock snapshot:restore
```