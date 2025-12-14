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

2. Transfer configuration files from global location:
```bash
cp -r ~/.madock/projects/{project_name}/* .madock/
```

3. All future CLI configuration changes will be saved to `{project_root}/.madock/config.xml`

## Benefits of Project-Local Configuration

- **Version Control**: Track configuration changes in Git
- **Team Sharing**: Share consistent environment settings with team members
- **Portability**: Move project with all settings intact

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

See also: [Scopes](./scopes.md) for managing multiple environments per project.