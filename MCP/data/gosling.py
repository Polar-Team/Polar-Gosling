"""Gosling CLI and .fly language data for the MCP server."""

from typing import Any

GOSLING_COMMANDS: list[dict[str, Any]] = [
    {
        "name": "init",
        "usage": "gosling init [flags]",
        "description": "Initialize a new Nest repository structure. Creates Eggs/, Jobs/, UF/, MG/ directories with starter .fly files.",
        "flags": [
            {"flag": "--name", "short": "-n", "type": "string", "description": "Nest repository name"},
            {"flag": "--cloud", "short": "-c", "type": "string", "values": ["yandex", "aws", "both"], "description": "Target cloud provider(s)"},
            {"flag": "--output", "short": "-o", "type": "string", "default": ".", "description": "Output directory"},
        ],
        "example": "gosling init --name my-nest --cloud both",
    },
    {
        "name": "add egg",
        "usage": "gosling add egg <name> [flags]",
        "description": "Scaffold a new Egg config.fly under Eggs/<name>/. Prompts for runner type, cloud provider, and GitLab project ID.",
        "flags": [
            {"flag": "--type", "short": "-t", "type": "string", "values": ["egg", "eggsbucket"], "default": "egg", "description": "Egg type"},
            {"flag": "--cloud", "short": "-c", "type": "string", "values": ["yandex", "aws"], "description": "Cloud provider"},
            {"flag": "--runner-type", "type": "string", "values": ["serverless", "apex", "nadir"], "description": "Runner deployment type"},
            {"flag": "--project-id", "type": "int", "description": "GitLab project ID (egg only)"},
            {"flag": "--group-id", "type": "int", "description": "GitLab group ID (eggsbucket only)"},
        ],
        "example": "gosling add egg my-service --cloud yandex --runner-type serverless --project-id 12345",
    },
    {
        "name": "add job",
        "usage": "gosling add job <name> [flags]",
        "description": "Scaffold a new Job .fly file under Jobs/<name>.fly.",
        "flags": [
            {"flag": "--schedule", "short": "-s", "type": "string", "description": "Cron schedule expression"},
            {"flag": "--cloud", "short": "-c", "type": "string", "values": ["yandex", "aws"], "description": "Cloud provider for the job runner"},
        ],
        "example": "gosling add job rotate-secrets --schedule '0 2 * * *' --cloud aws",
    },
    {
        "name": "validate",
        "usage": "gosling validate [path] [flags]",
        "description": "Validate .fly files in the given path (or current directory). Checks syntax, required fields, type constraints, and secret URI formats.",
        "flags": [
            {"flag": "--path", "short": "-p", "type": "string", "default": ".", "description": "Path to validate"},
            {"flag": "--strict", "type": "bool", "description": "Fail on warnings in addition to errors"},
            {"flag": "--format", "type": "string", "values": ["text", "json"], "default": "text", "description": "Output format"},
        ],
        "example": "gosling validate ./Eggs/my-service --format json",
        "exit_codes": {"0": "valid", "1": "validation errors", "2": "parse error"},
    },
    {
        "name": "parse",
        "usage": "gosling parse <file> [flags]",
        "description": "Parse a single .fly file and output the result as JSON. Used internally by MotherGoose FlyParserService via subprocess.",
        "flags": [
            {"flag": "--type", "short": "-t", "type": "string", "values": ["egg", "eggsbucket", "job", "uglyfox", "mothergoose"], "required": True, "description": "Block type to parse"},
            {"flag": "--output", "short": "-o", "type": "string", "values": ["json", "pretty"], "default": "json", "description": "Output format"},
        ],
        "example": "gosling parse Eggs/my-service/config.fly --type egg",
        "stdout": "JSON representation of the parsed block",
        "stderr": "Validation errors if any",
    },
    {
        "name": "deploy",
        "usage": "gosling deploy <egg-name> [flags]",
        "description": "Trigger a runner deployment for an egg via the MotherGoose API (POST /eggs/{egg_name}/deploy).",
        "flags": [
            {"flag": "--server", "short": "-s", "type": "string", "description": "MotherGoose API base URL"},
            {"flag": "--token", "type": "string", "description": "Bearer token for MotherGoose API auth"},
            {"flag": "--wait", "short": "-w", "type": "bool", "description": "Wait for deployment to complete"},
            {"flag": "--timeout", "type": "int", "default": 300, "description": "Wait timeout in seconds"},
        ],
        "example": "gosling deploy my-service --server https://mg.example.com --wait",
    },
    {
        "name": "rollback",
        "usage": "gosling rollback <egg-name> [flags]",
        "description": "Roll back the last deployment for an egg by applying the stored rollback plan via MotherGoose API.",
        "flags": [
            {"flag": "--server", "short": "-s", "type": "string", "description": "MotherGoose API base URL"},
            {"flag": "--token", "type": "string", "description": "Bearer token"},
            {"flag": "--plan-id", "type": "string", "description": "Specific deployment plan ID to roll back to (defaults to last)"},
        ],
        "example": "gosling rollback my-service --server https://mg.example.com",
    },
    {
        "name": "status",
        "usage": "gosling status [egg-name] [flags]",
        "description": "Show runner status for an egg (or all eggs) by querying the MotherGoose API.",
        "flags": [
            {"flag": "--server", "short": "-s", "type": "string", "description": "MotherGoose API base URL"},
            {"flag": "--token", "type": "string", "description": "Bearer token"},
            {"flag": "--format", "type": "string", "values": ["table", "json"], "default": "table", "description": "Output format"},
            {"flag": "--watch", "short": "-w", "type": "bool", "description": "Continuously poll and refresh status"},
        ],
        "example": "gosling status my-service --server https://mg.example.com --format json",
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
            "required_attributes": {
                "gitlab_server": {"type": "string", "description": "GitLab instance URL"},
                "project_id": {"type": "number", "description": "GitLab project ID"},
                "gitlab_token_secret": {"type": "secret_uri", "description": "Runner registration token secret"},
                "gitlab_webhook_secret": {"type": "secret_uri", "description": "Webhook validation token secret"},
                "git_repo_url_secret": {"type": "secret_uri", "description": "Repository URL secret"},
                "cloud_provider": {"type": "string", "values": ["yandex", "aws"], "description": "Target cloud"},
                "region": {"type": "string", "description": "Cloud region"},
                "runner_type": {"type": "string", "values": ["serverless", "apex", "nadir"]},
            },
            "optional_attributes": {
                "tags": {"type": "list(string)", "description": "GitLab runner tags"},
                "cpu": {"type": "cpu", "default": "1"},
                "memory": {"type": "memory", "default": "512MB"},
                "max_concurrent_jobs": {"type": "number", "default": 1},
                "runner_image": {"type": "string", "description": "Container image for the runner"},
                "environment": {"type": "list(string)", "description": "Extra environment variables"},
            },
        },
        "eggsbucket": {
            "location": "Eggs/<name>/config.fly",
            "description": "Group of GitLab projects sharing a single runner configuration.",
            "required_attributes": {
                "gitlab_server": {"type": "string"},
                "group_id": {"type": "number", "description": "GitLab group ID"},
                "gitlab_token_secret": {"type": "secret_uri"},
                "gitlab_webhook_secret": {"type": "secret_uri"},
                "cloud_provider": {"type": "string", "values": ["yandex", "aws"]},
                "region": {"type": "string"},
                "runner_type": {"type": "string", "values": ["serverless", "apex", "nadir"]},
            },
            "optional_attributes": {
                "tags": {"type": "list(string)"},
                "cpu": {"type": "cpu"},
                "memory": {"type": "memory"},
                "project_ids": {"type": "list(number)", "description": "Explicit project IDs within the group"},
            },
        },
        "job": {
            "location": "Jobs/<name>.fly",
            "description": "Internal self-management task run on a dedicated runner.",
            "required_attributes": {
                "schedule": {"type": "cron", "description": "Cron schedule for the job"},
                "script": {"type": "string", "description": "Shell script or command to execute"},
                "cloud_provider": {"type": "string", "values": ["yandex", "aws"]},
                "region": {"type": "string"},
            },
            "optional_attributes": {
                "runner_image": {"type": "string"},
                "cpu": {"type": "cpu"},
                "memory": {"type": "memory"},
                "timeout": {"type": "duration", "default": "1h"},
                "environment": {"type": "list(string)"},
                "secrets": {"type": "list(secret_uri)", "description": "Secrets to inject as env vars"},
            },
        },
        "uglyfox": {
            "location": "UF/config.fly",
            "description": "UglyFox lifecycle and pruning configuration.",
            "nested_blocks": ["pruning", "apex_pool", "nadir_pool", "runners_condition"],
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
