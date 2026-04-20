# MCP Server (AI Integration)

madock includes a built-in [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server that allows AI assistants to interact with your development environment directly.

## What is MCP?

MCP is an open standard that allows AI assistants (Claude Code, Cursor, VS Code Copilot, etc.) to use external tools. With `madock mcp`, an AI assistant can check container status, change configuration, run Magento/Composer commands, import databases, and more — all without you typing commands manually.

## Setup

### Claude Code

Add to `~/.claude/settings.json` (global) or `.claude/settings.json` (per-project):

```json
{
  "mcpServers": {
    "madock": {
      "command": "madock",
      "args": ["mcp"]
    }
  }
}
```

### Cursor

Add to `.cursor/mcp.json` in your project:

```json
{
  "mcpServers": {
    "madock": {
      "command": "madock",
      "args": ["mcp"]
    }
  }
}
```

### VS Code (GitHub Copilot)

Add to `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "madock": {
      "command": "madock",
      "args": ["mcp"]
    }
  }
}
```

### Other MCP clients

Any MCP-compatible client can connect using stdio transport:

```
madock mcp
```

## Available Tools

### Informational (read-only)

| Tool | Description |
|------|-------------|
| `madock_status` | Container status (JSON) |
| `madock_config_list` | Project configuration (JSON) |
| `madock_db_info` | Database connection info (JSON) |
| `madock_service_list` | Available services and their status (JSON) |
| `madock_scope_list` | Configuration scopes (JSON) |
| `madock_info_ports` | Exposed port mappings (JSON) |
| `madock_logs` | Container logs (optional: by service) |
| `madock_help` | Command help |

### Configuration

| Tool | Description |
|------|-------------|
| `madock_config_set` | Set a configuration option (key + value) |
| `madock_service_enable` | Enable a service (redis, elasticsearch, etc.) |
| `madock_service_disable` | Disable a service |

### Container Lifecycle

| Tool | Description |
|------|-------------|
| `madock_start` | Start containers |
| `madock_stop` | Stop containers |
| `madock_restart` | Restart containers |
| `madock_rebuild` | Rebuild containers (after config changes) |

### Development

| Tool | Description |
|------|-------------|
| `madock_composer` | Run Composer commands |
| `madock_magento` | Run Magento CLI commands |
| `madock_flush_cache` | Flush all caches |
| `madock_cron_enable` | Enable cron |
| `madock_cron_disable` | Disable cron |
| `madock_debug_enable` | Enable Xdebug |
| `madock_debug_disable` | Disable Xdebug |
| `madock_ssl_rebuild` | Rebuild SSL certificates |

### Database

| Tool | Description |
|------|-------------|
| `madock_db_import` | Import database dump |
| `madock_db_export` | Export database |
| `madock_db_execute` | Execute SQL query |

### Scopes (Multi-site)

| Tool | Description |
|------|-------------|
| `madock_scope_add` | Add a new scope |
| `madock_scope_set` | Set active scope |

### Remote Sync

| Tool | Description |
|------|-------------|
| `madock_remote_sync_db` | Sync database from remote server |
| `madock_remote_sync_media` | Sync media files from remote server |

## Resources

The MCP server also exposes a resource `madock://docs/llms.txt` containing the full madock documentation. AI assistants can read this resource to understand all available commands and configuration options.

## Usage Examples

Once configured, you can ask your AI assistant things like:

- "What's the status of my containers?"
- "Switch PHP version to 8.3"
- "Enable Elasticsearch and rebuild"
- "Import the database from dump.sql.gz"
- "Run magento setup:upgrade"
- "Show me the nginx logs"
- "Sync the database from production"
- "Add a new scope for the B2B store"

The AI assistant will use the appropriate madock tools to execute these operations.
