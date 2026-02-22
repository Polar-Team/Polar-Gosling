# Tech Stack

## Language & Runtime
- Python 3.10–3.13 (supports all four versions via tox)
- Package manager: `uv`
- Go (Gosling CLI)

## Frameworks & Libraries
- **FastAPI** + **uvicorn**: REST API (MotherGoose only)
- **Celery** + **Kombu**: Async task queue (both services)
- **Pydantic v2**: Data models and settings (`pydantic-settings`)
- **YDB SDK** (`ydb`): Yandex Cloud database
- **boto3** / **aioboto3**: AWS SDK (DynamoDB, SQS, Secrets Manager)
- **GitPython**: Git repository operations
- **Jinja2**: OpenTofu template rendering
- **tofupy**: Python wrapper for OpenTofu CLI
- **hypothesis**: Property-based testing
- **pytest** + **pytest-asyncio**: Test framework
- **testcontainers**: Integration test containers

## Code Quality Tools
- **black** + **isort**: Formatting (line length: 120, isort profile: black)
- **flake8** + **pylint**: Linting (pylint must score 10/10)
- **mypy**: Static type checking (strict — `disallow_untyped_defs = true`)
- **tox**: Multi-environment test runner

## Infrastructure
- **OpenTofu** (>= 1.3.5): IaC for runner provisioning
- **Terraform module** (`Polar-Gosling-Compute-Module`): Reusable compute module for AWS (EC2, ECS) and Yandex Cloud (VM, Serverless Container)
- Providers: `yandex >= 0.170.0`, `aws >= 4.66`, `random >= 3.4.3`

## Common Commands

All commands run from `Polar-Gosling-MotherGoose/` root via Makefile, or directly inside each service directory.

### MotherGoose
```bash
make mg-test              # Run pytest
make mg-lint              # flake8 + pylint via tox
make mg-type              # mypy via tox
make mg-format            # Apply black + isort
make mg-format-check      # Check formatting (CI)
make mg-tox-all           # Full tox suite (all Python versions + format + style + type)

# Direct (from mothergoose/)
uv run pytest -v
uv run pytest --cov=app --cov-report=html
uv run pytest -v --hypothesis-show-statistics
uv run tox
```

### UglyFox
```bash
make uf-test              # Run pytest
make uf-lint              # flake8 + pylint via tox
make uf-type              # mypy via tox
make uf-format            # Apply black + isort
make uf-tox-all           # Full tox suite

# Direct (from uglyfox/)
uv run pytest -v
uv run tox
```

### Compute Module (Terraform)
```bash
make pre-commit           # Run pre-commit hooks
# Tests use native Terraform test framework (.tftest.hcl files)
```

### Critical: Test Execution Warning

`make mg-tox-all` and `make uf-tox-all` spin up real containers (YDB + LocalStack) via testcontainers. **Always wait for the command to fully complete before running any other test command.** Running multiple test suites in parallel will cause YDB connection conflicts and test failures.

### Dependency Management
```bash
uv sync --all-groups      # Install all dependency groups
uv version --bump patch   # Bump patch version (or minor/major)
```

## UML Diagrams

### Architecture Diagram

```mermaid
graph TD
    Nest["Nest Git Repository\nEggs/ Jobs/ UF/ MG/\n(.fly files)"]

    subgraph MG["MotherGoose"]
        MGAPI["FastAPI\n/eggs /runners /webhooks\n/internal /health /binaries"]
        MGWorker["Celery Worker\ntasks: git_sync, runners\nwebhooks, maintenance"]
        MGServices["Services\nGitSync · FlyParser · SecretManager\nRunnerOrchestration · OpenTofuConfig\nGoslingBinary · DeploymentPlan"]
        MGAPI --> MGWorker
        MGWorker --> MGServices
    end

    subgraph UF["UglyFox"]
        UFWorker["Celery Worker\ntasks: health, pruning, lifecycle"]
    end

    subgraph Runners["Runners"]
        YCRunner["Yandex Cloud\nServerless Container\nCompute VM"]
        AWSRunner["AWS\nECS Fargate\nEC2"]
    end

    subgraph DB["YDB / DynamoDB"]
        Tables["runners · egg_configs · sync_history\ndeployment_plans · audit_logs\ntofu_versions · gosling_version"]
    end

    subgraph Secrets["Secret Backends"]
        SecretB["YC Lockbox · AWS SM · Vault\nyc-lockbox:// aws-sm:// vault://"]
    end

    Nest -->|"Git Sync\n5min / push webhook"| MG
    GitLab -->|"POST /webhooks/gitlab\nX-Gitlab-Token"| MGAPI
    CloudTrigger["Cloud Trigger\nYC Timer / AWS EventBridge"] -->|"POST /internal/sync-git"| MGAPI
    MGServices -->|"OpenTofu + Jinja2"| Runners
    MGWorker -->|"Celery Tasks\nSQS / YMQ"| UFWorker
    MG <-->|"read/write"| DB
    UF <-->|"read/write"| DB
    MGServices -->|"resolve secrets"| Secrets
    Rift["Rift\nRemote cache server\nDocker/Podman context"] --- Runners
```

