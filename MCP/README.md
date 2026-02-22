# Polar Gosling MCP Server

An MCP (Model Context Protocol) server that exposes structured information about the Polar Gosling system to AI assistants.

## Tools

| Tool | Description |
|------|-------------|
| `get_mothergoose_api_endpoints` | All REST endpoints with methods, paths, auth |
| `get_mothergoose_celery_tasks` | All Celery tasks with queues and triggers |
| `get_mothergoose_models` | Pydantic model schemas (Runner, EggConfig, etc.) |
| `get_mothergoose_services` | Service classes and their responsibilities |
| `get_mothergoose_env_vars` | `MOTHERGOOSE_*` environment variables |
| `get_uglyfox_celery_tasks` | UglyFox health/pruning/lifecycle tasks |
| `get_uglyfox_config_schema` | `UF/config.fly` schema with examples |
| `get_uglyfox_env_vars` | `UGLYFOX_*` environment variables |
| `get_gosling_commands` | All CLI commands with flags and examples |
| `get_fly_language_reference` | .fly block types, attributes, type system |
| `get_fly_examples` | Example .fly files for all block types |
| `get_compute_module_variables` | AWS + YC Terraform input variables |
| `get_compute_module_outputs` | Module outputs (hostname, ip, id) |
| `get_compute_module_providers` | Provider version constraints |
| `get_secret_uri_schemes` | yc-lockbox://, aws-sm://, vault:// reference |
| `get_database_schema` | All YDB/DynamoDB table schemas |
| `get_architecture_overview` | System flow, services, cloud targets |

## Setup

```bash
cd MCP
uv sync
```

## Running

```bash
uv run python server.py
```

## Kiro / MCP Client Config

Add to `.kiro/settings/mcp.json`:

```json
{
  "mcpServers": {
    "polar-gosling": {
      "command": "uv",
      "args": ["run", "--project", "/path/to/Polar-Gosling/dev-new-features/MCP", "python", "server.py"],
      "disabled": false,
      "autoApprove": []
    }
  }
}
```
