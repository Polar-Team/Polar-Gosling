"""Gosling CLI and .fly language data for the MCP server."""

from typing import Any

GOSLING_COMMANDS: list[dict[str, Any]] = [
    {
        "name": "init",
        "usage": "gosling init [path] [flags]",
        "description": "Initialize a new Nest repository structure. Creates Eggs/, Jobs/, UF/ directories with a README.md and .gitignore. Defaults to the current directory.",
        "flags": [
            {"flag": "--path", "short": "-p", "type": "string", "default": ".", "description": "Target directory for the Nest repository"},
        ],
        "example": "gosling init /path/to/nest",
    },
    {
        "name": "add egg",
        "usage": "gosling add egg <name> [flags]",
        "description": "Scaffold a new Egg config.fly under Eggs/<name>/. The name must be alphanumeric with hyphens or underscores.",
        "flags": [
            {"flag": "--type", "short": "-t", "type": "string", "values": ["vm", "serverless"], "default": "vm", "description": "Runner type"},
            {"flag": "--provider", "short": "-p", "type": "string", "values": ["yandex", "aws"], "default": "yandex", "description": "Cloud provider"},
            {"flag": "--region", "short": "-r", "type": "string", "description": "Cloud region (e.g. ru-central1-a, us-east-1)"},
            {"flag": "--interactive", "short": "-i", "type": "bool", "default": False, "description": "Interactive mode for guided setup"},
        ],
        "example": "gosling add egg my-app --type vm --provider yandex",
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
        "description": "Read all Egg configurations from the Nest, compute a config hash, and apply changes via the MotherGoose API. Skips eggs where the config hash is unchanged.",
        "flags": [
            {"flag": "--api-url", "type": "string", "required": True, "description": "MotherGoose API base URL"},
            {"flag": "--api-key", "type": "string", "required": True, "description": "MotherGoose API authentication key"},
            {"flag": "--cloud", "type": "string", "required": True, "values": ["yandex", "aws"], "description": "Cloud provider"},
            {"flag": "--region", "type": "string", "required": True, "description": "Target cloud region"},
            {"flag": "--dry-run", "type": "bool", "default": False, "description": "Preview changes without applying"},
        ],
        "example": "gosling deploy --cloud yandex --region ru-central1-a --api-url https://mg.example.com --api-key $MG_API_KEY --dry-run",
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
                "gitlab.token_secret": {"type": "string", "description": "Secret URI (yc-lockbox://, aws-sm://, vault://)"},
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
            "location": "UF/config.fly",
            "description": "UglyFox lifecycle and pruning configuration.",
            "nested_blocks": {
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
                "policies.rule.<name>": {
                    "condition": {"type": "string", "description": "Rule condition expression"},
                    "action": {"type": "string", "values": ["terminate", "demote_to_nadir", "promote_to_apex"]},
                },
            },
        },
        "mothergoose": {
            "location": "MG/config.fly",
            "description": "MotherGoose infrastructure configuration: API Gateway, queues, triggers, containers.",
            "nested_blocks": ["api_gateway", "message_queue", "cloud_trigger", "container"],
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
uglyfox {
  pruning {
    max_age_hours          = 72
    max_failures           = 5
    idle_timeout_minutes   = 30
    check_interval_seconds = 60
  }

  apex_pool {
    min_size           = 1
    max_size           = 10
    scale_up_threshold = 5
  }

  nadir_pool {
    min_size            = 0
    max_size            = 5
    warmup_time_seconds = 30
  }

  runners_condition {
    egg_name     = "my-service"
    max_failures = 3
    max_age_hours = 24
  }
}
""",
    "mothergoose_config": """
# MG/config.fly
mothergoose {
  api_gateway {
    cloud_provider = "yandex"
    region         = "ru-central1"
    domain         = "mg.example.com"
    tls             = true
  }

  message_queue {
    cloud_provider = "yandex"
    queue_name     = "polar-gosling-tasks"
  }

  cloud_trigger {
    type     = "timer"
    schedule = "*/5 * * * *"  # Every 5 minutes
    target   = "/internal/sync-git"
  }

  container {
    image  = "registry.example.com/polar-gosling/mothergoose:latest"
    cpu    = 1
    memory = "512MB"
    min_instances = 1
    max_instances = 5
  }
}
""",
}
