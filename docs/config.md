# Project Configuration

Madock configuration can be stored either globally or within a project folder.

## Configuration Locations

### Global configuration (default)
```
~/.madock/
├── config.xml                    # Global settings
└── projects/
    └── {project_name}/
        ├── config.xml            # Project-specific settings
        └── backup/
            └── db/               # Database backups
```

### Project-local configuration
```
{project_root}/
└── .madock/
    ├── config.xml                # Project settings (version controlled)
    ├── backup/
    │   └── db/                   # Database backups
    └── docker/                   # Custom Docker overrides
```

## Setting Up Project-Local Configuration

1. Create the `.madock` folder in your project root:
```bash
mkdir -p .madock
```

2. Create or copy `config.xml` with the settings you want:
```bash
cp ~/.madock/projects/{project_name}/config.xml .madock/config.xml
```

3. Edit `.madock/config.xml` manually as needed.

> **Important**: `.madock/config.xml` is read-only for madock — CLI commands (`service:enable/disable`, `config:set`, `debug:enable/disable`, `cron:enable/disable`) always write to `~/.madock/projects/{project_name}/config.xml`. This allows `.madock/config.xml` to be safely committed to your repository without unexpected modifications on servers or CI environments.

## Benefits of Project-Local Configuration

- **Version Control**: Track configuration changes in Git without risk of automatic overwrites
- **Team Sharing**: Share consistent environment settings with team members
- **Portability**: Move project with all settings intact
- **Server Safety**: CLI commands won't modify committed config files

## Configuration Commands

List all project settings:
```bash
madock config:list
```

Set a configuration value:
```bash
madock config:set --name=php/version --value=8.2
```

Clear configuration cache:
```bash
madock config:cache:clean
```

## Configuration Inheritance

Settings are inherited in this order (later overrides earlier):
1. `~/.madock/config.xml` (global defaults)
2. `~/.madock/projects/config.xml` (global project defaults)
3. `~/.madock/projects/{project_name}/config.xml` (project settings)
4. `{project_root}/.madock/config.xml` (local project settings)

## Key Configuration Options

| Key | Description | Default |
|-----|-------------|---------|
| `platform` | Project platform (`magento2`, `shopware`, `prestashop`, `shopify`, `custom`) | `magento2` |
| `language` | Programming language for custom platform (`php`, `nodejs`, `python`, `golang`, `ruby`, `none`) | `php` |
| `timezone` | Container timezone | `Europe/Kiev` |
| `php/enabled` | Enable PHP container | `false` (set `true` by setup for PHP-based platforms) |
| `php/version` | PHP version | `8.2` |
| `php/nodejs/enabled` | Node.js inside PHP container | `false` |
| `nodejs/enabled` | Standalone Node.js container | `false` |
| `python/version` | Python version (custom platform) | `3.12` |
| `go/version` | Go version (custom platform) | `1.22` |
| `ruby/version` | Ruby version (custom platform) | `3.3` |

See also: [Scopes](./scopes.md) for managing multiple environments per project.