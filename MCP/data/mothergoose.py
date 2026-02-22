"""MotherGoose service data for the MCP server."""

from typing import Any

MOTHERGOOSE_API_ENDPOINTS: list[dict[str, Any]] = [
    # Health
    {"method": "GET", "path": "/health", "description": "Service health check", "auth": "none", "response": "200 OK with status"},
    # Eggs
    {"method": "GET", "path": "/eggs", "description": "List all EggConfigs", "auth": "bearer", "response": "list[EggConfig]"},
    {"method": "GET", "path": "/eggs/{egg_name}", "description": "Get a single EggConfig by name", "auth": "bearer", "response": "EggConfig"},
    {"method": "POST", "path": "/eggs/{egg_name}/deploy", "description": "Trigger runner deployment for an egg", "auth": "bearer", "response": "202 Accepted"},
    # Runners
    {"method": "GET", "path": "/runners", "description": "List all runners, optional ?state= filter", "auth": "bearer", "response": "list[Runner]"},
    {"method": "GET", "path": "/runners/{runner_id}", "description": "Get a single runner by ID", "auth": "bearer", "response": "Runner"},
    {"method": "DELETE", "path": "/runners/{runner_id}", "description": "Terminate a runner", "auth": "bearer", "response": "202 Accepted"},
    # Webhooks
    {
        "method": "POST",
        "path": "/webhooks/gitlab",
        "description": "Receive GitLab push/pipeline webhooks. Validates X-Gitlab-Token header, matches egg, enqueues Celery task.",
        "auth": "X-Gitlab-Token header (per-egg secret)",
        "response": "202 Accepted",
    },
    # Internal
    {
        "method": "POST",
        "path": "/internal/sync-git",
        "description": "Trigger a Nest Git sync. Called by YC Timer or AWS EventBridge every 5 minutes, or on Nest push webhook.",
        "auth": "internal secret token",
        "response": "202 Accepted",
    },
    {
        "method": "POST",
        "path": "/internal/health-check",
        "description": "Internal health check endpoint for cloud load balancers.",
        "auth": "none",
        "response": "200 OK",
    },
    # Binaries
    {"method": "GET", "path": "/admin/binaries/tofu", "description": "List available OpenTofu binary versions", "auth": "bearer", "response": "list[TofuVersion]"},
    {"method": "POST", "path": "/admin/binaries/tofu", "description": "Download and register a new OpenTofu binary version", "auth": "bearer", "response": "TofuVersion"},
    {"method": "PUT", "path": "/admin/binaries/tofu/{version_id}/activate", "description": "Activate a specific OpenTofu binary version", "auth": "bearer", "response": "200 OK"},
    {"method": "GET", "path": "/admin/binaries/gosling", "description": "List available Gosling binary versions", "auth": "bearer", "response": "list[GoslingVersion]"},
    {"method": "POST", "path": "/admin/binaries/gosling", "description": "Download and register a new Gosling binary version", "auth": "bearer", "response": "GoslingVersion"},
    {"method": "PUT", "path": "/admin/binaries/gosling/{version_id}/activate", "description": "Activate a specific Gosling binary version", "auth": "bearer", "response": "200 OK"},
]

