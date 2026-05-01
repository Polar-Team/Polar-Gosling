"""Gosling CLI and .fly language data for the MCP server."""

from typing import Any

GOSLING_COMMANDS: list[dict[str, Any]] = [
    {
        "name": "init",
        "usage": "gosling init [path] [flags]",
        "description": "Initialize a new Nest repository structure. Creates Eggs/, Jobs/, UF/, MG/ directories with README.md, .gitignore, and default .fly config templates. Initializes a Git repository and optionally configures an upstream remote (interactively or via flags). Defaults to the current directory.",
        "flags": [
            {"flag": "--path", "short": "-p", "type": "string", "default": ".", "description": "Target directory for the Nest repository"},
            {"flag": "--remote-name", "type": "string", "default": "main", "description": "Name for the upstream Git remote"},
            {"flag": "--remote-url", "type": "string", "default": "", "description": "URL for the upstream Git remote. If set, skips interactive prompts"},
            {"flag": "--branch", "type": "string", "default": "main", "description": "Default branch name for the upstream remote"},
        ],
        "behavior": {
            "git_init": "Runs 'git init' in the target directory after creating files",
            "upstream_remote": {
                "flag_mode": "When --remote-url is provided, uses flag values directly without prompting",
                "interactive_mode": "When running in a terminal and no --remote-url flag, prompts for remote name (default 'main'), URL, and branch (default 'main'). Empty URL skips remote configuration.",
                "non_interactive": "When stdin is not a terminal and no --remote-url flag, skips remote configuration silently",
            },
            "success_output": "Shows configured remote info and suggests 'git push -u <remote> <branch>'. When no remote is configured, suggests manually adding one.",
        },
        "example": "gosling init /path/to/nest --remote-url https://gitlab.com/team/nest.git --branch main",
    },
    {
        "name": "add egg",
        "usage": "gosling add egg <name> [flags]",
        "description": "Scaffold a new Egg config.fly under Eggs/<name>/. The name must be alphanumeric with hyphens or underscores. In interactive mode, guides the user through secret store setup (create new, use existing, or placeholder URIs).",
        "flags": [
            {"flag": "--type", "short": "-t", "type": "string", "values": ["vm", "serverless"], "default": "vm", "description": "Runner type"},
            {"flag": "--provider", "short": "-p", "type": "string", "values": ["yandex", "aws"], "default": "yandex", "description": "Cloud provider"},
            {"flag": "--region", "short": "-r", "type": "string", "description": "Cloud region (e.g. ru-central1-a, us-east-1)"},
            {"flag": "--folder-id", "type": "string", "description": "Yandex Cloud folder ID (for interactive secret store creation)"},
            {"flag": "--interactive", "short": "-i", "type": "bool", "default": False, "description": "Interactive mode for guided setup including secret store provisioning"},
        ],
        "interactive_flow": {
            "description": "When --interactive is set, after collecting egg name/provider/region the CLI prompts for secret store setup.",
            "steps": [
                "Prompt: 'Does a secret store already exist? [y/n]'",
                "If yes: prompt for secret ID (YC) or secret name (AWS), generate URIs via GenerateAllURIs",
                "If no: prompt 'Create one now? [y/n]'",
                "If create yes: call SecretStore.Create() programmatically, use returned URIs",
                "If create no: use placeholder URIs with TODO comments in config.fly",
            ],
        },
        "example": "gosling add egg my-app --type vm --provider yandex --interactive",
    },
    {
        "name": "add job",
        "usage": "gosling add job <name> [flags]",
        "description": "Scaffold a new Job .fly file under Jobs/<name>.fly.",
        "flags": [
            {"flag": "--schedule", "short": "-s", "type": "string", "description": "Cron expression (e.g. '0 2 * * *')"},
            {"flag": "--interactive", "short": "-i", "type": "bool", "default": False, "description": "Interactive mode"},
        ],
        "example": "gosling add job rotate-secrets --schedule '0 2 * * *'",
    },
    {
        "name": "validate",
        "usage": "gosling validate [file] [flags]",
        "description": "Validate .fly files. Without arguments, discovers and validates all .fly files in the Nest repository. Exits non-zero if any file fails. Checks block type correctness, required attributes, value type constraints, identifier naming rules, and file-location consistency.",
        "flags": [
            {"flag": "--path", "short": "-p", "type": "string", "default": ".", "description": "Path to Nest repository root"},
            {"flag": "--all", "short": "-a", "type": "bool", "default": False, "description": "Validate all files (default when no file arg given)"},
        ],
        "example": "gosling validate Eggs/my-app/config.fly",
        "exit_codes": {"0": "valid", "1": "validation errors"},
    },
    {
        "name": "fmt",
        "usage": "gosling fmt [file...] [flags]",
        "description": "Format .fly files to canonical style: attributes sorted alphabetically, 2-space indentation, consistent brace placement, normalized list formatting. Without arguments, discovers and formats all .fly files in the Nest. Idempotent — files already formatted are left untouched. Parse errors leave the original file unchanged.",
        "flags": [
            {"flag": "--path", "short": "-p", "type": "string", "default": ".", "description": "Path to Nest repository root"},
            {"flag": "--check", "type": "bool", "default": False, "description": "Exit non-zero if any file is not formatted (no writes)"},
            {"flag": "--diff", "type": "bool", "default": False, "description": "Print unified diff of changes instead of writing"},
            {"flag": "--stdout", "type": "bool", "default": False, "description": "Print formatted output to stdout (requires exactly one file argument; mutually exclusive with --check and --diff)"},
        ],
        "canonical_format_rules": {
            "indentation": "2 spaces per nesting level",
            "brace_placement": "Opening { on same line as block type; closing } on its own line",
            "attribute_order": "Sorted alphabetically within each block",
            "nested_block_order": "Preserved from source",
            "short_lists": "Inline for <= 2 elements: [\"a\", \"b\"]",
            "long_lists": "Multi-line with trailing comma for > 2 elements",
            "top_level_blocks": "Separated by exactly one blank line; no trailing newline at EOF",
        },
        "example": "gosling fmt --check",
    },
    {
        "name": "parse",
        "usage": "gosling parse <file> [flags]",
        "description": "Parse a single .fly file and output the full AST as JSON to stdout. Used internally by MotherGoose FlyParserService via subprocess. Errors go to stderr; exits non-zero on failure. Output uses snake_case field names. Top-level key is 'blocks'; each block contains type, labels, attributes, and nested blocks.",
        "flags": [
            {"flag": "--type", "short": "-t", "type": "string", "values": ["egg", "eggsbucket", "job", "uglyfox"], "required": False, "description": "Expected block type (optional)"},
        ],
        "example": "gosling parse Eggs/my-app/config.fly --type egg",
        "stdout": "JSON AST of the parsed .fly file",
        "stderr": "Validation/parse errors",
    },
    {
        "name": "deploy",
        "usage": "gosling deploy [flags]",
        "description": "Bootstrap MotherGoose/UglyFox infrastructure by reading MG/ and UF/ .fly configs from the Nest, then provisioning cloud resources (YDB, YMQ, S3, Container Registry, Serverless Containers, API Gateway, Timer Triggers) via cloud SDKs. Cloud provider and region are read from the cloud {} block in the .fly files — no --cloud or --region flags needed. After all resources are created, triggers an initial Git sync.",
        "flags": [
            {"flag": "--api-url", "type": "string", "required": True, "description": "MotherGoose API base URL"},
            {"flag": "--api-key", "type": "string", "required": True, "description": "MotherGoose API authentication key"},
            {"flag": "--name", "type": "string", "required": False, "description": "Deploy a specific named MotherGoose/UglyFox instance (omit to deploy all instances)"},
            {"flag": "--dry-run", "type": "bool", "default": False, "description": "Preview changes without applying"},
        ],
        "note": "Cloud provider and region come from the cloud {} block inside the mothergoose .fly file. --cloud and --region flags were removed in favour of .fly-driven config.",
        "example": "gosling deploy --api-url https://mg.example.com --api-key $MG_API_KEY --name yandex_test --dry-run",
    },
    {
        "name": "rollback",
        "usage": "gosling rollback [flags]",
        "description": "Roll back an egg's deployment. Without --to, rolls back to the most recent previously applied plan. Prompts for confirmation before proceeding.",
        "flags": [
            {"flag": "--egg", "type": "string", "required": True, "description": "Egg name to roll back"},
            {"flag": "--api-url", "type": "string", "required": True, "description": "MotherGoose API base URL"},
            {"flag": "--api-key", "type": "string", "required": True, "description": "MotherGoose API authentication key"},
            {"flag": "--to", "type": "string", "description": "Specific plan ID to roll back to"},
        ],
        "example": "gosling rollback --egg my-app --api-url https://mg.example.com --api-key $MG_API_KEY",
    },
    {
        "name": "status",
        "usage": "gosling status [flags]",
        "description": "Display current deployment status, active runners, and deployment history for one or all eggs. Queries the MotherGoose API.",
        "flags": [
            {"flag": "--api-url", "type": "string", "required": True, "description": "MotherGoose API base URL"},
            {"flag": "--api-key", "type": "string", "required": True, "description": "MotherGoose API authentication key"},
            {"flag": "--egg", "type": "string", "description": "Show status for a specific egg"},
            {"flag": "--all", "type": "bool", "description": "Show status for all eggs"},
        ],
        "example": "gosling status --egg my-app --api-url https://mg.example.com --api-key $MG_API_KEY",
    },
    {
        "name": "runner",
        "usage": "gosling runner [flags]",
        "description": "Start the GitLab Runner Agent manager inside a deployed runner container or VM. Registers the agent with GitLab, manages the agent process lifecycle, synchronises the agent version, and reports health metrics and heartbeats to MotherGoose.",
        "flags": [
            {"flag": "--egg-name", "type": "string", "required": True, "description": "Name of the Egg this runner belongs to"},
            {"flag": "--runner-id", "type": "string", "description": "Unique runner ID (defaults to runner-<egg-name>)"},
            {"flag": "--token-secret", "type": "secret_uri", "required": True, "description": "Secret URI for the GitLab runner registration token (e.g. yc-lockbox://abc/runner-token)"},
            {"flag": "--gitlab-server", "type": "string", "default": "gitlab.com", "description": "GitLab server FQDN"},
            {"flag": "--tags", "type": "string", "description": "Comma-separated runner tags"},
            {"flag": "--agent-version", "type": "string", "description": "Required GitLab Runner Agent version (uses latest if not specified)"},
            {"flag": "--mothergoose-url", "type": "string", "description": "MotherGoose API URL for metrics reporting"},
            {"flag": "--api-key", "type": "string", "description": "MotherGoose API key"},
            {"flag": "--metrics-interval", "type": "duration", "default": "30s", "description": "How often to report full metrics to MotherGoose"},
            {"flag": "--heartbeat-interval", "type": "duration", "default": "30s", "description": "How often to send heartbeat pings to MotherGoose"},
        ],
        "signals": {
            "SIGTERM": "Graceful shutdown",
            "SIGINT": "Graceful shutdown",
            "SIGHUP": "Reload configuration without restart",
        },
        "note": "This is the container/VM entrypoint injected by OpenTofu during runner deployment. Not intended for direct developer use.",
        "example": "gosling runner --egg-name my-service --token-secret yc-lockbox://abc123/runner-token --mothergoose-url https://mg.example.com",
    },
    {
        "name": "rift serve",
        "usage": "rift serve [flags]",
        "binary": "rift",
        "description": "Start the Rift Docker context proxy and image cache server. Rift is a separate binary (cmd/rift/) that shares the CLI package. It proxies Docker API requests from runners to the local Docker daemon and caches image tarballs to S3. Lifecycle (VM create/destroy) is driven by MotherGoose via OpenTofu — Rift itself only handles the serve loop.",
        "flags": [
            {"flag": "--listen", "type": "string", "default": ":2376", "description": "TCP address to listen on"},
            {"flag": "--auth-token-secret", "type": "secret_uri", "required": True, "description": "Secret URI for the bearer auth token (yc-lockbox://, aws-sm://, vault://)"},
            {"flag": "--s3-bucket", "type": "string", "required": True, "description": "S3 bucket for image cache storage"},
            {"flag": "--s3-endpoint", "type": "string", "description": "S3-compatible endpoint URL (empty = AWS default)"},
            {"flag": "--s3-region", "type": "string", "description": "S3 bucket region"},
            {"flag": "--s3-key-prefix", "type": "string", "default": "rift/", "description": "Key prefix inside the S3 bucket"},
            {"flag": "--s3-credentials-secret", "type": "secret_uri", "description": "Secret URI for S3 credentials"},
            {"flag": "--docker-socket", "type": "string", "default": "/var/run/docker.sock", "description": "Path to Docker daemon socket"},
            {"flag": "--image-cache-dir", "type": "string", "default": "/var/cache/rift/images", "description": "Local directory for image tarballs"},
            {"flag": "--mothergoose-url", "type": "string", "description": "MotherGoose API URL for state reporting"},
            {"flag": "--api-key", "type": "string", "description": "MotherGoose API key"},
            {"flag": "--anti-flap", "type": "duration", "default": "2m", "description": "Minimum time in running state before shutdown is allowed"},
            {"flag": "--idle-timeout", "type": "duration", "default": "10m", "description": "Idle time before automatic shutdown"},
            {"flag": "--cache-sync-interval", "type": "duration", "default": "5m", "description": "How often to sync image cache to S3"},
        ],
        "signals": {
            "SIGTERM": "Graceful shutdown (30s timeout)",
            "SIGINT": "Graceful shutdown (30s timeout)",
        },
        "env_fallback": "RIFT_AUTH_TOKEN — used for local development when secret backend is not yet wired",
        "note": "Cannot be used by Job runners (Job runners cannot use Rift). Only Egg runners with tofu_rift_required=true get a Rift server.",
        "example": "rift serve --auth-token-secret yc-lockbox://abc123/rift-token --s3-bucket my-rift-cache --s3-region ru-central1",
    },
    {
        "name": "lockbox",
        "usage": "gosling lockbox <subcommand> [flags]",
        "description": "Manage cloud secret stores for Egg configurations. Parent command that groups create, list, and verify subcommands for provisioning and inspecting secrets in Yandex Cloud Lockbox or AWS Secrets Manager.",
        "subcommands": ["create", "list", "verify"],
        "example": "gosling lockbox --help",
    },
    {
        "name": "lockbox create",
        "usage": "gosling lockbox create [flags]",
        "description": "Create a new cloud secret store with placeholder entries for an Egg configuration. The secret is created with three required entries (runner-token, webhook-secret, repo-url) set to empty placeholder values. After creation, the generated Secret URIs are printed to stdout (one per line) for use in config.fly.",
        "flags": [
            {"flag": "--provider", "type": "string", "values": ["yandex", "aws"], "required": True, "description": "Cloud provider"},
            {"flag": "--egg-name", "type": "string", "required": True, "description": "Name of the Egg"},
            {"flag": "--folder-id", "type": "string", "description": "Yandex Cloud folder ID (required for yandex provider)"},
            {"flag": "--region", "type": "string", "description": "AWS region (optional, uses SDK default if omitted)"},
        ],
        "secret_naming": {
            "yandex": "pg-{egg-name}-secrets",
            "aws": "polar-gosling/{egg-name}",
        },
        "required_entries": ["runner-token", "webhook-secret", "repo-url"],
        "labels_tags": {"polar-gosling": "true", "egg-name": "{egg-name}"},
        "stdout": "One Secret URI per line for each required entry",
        "example": "gosling lockbox create --provider yandex --egg-name my-app --folder-id abc123",
    },
    {
        "name": "lockbox list",
        "usage": "gosling lockbox list [flags]",
        "description": "List all cloud secret stores tagged for Polar Gosling use. Displays name, ID/ARN, associated egg name, and creation date for each secret store. Prints 'no Polar Gosling secret stores found' to stderr when the list is empty.",
        "flags": [
            {"flag": "--provider", "type": "string", "values": ["yandex", "aws"], "required": True, "description": "Cloud provider"},
            {"flag": "--folder-id", "type": "string", "description": "Yandex Cloud folder ID (required for yandex provider)"},
            {"flag": "--region", "type": "string", "description": "AWS region (optional, uses SDK default if omitted)"},
        ],
        "output_columns": ["NAME", "ID", "EGG", "CREATED"],
        "example": "gosling lockbox list --provider yandex --folder-id abc123",
    },
    {
        "name": "lockbox verify",
        "usage": "gosling lockbox verify [flags]",
        "description": "Verify that a cloud secret store exists and contains all required entries (runner-token, webhook-secret, repo-url). Prints a success message listing found entries (exit 0) or a present/missing report (exit non-zero) if any entries are missing.",
        "flags": [
            {"flag": "--provider", "type": "string", "values": ["yandex", "aws"], "required": True, "description": "Cloud provider"},
            {"flag": "--secret-id", "type": "string", "description": "Yandex Cloud Lockbox secret ID"},
            {"flag": "--secret-name", "type": "string", "description": "AWS Secrets Manager secret name"},
            {"flag": "--folder-id", "type": "string", "description": "Yandex Cloud folder ID (required for yandex provider)"},
            {"flag": "--region", "type": "string", "description": "AWS region (optional, uses SDK default if omitted)"},
        ],
        "exit_codes": {"0": "All required entries present", "1": "Missing entries or error"},
        "example": "gosling lockbox verify --provider yandex --folder-id abc123 --secret-id e6q...",
    },
]

