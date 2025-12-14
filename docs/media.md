# Media Synchronization

With `madock` you can sync media files between your dev site and your local site.

## Configuration

For synchronization, you must specify the SSH connection data in the project settings file:
- `madock/projects/{project_name}/config.xml` or
- `{project_root}/.madock/config.xml`

See [SSH configuration examples](./ssh_example.md) for details.

## Commands

Sync all media files:
```
madock remote:sync:media
```

Sync only images with compression (reduces file size to ~30% of original):
```
madock remote:sync:media --images-only --compress
```

### Available options

| Option | Short | Description |
|--------|-------|-------------|
| `--images-only` | `-i` | Synchronize images only |
| `--compress` | `-c` | Apply lossy compression to images |
| `--ssh-type` | `-s` | SSH type (dev, stage, prod) |

## Example

Sync images from dev environment:
```
madock remote:sync:media --images-only --ssh-type dev
```
