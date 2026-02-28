"""Kiro steering and project context data for the Polar Gosling MCP server."""

STEERING_PRODUCT = """# Product: Polar Gosling GitOps Runner Orchestration

A multi-cloud GitOps CI/CD runner orchestration system that automatically provisions, manages, and terminates GitLab runners across Yandex Cloud and AWS.

## Services

- **MotherGoose**: Primary orchestration server. Handles GitLab webhook processing, Git sync (periodic + event-driven), runner deployment via OpenTofu, and exposes a REST API.
- **UglyFox**: Lifecycle management worker. Monitors runner health, enforces pruning policies, and manages Apex (active) / Nadir (dormant) runner pool transitions.
- **Rift**: Shared cache storage and remote docker context execution vm.

## Key Concepts

- **Egg**: A runner configuration unit defined in the Nest Git repository (`Eggs/` directory), parsed from `.fly` config files.
- **EggsBucket**: A group of multiple managed repositories configured together with shared runner configuration.
- **Jobs**: Internal self-management tasks defined in the Nest `Jobs/` directory as `.fly` files. Executed by MotherGoose on dedicated runners. Used for secret rotation, Nest updates, runner image updates, and other recurring maintenance tasks.
- **Nest Repository**: Single source of truth Git repo containing `Eggs/`, `Jobs/`, `UF/`, and `MG/` directories.
- **Apex/Nadir pools**: Active vs dormant runner states managed by UglyFox.
- **Runners**: VMs or serverless containers deployed via OpenTofu + Jinja2 templates.

## Nest Repository Structure

```
Nest/
├── Eggs/
│   ├── my-project/
│   │   └── config.fly          # Single Egg: one managed repo/project
│   └── my-group-bucket/
│       └── config.fly          # EggsBucket: multiple repos with shared runner config
├── Jobs/
│   └── {job_name}.fly          # Internal self-management tasks
├── UF/
│   └── config.fly              # UglyFox pruning policies and runner lifecycle rules
└── MG/
    └── config.fly              # MotherGoose infra config: API Gateway, queues, triggers, containers
```

### .fly File Types

- `Eggs/*/config.fly` — defines `egg` or `eggsbucket` block: runner type, resources, tags, GitLab server, cloud provider, secrets
- `Jobs/*.fly` — defines a self-management job: schedule (cron), runner config, script
- `UF/config.fly` — defines pruning policies: failure thresholds, max age, Apex/Nadir pool rules
- `MG/config.fly` — defines MotherGoose infrastructure: API Gateway, serverless containers, message queues, cloud triggers

Secret references in `.fly` files use URI schemes: `yc-lockbox://{id}/{key}`, `aws-sm://{name}/{key}`, `vault://{path}/{key}`

## Cloud Targets

- Yandex Cloud: YDB, YMQ, Lockbox secrets, Serverless Containers, Compute VMs
- AWS: DynamoDB, SQS, Secrets Manager, ECS Fargate, EC2
"""

STEERING_STRUCTURE = """# Project Structure

## Repository Layout

```
Polar-Gosling-MotherGoose/     # Backend services (Python)
Polar-Gosling-Compute-Module/  # Terraform/OpenTofu compute module
Polar-Gosling/                 # Gosling CLI (Go) — main monorepo
```

## Polar-Gosling-MotherGoose

```
mothergoose/
  src/app/
    core/          # Celery app setup, config (pydantic-settings)
    db/            # DB clients: YDB + DynamoDB connections, manage_db
    model/         # Pydantic models: runners, audit, opentofu, gosling, base
    repository/    # (Reserved for repository pattern)
    routers/       # FastAPI route handlers: eggs, runners, webhooks, internal, health, binaries
    schema/        # Schema definitions: API, DB tables, YDB, DynamoDB, payload, tofu, URL
    services/      # Business logic: git sync, runner orchestration, secret manager,
                   #   opentofu config/binary, gosling binary/manager, egg service,
                   #   deployment plans, fly parser, s3 artifact cache
    tasks/         # Celery tasks: git_sync, runners, webhooks, maintenance
    templates/     # Jinja2 templates for OpenTofu (.j2 files)
    types/         # YDB type definitions
    util/          # Helpers: logging, exceptions, generators, model converters
    main.py        # FastAPI app entry point
    celery_worker.py  # Celery worker entry point
  tests/
  pyproject.toml
  uv.lock

uglyfox/
  src/app/
    core/          # Celery app setup, config
    db/            # DB clients (mirrors mothergoose structure)
    model/         # Pydantic models (shared schema with mothergoose)
    schema/        # DB table schemas
    tasks/         # Celery tasks: health, pruning, lifecycle
    types/         # YDB type definitions
    util/          # Helpers: logging, generator
    celery_worker.py
  tests/
  pyproject.toml
  uv.lock

Dockerfile.mg      # MotherGoose container image
Dockerfile.uf      # UglyFox container image
Makefile           # Build/test automation for both services
```

## Polar-Gosling-Compute-Module

```
aws_resources.tf   # AWS EC2 + ECS resources
aws_variables.tf   # AWS-specific input variables
yc_resources.tf    # Yandex Cloud VM + Serverless Container resources
yc_variables.tf    # YC-specific input variables
data.tf            # Data sources
locals.tf          # Computed locals
output.tf          # Module outputs: hostname, public_ip, private_ip, id
versions.tf        # Provider version constraints
tests/
  tests_aws_ecs_base/
  tests_aws_vm_base/
  tests_yc_serverless_base/
  tests_yc_vm_base/
scripts/
  pre-commit-hook.sh / .ps1
```

## Polar-Gosling (Go — Gosling CLI)

```
internal/
  cli/         # Command implementations: init, add, validate, deploy, rollback, parse, status
  parser/      # .fly file parser and AST
  mothergoose/ # MotherGoose API client
  cloud/       # Yandex Cloud + AWS SDK integrations
  gitlab/      # GitLab Go SDK integration
main.go
```

## Conventions

- Source code lives under `src/app/` in both Python services
- Tests live in `tests/` at the service root, discovered by pytest
- Config is always via environment variables with service-prefix (`MOTHERGOOSE_*`, `UGLYFOX_*`)
- Shared concepts (models, schemas, DB clients) are duplicated between services — not shared as a library
- Jinja2 templates for OpenTofu are in `mothergoose/src/app/templates/` with `.j2` extension
- Terraform test files use `.tftest.hcl` extension (native OpenTofu test framework)
- All new feature development happens in the `dev-new-features` git worktree (MotherGoose repo)
"""

STEERING_TECH = """# Tech Stack

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
`make mg-tox-all` and `make uf-tox-all` spin up real containers (YDB + LocalStack) via testcontainers.
Always wait for the command to fully complete before running any other test command.
Running multiple test suites in parallel will cause YDB connection conflicts and test failures.

### Dependency Management
```bash
uv sync --all-groups      # Install all dependency groups
uv version --bump patch   # Bump patch version (or minor/major)
```
"""