FLY_LANGUAGE_REFERENCE: dict[str, Any] = {
    "description": ".fly is an HCL-like DSL with stronger typing used to define all Polar Gosling configuration. Parsed by the Gosling CLI.",
    "syntax_rules": [
        "Block syntax: block_type 'label' { ... } or block_type { ... }",
        "Attribute syntax: key = value",
        "String values use double quotes",
        "Numbers are unquoted",
        "Booleans: true / false",
        "Lists: [item1, item2]",
        "Secret URIs are strings with scheme prefix: yc-lockbox://, aws-sm://, vault://",
        "Comments: # single line",
    ],
    "type_system": {
        "string": "Double-quoted UTF-8 string",
        "number": "Integer or float",
        "bool": "true or false",
        "list(string)": "List of strings",
        "list(number)": "List of numbers",
        "secret_uri": "String matching yc-lockbox://, aws-sm://, or vault:// scheme",
        "cron": "Standard 5-field cron expression string",
        "duration": "String with unit suffix: 30s, 5m, 2h, 7d",
        "memory": "String with unit: 256MB, 1GB, 2048MB",
        "cpu": "Number of vCPUs (float allowed: 0.5, 1, 2)",
    },
    "block_types": {
        "egg": {
            "location": "Eggs/<name>/config.fly",
            "description": "Single managed GitLab project runner configuration.",
            "required_blocks_and_attributes": {
                "type": {"type": "string", "values": ["vm", "serverless"], "description": "Runner deployment type"},
                "cloud.provider": {"type": "string", "values": ["yandex", "aws"], "description": "Target cloud provider"},
                "cloud.region": {"type": "string", "description": "Valid cloud region string"},
                "resources.cpu": {"type": "number", "constraints": "1–128 cores"},
                "resources.memory": {"type": "number", "constraints": "512–524288 MB"},
                "resources.disk": {"type": "number", "constraints": "10–10240 GB"},
                "runner.tags": {"type": "list(string)", "constraints": "At least one tag"},
                "runner.concurrent": {"type": "number", "constraints": "1–100"},
                "gitlab.project_id": {"type": "number", "constraints": "1–999999999"},
                "gitlab.server_name": {"type": "string", "description": "GitLab instance URL or FQDN"},
                "gitlab_token_secret": {"type": "secret_uri", "description": "Secret URI for GitLab runner registration token (yc-lockbox://, aws-sm://, vault://). Top-level egg attribute."},
                "gitlab_webhook_secret": {"type": "secret_uri", "description": "Secret URI for GitLab webhook validation secret. Top-level egg attribute."},
                "git_repo_url_secret": {"type": "secret_uri", "description": "Secret URI for the Git repository clone URL. Top-level egg attribute."},
            },
            "optional_blocks": {
                "runner.idle_timeout": {"type": "string", "description": "Duration string e.g. '10m'"},
                "environment": {"type": "map", "description": "Key-value environment variables"},
            },
        },
        "eggsbucket": {
            "location": "Eggs/<name>/config.fly",
            "description": "Group of GitLab projects sharing a single runner configuration. Replaces the gitlab block with a repositories block containing individual repo entries.",
            "required_blocks_and_attributes": {
                "type": {"type": "string", "values": ["vm", "serverless"]},
                "cloud.provider": {"type": "string", "values": ["yandex", "aws"]},
                "cloud.region": {"type": "string"},
                "resources.cpu": {"type": "number"},
                "resources.memory": {"type": "number"},
                "resources.disk": {"type": "number"},
                "runner.tags": {"type": "list(string)"},
                "runner.concurrent": {"type": "number"},
                "repositories.repo.<name>.gitlab.project_id": {"type": "number"},
                "repositories.repo.<name>.gitlab.server_name": {"type": "string"},
                "repositories.repo.<name>.gitlab.token_secret": {"type": "string", "description": "Secret URI"},
            },
        },
        "job": {
            "location": "Jobs/<name>.fly",
            "description": "Internal self-management task run on a dedicated runner. 10-minute time limit. Cannot use Rift servers.",
            "required_attributes": {
                "schedule": {"type": "string", "description": "Cron expression (5 or 6 space-separated fields)"},
                "script": {"type": "string", "description": "Shell script to execute (heredoc supported)"},
                "runner.type": {"type": "string", "values": ["vm", "serverless"]},
                "runner.tags": {"type": "list(string)", "description": "GitLab runner tags for job routing"},
            },
            "optional_blocks": {
                "on_failure.notify": {"type": "list(string)", "description": "Email addresses to notify on failure"},
            },
        },
        "uglyfox": {
            "location": "UF/*.fly (multiple files allowed; multiple blocks per file allowed)",
            "description": "UglyFox lifecycle and pruning configuration. Has a label (instance name) and a required mothergoose attribute that binds it to a named MotherGoose instance. UglyFox does NOT have its own cloud block — it inherits cloud settings from the referenced MotherGoose instance.",
            "label": "instance name — e.g. uglyfox \"yandex_test\" { ... }",
            "required_attributes": {
                "mothergoose": {"type": "string", "description": "Name of the MotherGoose instance to reference (e.g. \"yandex_test\"). Cloud settings are copied from that instance."},
            },
            "nested_blocks": {
                "workers": {"description": "ServerlessContainerConfig for UglyFox Celery worker container"},
                "service_account": {"description": "ServiceAccountConfig for UglyFox"},
                "pruning": {
                    "failed_threshold": {"type": "number", "constraints": "1–100", "description": "Terminate after N failures"},
                    "max_age": {"type": "string", "description": "Max runner lifetime duration"},
                    "check_interval": {"type": "string", "description": "How often UglyFox checks"},
                },
                "runners_condition.<name>": {
                    "eggs_entities": {"type": "list(string)", "description": "Egg names this condition applies to"},
                    "apex.max_count": {"type": "number"},
                    "apex.min_count": {"type": "number"},
                    "apex.cpu_threshold": {"type": "number", "description": "% — scale up when exceeded"},
                    "apex.memory_threshold": {"type": "number", "description": "% — scale up when exceeded"},
                    "nadir.max_count": {"type": "number"},
                    "nadir.min_count": {"type": "number"},
                    "nadir.idle_timeout": {"type": "string", "description": "Demote to nadir after idle"},
                },
                "policies": {
                    "description": "Optional policy rules for runner lifecycle actions.",
                    "rule.<name>": {
                        "condition": {"type": "string", "description": "Rule condition expression"},
                        "action": {"type": "string", "values": ["terminate", "demote_to_nadir", "promote_to_apex"]},
                    },
                },
            },
        },
        "mothergoose": {
            "location": "MG/*.fly (multiple files allowed; multiple blocks per file allowed)",
            "description": "MotherGoose infrastructure configuration. Has a label (instance name). Defines cloud provider, API Gateway, FastAPI app container, Celery worker container, message queues, cloud triggers, YDB database, S3 storage, and service accounts. Multiple named instances can coexist (e.g. yandex_test, aws_prod).",
            "label": "instance name — e.g. mothergoose \"yandex_test\" { ... }",
            "nested_blocks": {
                "cloud": {
                    "provider": {"type": "string", "values": ["yandex", "aws"]},
                    "yc_folder_id": {"type": "string", "description": "Required when provider=yandex"},
                    "yc_cloud_id": {"type": "string", "description": "Required when provider=yandex"},
                    "aws_region": {"type": "string", "description": "Required when provider=aws"},
                    "aws_account_id": {"type": "string", "description": "Required when provider=aws"},
                },
                "api_gateway": {
                    "name": {"type": "string"},
                    "description": {"type": "string"},
                    "service_account": {"type": "string"},
                },
                "fastapi_app": {
                    "description": "ServerlessContainerConfig for the FastAPI container",
                    "name": {"type": "string"},
                    "image": {"type": "string"},
                    "memory": {"type": "number", "description": "MB"},
                    "cores": {"type": "number", "description": "vCPUs"},
                    "core_fraction": {"type": "number", "description": "% of vCPU (YC only)"},
                    "execution_timeout": {"type": "string", "description": "Duration string e.g. '60s'"},
                    "concurrency": {"type": "number"},
                    "service_account": {"type": "string"},
                },
                "celery_workers": {
                    "description": "Celery worker container config. Name, image, and service_account inherited from fastapi_app.",
                    "memory": {"type": "number", "description": "MB"},
                    "cores": {"type": "number", "description": "vCPUs"},
                    "core_fraction": {"type": "number", "description": "% of vCPU (YC only)"},
                    "execution_timeout": {"type": "string", "description": "Duration string"},
                    "concurrency": {"type": "number"},
                },
                "git_sync_trigger": {
                    "schedule": {"type": "cron", "description": "Cron expression e.g. '*/5 * * * *'"},
                    "service_account": {"type": "string"},
                },
                "mothergoose_queues": {
                    "description": "Task queue and dead-letter queue configuration.",
                    "task_queue": {
                        "name": {"type": "string"},
                        "visibility_timeout": {"type": "number"},
                        "message_retention": {"type": "number"},
                    },
                    "dlq": {
                        "name": {"type": "string"},
                        "message_retention": {"type": "number"},
                    },
                },
                "database": {
                    "name": {"type": "string"},
                    "type": {"type": "string", "description": "e.g. 'ydb'"},
                    "serverless_mode": {"type": "bool"},
                },
                "storage": {
                    "bucket_name": {"type": "string"},
                    "region": {"type": "string"},
                },
                "service_account.<name>": {
                    "description": {"type": "string"},
                    "roles": {"type": "list(string)"},
                },
            },
        },
    },
    "validation_error_format": {
        "description": "Validation errors are returned as JSON when using --format json flag.",
        "schema": {
            "file": "string — path to the .fly file",
            "errors": [
                {
                    "line": "number",
                    "column": "number",
                    "severity": "error | warning",
                    "code": "string — error code (e.g. MISSING_REQUIRED, INVALID_TYPE, INVALID_SECRET_URI)",
                    "message": "string — human-readable description",
                    "attribute": "string — attribute name that failed",
                }
            ],
        },
    },
}

