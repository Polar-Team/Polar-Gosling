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
from data.kiro import (
    STEERING_PRODUCT,
    STEERING_STRUCTURE,
    STEERING_TECH,
)
from data.yandex_cloud import (
    YC_GRPC_API_OVERVIEW,
    YC_API_GATEWAY,
    YC_OBJECT_STORAGE,
    YC_YDB,
    YC_SERVERLESS_CONTAINERS,
    YC_COMPUTE,
    YC_LOCKBOX,
    YC_VPC,
    YC_IAM,
    YC_CONTAINER_REGISTRY,
    YC_RESOURCE_MANAGER,
    YC_TRIGGERS,
    YC_MESSAGE_QUEUE,
    YC_SERVICES_INDEX,
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
        description="Get all YDB/DynamoDB table schemas: runners, egg_configs, sync_history, deployment_plans, audit_logs, binary_versions.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_architecture_overview",
        description="Get a high-level architecture overview: system flow, services, deployment mechanisms, cloud targets.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    # Kiro steering
    Tool(
        name="get_steering_product",
        description="Get the product overview: services, key concepts, Nest repo structure, .fly file types, and cloud targets.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_steering_structure",
        description="Get the full project structure: repository layout, directory conventions, and file locations for all three repos.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_steering_tech",
        description="Get the tech stack: languages, frameworks, code quality tools, common commands, and dependency management.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    # Yandex Cloud gRPC API
    Tool(
        name="get_yc_grpc_overview",
        description="Get Yandex Cloud gRPC API overview: common patterns, auth, async operations, pagination, protobuf repo link.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_services_index",
        description="List all Yandex Cloud gRPC API services covered by this module with endpoints and tool names.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_api_gateway",
        description="Get Yandex Cloud API Gateway gRPC API: ApiGatewayService RPCs, messages, OpenAPI spec management.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_object_storage",
        description="Get Yandex Cloud Object Storage (S3) gRPC API: BucketService RPCs, HTTPS config, inventory, S3-compatible REST reference.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_ydb",
        description="Get Yandex Cloud Managed YDB gRPC API: DatabaseService RPCs (CRUD, Start/Stop, Backup/Restore), BackupService, LocationService.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_serverless_containers",
        description="Get Yandex Cloud Serverless Containers gRPC API: ContainerService RPCs, DeployRevision, revision management.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_compute",
        description="Get Yandex Cloud Compute gRPC API: InstanceService, DiskService, ImageService RPCs for VM lifecycle management.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_lockbox",
        description="Get Yandex Cloud Lockbox gRPC API: SecretService and PayloadService RPCs for secret management and retrieval.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_vpc",
        description="Get Yandex Cloud VPC gRPC API: NetworkService, SubnetService, SecurityGroupService, RouteTableService, GatewayService.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_iam",
        description="Get Yandex Cloud IAM gRPC API: IamTokenService, ServiceAccountService, KeyService, ApiKeyService.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_container_registry",
        description="Get Yandex Cloud Container Registry gRPC API: RegistryService, RepositoryService, ImageService, ScannerService.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_resource_manager",
        description="Get Yandex Cloud Resource Manager gRPC API: CloudService and FolderService for managing the resource hierarchy.",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_triggers",
        description="Get Yandex Cloud Serverless Triggers gRPC API: TriggerService RPCs, all trigger types (Timer, MessageQueue, ObjectStorage, ContainerRegistry, Logging, DataStream, etc.).",
        inputSchema={"type": "object", "properties": {}, "required": []},
    ),
    Tool(
        name="get_yc_message_queue",
        description="Get Yandex Message Queue (YMQ) API reference: SQS-compatible HTTP API for queue and message operations, Celery broker config, SDK usage.",
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
    "get_steering_product": STEERING_PRODUCT,
    "get_steering_structure": STEERING_STRUCTURE,
    "get_steering_tech": STEERING_TECH,
    "get_yc_grpc_overview": YC_GRPC_API_OVERVIEW,
    "get_yc_services_index": YC_SERVICES_INDEX,
    "get_yc_api_gateway": YC_API_GATEWAY,
    "get_yc_object_storage": YC_OBJECT_STORAGE,
    "get_yc_ydb": YC_YDB,
    "get_yc_serverless_containers": YC_SERVERLESS_CONTAINERS,
    "get_yc_compute": YC_COMPUTE,
    "get_yc_lockbox": YC_LOCKBOX,
    "get_yc_vpc": YC_VPC,
    "get_yc_iam": YC_IAM,
    "get_yc_container_registry": YC_CONTAINER_REGISTRY,
    "get_yc_resource_manager": YC_RESOURCE_MANAGER,
    "get_yc_triggers": YC_TRIGGERS,
    "get_yc_message_queue": YC_MESSAGE_QUEUE,
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
