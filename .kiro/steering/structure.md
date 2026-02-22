# Project Structure

## Repository Layout

This workspace contains three repositories:

```
Polar-Gosling-MotherGoose/     # Backend services (Python)
Polar-Gosling-Compute-Module/  # Terraform/OpenTofu compute module
Polar-Gosling/                 # Gosling CLI (Go) — main monorepo
```

## Polar-Gosling-MotherGoose

```
mothergoose/                   # MotherGoose service
  src/app/
    core/                      # Celery app setup, config (pydantic-settings)
    db/                        # DB clients: YDB + DynamoDB connections, manage_db
    model/                     # Pydantic models: runners, audit, opentofu, gosling, base
    repository/                # (Reserved for repository pattern)
    routers/                   # FastAPI route handlers: eggs, runners, webhooks, internal, health, binaries
    schema/                    # Schema definitions: API, DB tables, YDB, DynamoDB, payload, tofu, URL
    services/                  # Business logic: git sync, runner orchestration, secret manager,
                               #   opentofu config/binary, gosling binary/manager, egg service,
                               #   deployment plans, fly parser, s3 artifact cache
    tasks/                     # Celery tasks: git_sync, runners, webhooks, maintenance
    templates/                 # Jinja2 templates for OpenTofu (.j2 files)
    types/                     # YDB type definitions
    util/                      # Helpers: logging, exceptions, generators, model converters
    main.py                    # FastAPI app entry point
    celery_worker.py           # Celery worker entry point
  tests/
  pyproject.toml
  uv.lock

uglyfox/                       # UglyFox service
  src/app/
    core/                      # Celery app setup, config
    db/                        # DB clients (mirrors mothergoose structure)
    model/                     # Pydantic models (shared schema with mothergoose)
    schema/                    # DB table schemas
    tasks/                     # Celery tasks: health, pruning, lifecycle
    types/                     # YDB type definitions
    util/                      # Helpers: logging, generator
    celery_worker.py           # Celery worker entry point
  tests/
  pyproject.toml
  uv.lock

Dockerfile.mg                  # MotherGoose container image
Dockerfile.uf                  # UglyFox container image
Makefile                       # Build/test automation for both services
```

## Polar-Gosling-Compute-Module

```
aws_resources.tf               # AWS EC2 + ECS resources
aws_variables.tf               # AWS-specific input variables
yc_resources.tf                # Yandex Cloud VM + Serverless Container resources
yc_variables.tf                # YC-specific input variables
data.tf                        # Data sources (AMI SSM param, YC image, client config)
locals.tf                      # Computed locals: labels, container JSON, output helpers
output.tf                      # Module outputs: hostname, public_ip, private_ip, id
versions.tf                    # Provider version constraints
tests/
  tests_aws_ecs_base/          # Terraform native tests (.tftest.hcl)
  tests_aws_vm_base/
  tests_yc_serverless_base/
  tests_yc_vm_base/
scripts/
  pre-commit-hook.sh / .ps1    # Pre-commit validation scripts
```

## Polar-Gosling (Go — Gosling CLI)

```
internal/
  cli/                         # Command implementations: init, add, validate, deploy, rollback, parse, status
  parser/                      # .fly file parser and AST
  mothergoose/                 # MotherGoose API client
  cloud/                       # Yandex Cloud + AWS SDK integrations
  gitlab/                      # GitLab Go SDK integration
main.go
```

## Conventions

- Source code lives under `src/app/` in both Python services (setuptools `find_packages where=["src"]`)
- Tests live in `tests/` at the service root, discovered by pytest
- Config is always via environment variables with service-prefix (`MOTHERGOOSE_*`, `UGLYFOX_*`)
- Shared concepts (models, schemas, DB clients) are duplicated between services — not shared as a library
- Jinja2 templates for OpenTofu are in `mothergoose/src/app/templates/` with `.j2` extension
- Terraform test files use `.tftest.hcl` extension (native OpenTofu test framework)
- All new feature development happens in the `dev-new-features` git worktree (MotherGoose repo)