FLY_EXAMPLES: dict[str, str] = {
    "egg_yandex_serverless": """
# Eggs/my-service/config.fly
egg "my-service" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_token_secret   = "yc-lockbox://abc123def456/runner-token"
  gitlab_webhook_secret = "yc-lockbox://abc123def456/webhook-secret"
  git_repo_url_secret   = "yc-lockbox://abc123def456/repo-url"

  cloud_provider = "yandex"
  region         = "ru-central1"
  runner_type    = "serverless"

  tags   = ["docker", "linux", "yandex"]
  cpu    = 1
  memory = "512MB"
  max_concurrent_jobs = 2
}
""",
    "eggsbucket_aws": """
# Eggs/platform-team/config.fly
eggsbucket "platform-team" {
  gitlab_server         = "https://gitlab.example.com"
  group_id              = 999
  gitlab_token_secret   = "aws-sm://prod/platform-runner-token/value"
  gitlab_webhook_secret = "aws-sm://prod/platform-webhook-secret/value"

  cloud_provider = "aws"
  region         = "us-east-1"
  runner_type    = "apex"

  tags       = ["docker", "linux", "aws"]
  cpu        = 2
  memory     = "1GB"
  project_ids = [101, 102, 103]
}
""",
    "job_secret_rotation": """
# Jobs/rotate-secrets.fly
job "rotate-secrets" {
  schedule       = "0 2 * * 0"  # Every Sunday at 2am
  cloud_provider = "yandex"
  region         = "ru-central1"
  runner_image   = "registry.example.com/tools/secret-rotator:latest"
  cpu            = 0.5
  memory         = "256MB"
  timeout        = "30m"

  script = <<-EOT
    #!/bin/bash
    set -euo pipefail
    python3 /app/rotate.py --all
  EOT

  secrets = [
    "yc-lockbox://abc123/rotation-key",
  ]
}
""",
    "uglyfox_config": """
# UF/config.fly
uglyfox "yandex_test" {
  mothergoose = "yandex_test"

  pruning {
    failed_threshold = 3
    max_age          = "24h"
    check_interval   = "5m"
  }

  runners_condition "default" {
    eggs_entities = ["my-service", "platform-team"]

    apex {
      max_count        = 10
      min_count        = 2
      cpu_threshold    = 80
      memory_threshold = 70
    }

    nadir {
      max_count    = 5
      min_count    = 0
      idle_timeout = "30m"
    }
  }

  policies {
    rule "terminate_old_failed" {
      condition = "failed_count >= 3 AND age > 1h"
      action    = "terminate"
    }
  }
}
""",
    "mothergoose_config": """
# MG/config.fly
mothergoose "yandex_test" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1gxxxxxxxxxxxxxxx"
    yc_cloud_id  = "b1gyyyyyyyyyyyyyyy"
  }

  api_gateway {
    name            = "polar-gosling-gw"
    description     = "Main API gateway"
    service_account = "mg-sa"
  }

  fastapi_app {
    name              = "mg-fastapi"
    image             = "ghcr.io/polar-team/mothergoose:latest"
    memory            = 512
    cores             = 1
    execution_timeout = "60s"
    concurrency       = 4
    service_account   = "mg-sa"
  }

  celery_workers {
    memory            = 1024
    cores             = 2
    execution_timeout = "300s"
    concurrency       = 2
  }

  git_sync_trigger {
    schedule        = "*/5 * * * *"
    service_account = "mg-sa"
  }

  mothergoose_queues {
    task_queue {
      name               = "mg-tasks"
      visibility_timeout = 30
      message_retention  = 86400
    }
    dlq {
      name              = "mg-tasks-dlq"
      message_retention = 86400
    }
  }

  database {
    name            = "polar-gosling-db"
    type            = "ydb"
    serverless_mode = true
  }

  storage {
    bucket_name = "polar-gosling-storage"
    region      = "ru-central1"
  }

  service_account "mg-sa" {
    description = "MotherGoose service account"
    roles       = ["lockbox.payloadViewer", "ydb.editor"]
  }
}
""",
}