MOTHERGOOSE_CELERY_TASKS: list[dict[str, Any]] = [
    {
        "name": "sync_nest_config",
        "module": "tasks.git_sync",
        "queue": "mothergoose.git_sync",
        "description": "Clone/pull the Nest repo, parse all .fly files via Gosling CLI, upsert egg_configs/jobs/uf_config in DB, write sync_history record.",
        "triggered_by": ["POST /internal/sync-git", "YC Timer (5 min)", "AWS EventBridge (5 min)", "Nest push webhook"],
    },
    {
        "name": "process_webhook",
        "module": "tasks.webhooks",
        "queue": "mothergoose.webhooks",
        "description": "Handle a GitLab push/pipeline webhook: fetch EggConfig, resolve secrets, render Jinja2 templates, run tofu plan+apply, upsert Runner, write audit_log.",
        "triggered_by": ["POST /webhooks/gitlab"],
    },
    {
        "name": "deploy_runner",
        "module": "tasks.runners",
        "queue": "mothergoose.runners",
        "description": "Deploy a new runner for an egg using OpenTofu. Determines runner type (SERVERLESS/APEX/NADIR), renders templates, applies plan, stores DeploymentPlan.",
        "triggered_by": ["POST /eggs/{egg_name}/deploy", "process_webhook"],
    },
    {
        "name": "terminate_runner",
        "module": "tasks.runners",
        "queue": "mothergoose.runners",
        "description": "Terminate a runner via OpenTofu destroy. Updates runner state to TERMINATED, writes audit_log.",
        "triggered_by": ["DELETE /runners/{runner_id}", "UglyFox lifecycle tasks"],
    },
    {
        "name": "deploy_serverless_runner",
        "module": "tasks.runners",
        "queue": "mothergoose.runners",
        "description": "Deploy a YC Serverless Container or AWS ECS Fargate runner. Handles container image, resource limits, and cloud-specific config.",
        "triggered_by": ["deploy_runner (when type=SERVERLESS)"],
    },
    {
        "name": "cleanup_serverless_runner",
        "module": "tasks.runners",
        "queue": "mothergoose.runners",
        "description": "Clean up a serverless runner after job completion. Removes container, updates state.",
        "triggered_by": ["enforce_serverless_timeout", "UglyFox pruning"],
    },
    {
        "name": "enforce_serverless_timeout",
        "module": "tasks.maintenance",
        "queue": "mothergoose.maintenance",
        "description": "Periodic task that finds serverless runners exceeding max runtime and triggers cleanup.",
        "triggered_by": ["Celery beat schedule"],
    },
    {
        "name": "cleanup_old_results",
        "module": "tasks.maintenance",
        "queue": "mothergoose.maintenance",
        "description": "Prune old Celery task results and expired sync_history records from the database.",
        "triggered_by": ["Celery beat schedule"],
    },
    {
        "name": "update_metrics",
        "module": "tasks.maintenance",
        "queue": "mothergoose.maintenance",
        "description": "Collect and publish runner/egg metrics to cloud monitoring (YC Monitoring / CloudWatch).",
        "triggered_by": ["Celery beat schedule"],
    },
]

