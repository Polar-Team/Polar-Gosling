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
| `get_steering_product` | Product overview: services, key concepts, Nest repo structure, .fly file types, cloud targets |
| `get_steering_structure` | Full project structure: repository layout, directory conventions, file locations for all three repos |
| `get_steering_tech` | Tech stack: languages, frameworks, code quality tools, common commands, dependency management |
| `get_yc_grpc_overview` | Yandex Cloud gRPC API overview: common patterns, auth, async operations, pagination |
| `get_yc_services_index` | Index of all YC gRPC services with endpoints and tool names |
| `get_yc_api_gateway` | YC API Gateway gRPC API: ApiGatewayService RPCs, OpenAPI spec management |
| `get_yc_object_storage` | YC Object Storage (S3) gRPC API: BucketService, HTTPS config, inventory |
| `get_yc_ydb` | YC Managed YDB gRPC API: DatabaseService, BackupService, LocationService |
| `get_yc_serverless_containers` | YC Serverless Containers gRPC API: ContainerService, DeployRevision |
| `get_yc_compute` | YC Compute gRPC API: InstanceService, DiskService, ImageService |
| `get_yc_lockbox` | YC Lockbox gRPC API: SecretService, PayloadService |
| `get_yc_vpc` | YC VPC gRPC API: NetworkService, SubnetService, SecurityGroupService |
| `get_yc_iam` | YC IAM gRPC API: IamTokenService, ServiceAccountService, KeyService |
| `get_yc_container_registry` | YC Container Registry gRPC API: RegistryService, ImageService, ScannerService |
| `get_yc_resource_manager` | YC Resource Manager gRPC API: CloudService, FolderService |
| `get_yc_triggers` | YC Serverless Triggers gRPC API: Timer, MessageQueue, ObjectStorage, Logging, etc. |
| `get_yc_message_queue` | YC Message Queue (YMQ) SQS-compatible HTTP API: queue/message operations, Celery broker config |

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