---

### Class Diagram — Core Models

```mermaid
classDiagram
    class PydanticBaseModelORM {
        <<abstract>>
        model_config: frozen=True
    }

    class Runner {
        +id: str
        +egg_name: str
        +type: RunnerType
        +state: RunnerState
        +cloud_provider: CloudProvider
        +region: str
        +gitlab_runner_id: int
        +deployed_from_commit: str
        +created_at: datetime
        +updated_at: datetime
        +last_heartbeat: datetime
        +failure_count: int
        +metadata: dict
        +to_storage_dict() dict
    }

    class EggConfig {
        +id: str
        +name: str
        +project_id: int
        +group_id: int
        +config: dict
        +git_commit: str
        +git_repo_url_secret: str
        +gitlab_token_secret_uri: str
        +gitlab_webhook_secret_uri: str
        +synced_at: datetime
        +created_at: datetime
        +updated_at: datetime
        +to_storage_dict() dict
    }

    class SyncHistory {
        +id: str
        +git_commit: str
        +sync_type: str
        +status: SyncStatus
        +changes_detected: int
        +eggs_synced: int
        +jobs_synced: int
        +uf_config_synced: bool
        +error_message: str
        +synced_at: datetime
        +duration_ms: int
    }

    class DeploymentPlan {
        +id: str
        +egg_name: str
        +plan_type: str
        +config_hash: str
        +status: DeploymentStatus
        +plan_binary: bytes
        +rollback_plan_id: str
        +created_at: datetime
        +applied_at: datetime
        +metadata: dict
        +to_storage_dict() dict
    }

    class RunnerState {
        <<enumeration>>
        ACTIVE
        IDLE
        BUSY
        FAILED
        TERMINATED
    }

    class RunnerType {
        <<enumeration>>
        SERVERLESS
        APEX
        NADIR
    }

    class CloudProvider {
        <<enumeration>>
        YANDEX
        AWS
    }

    class DeploymentStatus {
        <<enumeration>>
        PENDING
        APPLIED
        ROLLED_BACK
        FAILED
    }

    class SyncStatus {
        <<enumeration>>
        SUCCESS
        FAILED
    }

    PydanticBaseModelORM <|-- Runner
    PydanticBaseModelORM <|-- EggConfig
    PydanticBaseModelORM <|-- SyncHistory
    PydanticBaseModelORM <|-- DeploymentPlan
    Runner --> RunnerState
    Runner --> RunnerType
    Runner --> CloudProvider
    DeploymentPlan --> DeploymentStatus
    SyncHistory --> SyncStatus
```

---

### Class Diagram — Services