MOTHERGOOSE_MODELS: dict[str, Any] = {
    "Runner": {
        "description": "Represents a deployed GitLab runner (VM or serverless container).",
        "fields": {
            "id": {"type": "str", "description": "Unique runner ID (UUID)"},
            "egg_name": {"type": "str", "description": "Name of the Egg that owns this runner"},
            "type": {"type": "RunnerType", "enum": ["SERVERLESS", "APEX", "NADIR"], "description": "Runner deployment type"},
            "state": {"type": "RunnerState", "enum": ["ACTIVE", "IDLE", "BUSY", "FAILED", "TERMINATED"], "description": "Current runner state"},
            "cloud_provider": {"type": "CloudProvider", "enum": ["YANDEX", "AWS"]},
            "region": {"type": "str", "description": "Cloud region (e.g. ru-central1, us-east-1)"},
            "gitlab_runner_id": {"type": "int", "description": "GitLab runner registration ID"},
            "deployed_from_commit": {"type": "str", "description": "Nest repo commit SHA used for deployment"},
            "created_at": {"type": "datetime"},
            "updated_at": {"type": "datetime"},
            "last_heartbeat": {"type": "datetime", "description": "Last time UglyFox confirmed runner is alive"},
            "failure_count": {"type": "int", "description": "Consecutive failure count, used by UglyFox pruning"},
            "metadata": {"type": "dict", "description": "Cloud-specific metadata (instance ID, container ID, etc.)"},
        },
    },
    "EggConfig": {
        "description": "Parsed configuration for a single Egg or EggsBucket from the Nest repo.",
        "fields": {
            "id": {"type": "str", "description": "UUID"},
            "name": {"type": "str", "description": "Egg name (directory name under Eggs/)"},
            "project_id": {"type": "int", "description": "GitLab project ID (single egg)"},
            "group_id": {"type": "int", "description": "GitLab group ID (eggsbucket)"},
            "config": {"type": "dict", "description": "Full parsed .fly config as dict"},
            "git_commit": {"type": "str", "description": "Nest commit SHA when this config was synced"},
            "git_repo_url_secret": {"type": "str", "description": "Secret URI for the repo URL"},
            "gitlab_token_secret_uri": {"type": "str", "description": "Secret URI for GitLab registration token"},
            "gitlab_webhook_secret_uri": {"type": "str", "description": "Secret URI for webhook validation token"},
            "synced_at": {"type": "datetime"},
            "created_at": {"type": "datetime"},
            "updated_at": {"type": "datetime"},
        },
    },
    "SyncHistory": {
        "description": "Record of a single Nest Git sync operation.",
        "fields": {
            "id": {"type": "str"},
            "git_commit": {"type": "str"},
            "sync_type": {"type": "str", "enum": ["scheduled", "webhook", "manual"]},
            "status": {"type": "SyncStatus", "enum": ["SUCCESS", "FAILED"]},
            "changes_detected": {"type": "int"},
            "eggs_synced": {"type": "int"},
            "jobs_synced": {"type": "int"},
            "uf_config_synced": {"type": "bool"},
            "error_message": {"type": "str | None"},
            "synced_at": {"type": "datetime"},
            "duration_ms": {"type": "int"},
        },
    },
    "DeploymentPlan": {
        "description": "Stored OpenTofu plan for a runner deployment, enabling rollback.",
        "fields": {
            "id": {"type": "str"},
            "egg_name": {"type": "str"},
            "plan_type": {"type": "str", "enum": ["deploy", "terminate", "update"]},
            "config_hash": {"type": "str", "description": "SHA256 of the rendered Jinja2 config"},
            "status": {"type": "DeploymentStatus", "enum": ["PENDING", "APPLIED", "ROLLED_BACK", "FAILED"]},
            "plan_binary": {"type": "bytes", "description": "Serialized OpenTofu plan binary"},
            "rollback_plan_id": {"type": "str | None", "description": "ID of the plan to apply for rollback"},
            "created_at": {"type": "datetime"},
            "applied_at": {"type": "datetime | None"},
            "metadata": {"type": "dict"},
        },
    },
    "AuditLog": {
        "description": "Immutable audit trail for all system actions.",
        "fields": {
            "id": {"type": "str"},
            "timestamp": {"type": "datetime"},
            "actor": {"type": "str", "description": "Service or user that performed the action"},
            "action": {"type": "str", "description": "Action name (e.g. runner.deploy, egg.sync)"},
            "resource_type": {"type": "str"},
            "resource_id": {"type": "str"},
            "details": {"type": "dict"},
        },
    },
}

