"""Polar Gosling MCP Server.

Exposes structured information about MotherGoose, UglyFox, Gosling CLI,
and the Compute Module as MCP tools for AI assistants.
"""

import json
from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent

from data.mothergoose import (
    MOTHERGOOSE_API_ENDPOINTS,
    MOTHERGOOSE_CELERY_TASKS,
    MOTHERGOOSE_MODELS,
    MOTHERGOOSE_SERVICES,
    MOTHERGOOSE_ENV_VARS,
)
from data.uglyfox import (
    UGLYFOX_CELERY_TASKS,
    UGLYFOX_CONFIG_SCHEMA,
    UGLYFOX_ENV_VARS,
)
from data.gosling import (
    GOSLING_COMMANDS,
    FLY_LANGUAGE_REFERENCE,
    FLY_EXAMPLES,
)
from data.compute_module import (
    COMPUTE_MODULE_VARIABLES,
    COMPUTE_MODULE_OUTPUTS,
    COMPUTE_MODULE_PROVIDERS,
)
from data.cross_cutting import (
    SECRET_URI_SCHEMES,
    DATABASE_SCHEMA,
    ARCHITECTURE_OVERVIEW,
)

app = Server("polar-gosling")

TOOLS: list[Tool] = [
    # MotherGoose
    Tool(
        name="get_mothergoose_api_endpoints",
        description="List all MotherGoose REST API endpoints with methods, paths, descriptions, and auth requirements.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_mothergoose_celery_tasks",
        description="List all MotherGoose Celery tasks with names, queues, priorities, and descriptions.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_mothergoose_models",
        description="Get Pydantic model schemas for Runner, EggConfig, SyncHistory, DeploymentPlan, AuditLog.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_mothergoose_services",
        description="List all MotherGoose service classes with their responsibilities and key methods.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_mothergoose_env_vars",
        description="Get all MOTHERGOOSE_* environment variable definitions with types and descriptions.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    # UglyFox
    Tool(
        name="get_uglyfox_celery_tasks",
        description="List all UglyFox Celery tasks: health checks, pruning, lifecycle management.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_uglyfox_config_schema",
        description="Get the UF/config.fly schema: pruning policies, runner conditions, Apex/Nadir pool rules.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_uglyfox_env_vars",
        description="Get all UGLYFOX_* environment variable definitions with types and descriptions.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    # Gosling CLI
    Tool(
        name="get_gosling_commands",
        description="Get all Gosling CLI commands with flags, usage patterns, and examples.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_fly_language_reference",
        description="Get the .fly language reference: block types, attributes, type system, and validation rules.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_fly_examples",
        description="Get example .fly files for egg, eggsbucket, job, uglyfox, and mothergoose blocks.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    # Compute Module
    Tool(
        name="get_compute_module_variables",
        description="Get all Terraform/OpenTofu input variables for the Compute Module (AWS + Yandex Cloud).",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_compute_module_outputs",
        description="Get Compute Module output values: hostname, public_ip, private_ip, id.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_compute_module_providers",
        description="Get provider version constraints for the Compute Module (yandex, aws, random).",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    # Cross-cutting
    Tool(
        name="get_secret_uri_schemes",
        description="Get secret URI scheme reference: yc-lockbox://, aws-sm://, vault:// formats and usage.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_database_schema",
        description="Get all YDB/DynamoDB table schemas: runners, egg_configs, sync_history, deployment_plans, audit_logs, tofu_versions, gosling_version.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_architecture_overview",
        description="Get a high-level architecture overview: system flow, services, deployment mechanisms, cloud targets.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
]

_DISPATCH: dict[str, object] = {
    "get_mothergoose_api_endpoints": MOTHERGOOSE_API_ENDPOINTS,
    "get_mothergoose_celery_tasks": MOTHERGOOSE_CELERY_TASKS,
    "get_mothergoose_models": MOTHERGOOSE_MODELS,
    "get_mothergoose_services": MOTHERGOOSE_SERVICES,
    "get_mothergoose_env_vars": MOTHERGOOSE_ENV_VARS,
    "get_uglyfox_celery_tasks": UGLYFOX_CELERY_TASKS,
    "get_uglyfox_config_schema": UGLYFOX_CONFIG_SCHEMA,
    "get_uglyfox_env_vars": UGLYFOX_ENV_VARS,
    "get_gosling_commands": GOSLING_COMMANDS,
    "get_fly_language_reference": FLY_LANGUAGE_REFERENCE,
    "get_fly_examples": FLY_EXAMPLES,
    "get_compute_module_variables": COMPUTE_MODULE_VARIABLES,
    "get_compute_module_outputs": COMPUTE_MODULE_OUTPUTS,
    "get_compute_module_providers": COMPUTE_MODULE_PROVIDERS,
    "get_secret_uri_schemes": SECRET_URI_SCHEMES,
    "get_database_schema": DATABASE_SCHEMA,
    "get_architecture_overview": ARCHITECTURE_OVERVIEW,
}


@app.list_tools()  # type: ignore[misc]
async def list_tools() -> list[Tool]:
    """Return all available tools."""
    return TOOLS


@app.call_tool()  # type: ignore[misc]
async def call_tool(name: str, arguments: dict[str, object]) -> list[TextContent]:
    """Dispatch tool calls to the appropriate data module."""
    data = _DISPATCH.get(name)
    if data is None:
        return [TextContent(type="text", text=f"Unknown tool: {name}")]
    return [TextContent(type="text", text=json.dumps(data, indent=2))]


def main() -> None:
    """Entry point — run the MCP server over stdio."""
    import asyncio

    async def _run() -> None:
        async with stdio_server() as (read_stream, write_stream):
            await app.run(read_stream, write_stream, app.create_initialization_options())

    asyncio.run(_run())


if __name__ == "__main__":
    main()