```mermaid
classDiagram
    class GitSyncService {
        +sync_from_git(commit: str) SyncHistory
        +parse_eggs(path: str) list
        +parse_jobs(path: str) list
        +parse_uf_config(path: str) dict
    }

    class FlyParserService {
        +parse_egg(path: str) dict
        +parse_job(path: str) dict
        +parse_uf_config(path: str) dict
        -_call_gosling_parse(path, type) dict
    }

    class SecretManager {
        +get_secret(uri: str) str
        -_yc_lockbox(uri: str) str
        -_aws_sm(uri: str) str
        -_vault(uri: str) str
    }

    class SecretCache {
        +get(uri: str) str
        +set(uri: str, value: str) None
        -ttl: int
    }

    class RunnerOrchestrationService {
        +deploy_runner(egg: EggConfig) Runner
        +terminate_runner(runner_id: str) None
        +determine_runner_type(egg: EggConfig) RunnerType
    }

    class OpenTofuConfiguration {
        +render_templates(settings: TofuSetting) dict
        +generate_plan() bytes
        +apply_plan(plan: bytes) None
    }

    class OpenTofuBinary {
        <<abstract>>
        +store_downloaded_bin() tuple
        #_download_and_extract(extract_to: str) None
    }

    class OpenTofuDownloadGithub {
        +store_downloaded_bin() tuple
    }

    class OpenTofuDownloadFromOtherSource {
        +store_downloaded_bin() tuple
    }

    class GoslingBinary {
        <<abstract>>
        +store_downloaded_bin() tuple
        #_download_and_extract(extract_to: str) None
    }

    class GoslingDownloadGithub {
        +store_downloaded_bin() tuple
    }

    class GoslingDownloadFromOtherSource {
        +store_downloaded_bin() tuple
    }

    class GoslingBinaryManager {
        +mount_s3_binaries() None
        +get_active_binary_path() str
        +verify_and_activate(version: str) None
    }

    class DeploymentPlanService {
        +create_plan(egg_name: str, config: dict) DeploymentPlan
        +apply_plan(plan_id: str) None
        +rollback(plan_id: str) None
    }

    GitSyncService --> FlyParserService
    FlyParserService --> GoslingBinaryManager
    SecretManager --> SecretCache
    RunnerOrchestrationService --> OpenTofuConfiguration
    RunnerOrchestrationService --> DeploymentPlanService
    OpenTofuBinary <|-- OpenTofuDownloadGithub
    OpenTofuBinary <|-- OpenTofuDownloadFromOtherSource
    GoslingBinary <|-- GoslingDownloadGithub
    GoslingBinary <|-- GoslingDownloadFromOtherSource
```

---

### Sequence Diagram — Webhook Processing

```mermaid
sequenceDiagram
    participant GL as GitLab
    participant API as MotherGoose API
    participant CW as Celery Worker
    participant DB as YDB/DynamoDB
    participant SM as SecretManager
    participant TF as OpenTofu

    GL->>API: POST /webhooks/gitlab (X-Gitlab-Token)
    API->>API: validate token, match Egg
    API->>CW: enqueue webhook task (SQS/YMQ)
    API-->>GL: 202 Accepted

    CW->>DB: fetch EggConfig by egg_name
    DB-->>CW: EggConfig

    CW->>SM: resolve secret URIs
    SM-->>CW: secrets

    CW->>TF: render Jinja2 templates
    CW->>TF: tofu plan + apply
    TF-->>CW: runner deployed

    CW->>DB: upsert Runner record
    CW->>DB: write audit_log

    Note over GL,API: Nest repo push → triggers Git sync (not runner deployment)
    Note over GL,API: Egg repo webhook → triggers runner deployment via OpenTofu
```

---

### Sequence Diagram — Git Sync

```mermaid
sequenceDiagram
    participant CT as Cloud Trigger (YC Timer / EventBridge)
    participant API as MotherGoose API
    participant CW as Celery Worker
    participant GP as GitPython
    participant GC as Gosling CLI (subprocess)
    participant DB as YDB/DynamoDB

    CT->>API: POST /internal/sync-git (secret token)
    API->>CW: enqueue git_sync task
    API-->>CT: 202 Accepted

    CW->>GP: clone / pull Nest repo
    GP-->>CW: repo files

    CW->>GC: parse Eggs/*.fly --type=egg
    GC-->>CW: EggConfig JSON

    CW->>GC: parse Jobs/*.fly --type=job
    GC-->>CW: Job JSON

    CW->>GC: parse UF/config.fly --type=uglyfox
    GC-->>CW: UFConfig JSON

    CW->>DB: upsert egg_configs
    CW->>DB: upsert jobs
    CW->>DB: upsert uf_config
    CW->>DB: write sync_history

    Note over CT: Every 5 minutes
    Note over CT: Also triggered on Nest repo push webhook
```

