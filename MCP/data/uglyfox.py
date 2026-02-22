"""UglyFox service data for the MCP server."""

from typing import Any

UGLYFOX_CELERY_TASKS: list[dict[str, Any]] = [
    # Health tasks
    {
        "name": "check_runner_health",
        "module": "tasks.health",
        "queue": "uglyfox.health",
        "schedule": "every 60 seconds",
        "description": "Polls all ACTIVE/IDLE/BUSY runners for heartbeat. Updates last_heartbeat and state. Increments failure_count on no response.",
    },
    {
        "name": "collect_runner_metrics",
        "module": "tasks.health",
        "queue": "uglyfox.health",
        "schedule": "every 5 minutes",
        "description": "Collects CPU/memory/job metrics from runners and publishes to cloud monitoring.",
    },
    {
        "name": "identify_unhealthy_runners",
        "module": "tasks.health",
        "queue": "uglyfox.health",
        "schedule": "every 2 minutes",
        "description": "Finds runners with failure_count exceeding threshold or stale heartbeat. Marks them FAILED and enqueues pruning.",
    },
    # Lifecycle tasks
    {
        "name": "manage_apex_nadir_pools",
        "module": "tasks.lifecycle",
        "queue": "uglyfox.lifecycle",
        "schedule": "every 5 minutes",
        "description": "Evaluates Apex/Nadir pool sizes per egg against UF config targets. Triggers promotions or demotions as needed.",
    },
    {
        "name": "promote_nadir_to_apex",
        "module": "tasks.lifecycle",
        "queue": "uglyfox.lifecycle",
        "description": "Promotes a NADIR (dormant) runner to APEX (active) state. Sends Celery task to MotherGoose to reconfigure the runner.",
        "triggered_by": ["manage_apex_nadir_pools"],
    },
    {
        "name": "demote_apex_to_nadir",
        "module": "tasks.lifecycle",
        "queue": "uglyfox.lifecycle",
        "description": "Demotes an APEX runner to NADIR state when pool is oversized or runner is idle beyond threshold.",
        "triggered_by": ["manage_apex_nadir_pools"],
    },
    {
        "name": "transition_runner_state",
        "module": "tasks.lifecycle",
        "queue": "uglyfox.lifecycle",
        "description": "Generic state machine transition for a runner. Validates allowed transitions and writes audit_log.",
        "triggered_by": ["promote_nadir_to_apex", "demote_apex_to_nadir", "prune_failed_runners"],
    },
    # Pruning tasks
    {
        "name": "evaluate_pruning_policies",
        "module": "tasks.pruning",
        "queue": "uglyfox.pruning",
        "schedule": "every 10 minutes",
        "description": "Reads UF/config.fly pruning rules from DB. Identifies runners that violate max_age, max_failures, or idle_timeout policies.",
    },
    {
        "name": "prune_failed_runners",
        "module": "tasks.pruning",
        "queue": "uglyfox.pruning",
        "description": "Terminates runners in FAILED state that exceed the failure threshold. Sends terminate_runner task to MotherGoose.",
        "triggered_by": ["evaluate_pruning_policies", "identify_unhealthy_runners"],
    },
    {
        "name": "prune_old_runners",
        "module": "tasks.pruning",
        "queue": "uglyfox.pruning",
        "description": "Terminates runners that exceed max_age_hours from UF config. Ensures runner fleet stays fresh.",
        "triggered_by": ["evaluate_pruning_policies"],
    },
    {
        "name": "terminate_runner",
        "module": "tasks.pruning",
        "queue": "uglyfox.pruning",
        "description": "Sends a terminate_runner Celery task to MotherGoose via SQS/YMQ. UglyFox does not call OpenTofu directly.",
        "triggered_by": ["prune_failed_runners", "prune_old_runners"],
    },
]

UGLYFOX_CONFIG_SCHEMA: dict[str, Any] = {
    "description": "Schema for UF/config.fly — UglyFox lifecycle and pruning configuration.",
    "file_location": "Nest/UF/config.fly",
    "top_level_block": "uglyfox",
    "blocks": {
        "pruning": {
            "description": "Global pruning policies applied to all runners unless overridden per-egg.",
            "attributes": {
                "max_age_hours": {"type": "number", "description": "Maximum runner age before forced termination"},
                "max_failures": {"type": "number", "description": "Consecutive failure count before pruning"},
                "idle_timeout_minutes": {"type": "number", "description": "Minutes idle before APEX→NADIR demotion"},
                "check_interval_seconds": {"type": "number", "default": 60, "description": "Health check polling interval"},
            },
        },
        "runners_condition": {
            "description": "Per-egg overrides for pruning conditions.",
            "attributes": {
                "egg_name": {"type": "string", "description": "Target egg name"},
                "max_age_hours": {"type": "number", "description": "Override global max_age_hours"},
                "max_failures": {"type": "number", "description": "Override global max_failures"},
            },
        },
        "apex_pool": {
            "description": "Apex (active) runner pool configuration.",
            "attributes": {
                "min_size": {"type": "number", "description": "Minimum number of APEX runners to maintain"},
                "max_size": {"type": "number", "description": "Maximum APEX runners allowed"},
                "scale_up_threshold": {"type": "number", "description": "Job queue depth to trigger scale-up"},
            },
        },
        "nadir_pool": {
            "description": "Nadir (dormant/warm standby) runner pool configuration.",
            "attributes": {
                "min_size": {"type": "number", "description": "Minimum NADIR runners to keep warm"},
                "max_size": {"type": "number", "description": "Maximum NADIR runners allowed"},
                "warmup_time_seconds": {"type": "number", "description": "Expected time to promote NADIR→APEX"},
            },
        },
    },
    "example": """
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
    egg_name     = "my-project"
    max_failures = 3
  }
}
""",
}

UGLYFOX_ENV_VARS: list[dict[str, Any]] = [
    {"name": "UGLYFOX_DB_BACKEND", "type": "str", "values": ["ydb", "dynamodb"], "description": "Database backend"},
    {"name": "UGLYFOX_YDB_ENDPOINT", "type": "str", "description": "YDB gRPC endpoint"},
    {"name": "UGLYFOX_YDB_DATABASE", "type": "str", "description": "YDB database path"},
    {"name": "UGLYFOX_DYNAMODB_TABLE_PREFIX", "type": "str", "description": "DynamoDB table name prefix"},
    {"name": "UGLYFOX_AWS_REGION", "type": "str", "description": "AWS region"},
    {"name": "UGLYFOX_CELERY_BROKER_URL", "type": "str", "description": "Celery broker URL (SQS or YMQ)"},
    {"name": "UGLYFOX_CELERY_RESULT_BACKEND", "type": "str", "description": "Celery result backend URL"},
    {"name": "UGLYFOX_MOTHERGOOSE_QUEUE", "type": "str", "description": "Queue name for sending tasks to MotherGoose"},
    {"name": "UGLYFOX_HEALTH_CHECK_INTERVAL", "type": "int", "default": 60, "description": "Runner health check interval in seconds"},
    {"name": "UGLYFOX_LOG_LEVEL", "type": "str", "default": "INFO", "values": ["DEBUG", "INFO", "WARNING", "ERROR"]},
    {"name": "UGLYFOX_YC_FOLDER_ID", "type": "str", "description": "Yandex Cloud folder ID"},
]