MOTHERGOOSE_SERVICES: list[dict[str, Any]] = [
    {
        "name": "GitSyncService",
        "module": "services.git_sync_service",
        "description": "Clones/pulls the Nest repo via GitPython, delegates .fly parsing to FlyParserService, upserts results to DB.",
        "methods": ["sync_from_git(commit)", "parse_eggs(path)", "parse_jobs(path)", "parse_uf_config(path)"],
    },
    {
        "name": "FlyParserService",
        "module": "services.fly_parser",
        "description": "Invokes the Gosling CLI binary as a subprocess to parse .fly files into JSON dicts.",
        "methods": ["parse_egg(path)", "parse_job(path)", "parse_uf_config(path)", "_call_gosling_parse(path, type)"],
    },
    {
        "name": "SecretManager",
        "module": "services.secret_manager",
        "description": "Resolves secret URIs (yc-lockbox://, aws-sm://, vault://) to plaintext values. Uses SecretCache for TTL-based caching.",
        "methods": ["get_secret(uri)", "_yc_lockbox(uri)", "_aws_sm(uri)", "_vault(uri)"],
    },
    {
        "name": "RunnerOrchestrationService",
        "module": "services.runner_orchestration",
        "description": "Orchestrates runner lifecycle: determines type, calls OpenTofuConfiguration to deploy/terminate, stores DeploymentPlan.",
        "methods": ["deploy_runner(egg)", "terminate_runner(runner_id)", "determine_runner_type(egg)"],
    },
    {
        "name": "OpenTofuConfiguration",
        "module": "services.opentofu_config",
        "description": "Renders Jinja2 templates from templates/ directory, generates and applies OpenTofu plans via tofupy.",
        "methods": ["render_templates(settings)", "generate_plan()", "apply_plan(plan)"],
    },
    {
        "name": "OpenTofuBinary / OpenTofuDownloadGithub / OpenTofuDownloadFromOtherSource",
        "module": "services.opentofu_binary",
        "description": "Abstract base + concrete implementations for downloading and extracting OpenTofu binaries from GitHub releases or custom sources.",
        "methods": ["store_downloaded_bin()", "_download_and_extract(extract_to)"],
    },
    {
        "name": "GoslingBinaryManager",
        "module": "services.gosling_manager",
        "description": "Manages Gosling CLI binary versions: mounts from S3/object storage, verifies SHA256, activates a version for use by FlyParserService.",
        "methods": ["mount_s3_binaries()", "get_active_binary_path()", "verify_and_activate(version)"],
    },
    {
        "name": "EggService",
        "module": "services.egg_service",
        "description": "CRUD operations for EggConfig records. Used by routers and git sync.",
        "methods": ["get_egg(name)", "list_eggs()", "upsert_egg(config)", "delete_egg(name)"],
    },
    {
        "name": "DeploymentPlanService",
        "module": "services.deployment_plan_service",
        "description": "Creates, stores, applies, and rolls back OpenTofu deployment plans.",
        "methods": ["create_plan(egg_name, config)", "apply_plan(plan_id)", "rollback(plan_id)"],
    },
]

MOTHERGOOSE_ENV_VARS: list[dict[str, Any]] = [
    {"name": "MOTHERGOOSE_DB_BACKEND", "type": "str", "values": ["ydb", "dynamodb"], "description": "Database backend to use"},
    {"name": "MOTHERGOOSE_YDB_ENDPOINT", "type": "str", "description": "YDB gRPC endpoint (e.g. grpcs://ydb.serverless.yandexcloud.net:2135)"},
    {"name": "MOTHERGOOSE_YDB_DATABASE", "type": "str", "description": "YDB database path"},
    {"name": "MOTHERGOOSE_DYNAMODB_TABLE_PREFIX", "type": "str", "description": "Prefix for DynamoDB table names"},
    {"name": "MOTHERGOOSE_AWS_REGION", "type": "str", "description": "AWS region for DynamoDB/SQS/Secrets Manager"},
    {"name": "MOTHERGOOSE_CELERY_BROKER_URL", "type": "str", "description": "Celery broker URL (SQS or YMQ endpoint)"},
    {"name": "MOTHERGOOSE_CELERY_RESULT_BACKEND", "type": "str", "description": "Celery result backend URL"},
    {"name": "MOTHERGOOSE_NEST_REPO_URL_SECRET", "type": "str", "description": "Secret URI for the Nest Git repo URL"},
    {"name": "MOTHERGOOSE_NEST_REPO_BRANCH", "type": "str", "default": "main", "description": "Nest repo branch to sync"},
    {"name": "MOTHERGOOSE_INTERNAL_SECRET_TOKEN", "type": "str", "description": "Token for /internal/* endpoints"},
    {"name": "MOTHERGOOSE_OPENTOFU_BINARY_PATH", "type": "str", "description": "Path to the active OpenTofu binary"},
    {"name": "MOTHERGOOSE_GOSLING_BINARY_PATH", "type": "str", "description": "Path to the active Gosling CLI binary"},
    {"name": "MOTHERGOOSE_SECRET_CACHE_TTL", "type": "int", "default": 300, "description": "Secret cache TTL in seconds"},
    {"name": "MOTHERGOOSE_LOG_LEVEL", "type": "str", "default": "INFO", "values": ["DEBUG", "INFO", "WARNING", "ERROR"]},
    {"name": "MOTHERGOOSE_YC_FOLDER_ID", "type": "str", "description": "Yandex Cloud folder ID for resource provisioning"},
    {"name": "MOTHERGOOSE_COMPUTE_MODULE_SOURCE", "type": "str", "description": "OpenTofu module source path or registry URL"},
]