---

### Database Schema

```mermaid
erDiagram
    runners {
        Utf8 id PK
        Utf8 egg_name
        Utf8 type
        Utf8 state
        Utf8 cloud_provider
        Utf8 region
        Int64 gitlab_runner_id
        Utf8 deployed_from_commit
        Utf8 created_at
        Utf8 updated_at
        Utf8 last_heartbeat
        Int64 failure_count
        String metadata
    }

    egg_configs {
        Utf8 id PK
        Utf8 name
        Int64 project_id
        Int64 group_id
        String config
        Utf8 git_commit
        Utf8 git_repo_url_secret
        Utf8 gitlab_token_secret_uri
        Utf8 gitlab_webhook_secret_uri
        Utf8 synced_at
        Utf8 created_at
        Utf8 updated_at
    }

    sync_history {
        Utf8 id PK
        Utf8 git_commit
        Utf8 sync_type
        Utf8 status
        Int64 changes_detected
        Int64 eggs_synced
        Int64 jobs_synced
        Utf8 uf_config_synced
        Utf8 error_message
        Utf8 synced_at
        Int64 duration_ms
    }

    deployment_plans {
        Utf8 id PK
        Utf8 egg_name
        Utf8 plan_type
        Utf8 config_hash
        Utf8 status
        String plan_binary
        Utf8 rollback_plan_id
        Utf8 created_at
        Utf8 applied_at
        String metadata
    }

    audit_logs {
        Utf8 id PK
        Utf8 timestamp
        Utf8 actor
        Utf8 action
        Utf8 resource_type
        Utf8 resource_id
        String details
    }

    tofu_versions {
        Utf8 version_id PK
        Utf8 version
        Utf8 source
        Utf8 downloaded_at
        Utf8 sha256_hash
        Bool active
    }

    gosling_version {
        Utf8 version_id PK
        Utf8 version
        Utf8 source
        Utf8 downloaded_at
        Utf8 sha256_hash
        Bool active
    }

    runners ||--o{ egg_configs : "egg_name → name"
    deployment_plans ||--o{ egg_configs : "egg_name → name"
```

---

### Component Diagram

```mermaid
graph TB
    subgraph MGRepo["Polar-Gosling-MotherGoose (Python)"]
        subgraph MGApp["MotherGoose Service"]
            FastAPI["FastAPI\nrouters: eggs, runners\nwebhooks, internal\nhealth, binaries"]
            CeleryMG["Celery Worker\ntasks: git_sync, runners\nwebhooks, maintenance"]
            ServicesMG["services/\ngit_sync_service\nrunner_orchestration\nsecret_manager\nfly_parser\nopentofu_binary/config\ngosling_binary/manager\negg_service\ndeployment_plan_service"]
            ModelsMG["model/\nrunners_models\naudit_models\nopentofu_models\ngosling_models"]
            DBMG["db/\nydb_connection\ndynamodb_connection\nmanage_db"]
        end

        subgraph UFApp["UglyFox Service"]
            CeleryUF["Celery Worker\ntasks: health\npruning\nlifecycle"]
            ServicesUF["services/ db/ model/\n(mirrors MG structure)"]
        end
    end

    subgraph CMRepo["Polar-Gosling-Compute-Module (OpenTofu)"]
        TFModule["Reusable compute module\naws_resources.tf\nyc_resources.tf\nlocals.tf · output.tf\nProvisions: EC2, ECS\nYC VM, YC Serverless"]
    end

    subgraph GoRepo["Polar-Gosling (Go)"]
        GoslingCLI["Gosling CLI\ncommands: init, add egg/job\nvalidate, deploy, rollback\nparse (JSON output)\nstatus (via MG API)"]
    end

    FastAPI --> CeleryMG
    CeleryMG --> ServicesMG
    ServicesMG --> ModelsMG
    ServicesMG --> DBMG
    ServicesMG -->|"tofupy"| TFModule
    ServicesMG -->|"subprocess"| GoslingCLI
    CeleryMG -->|"SQS/YMQ"| CeleryUF
    CeleryUF --> ServicesUF
```
