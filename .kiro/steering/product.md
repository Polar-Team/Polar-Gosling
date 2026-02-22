# Product: Polar Gosling GitOps Runner Orchestration

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

The Nest is the single source of truth GitOps repo. All configuration is written in `.fly` files (HCL-like DSL with stronger typing).

```
Nest/
├── Eggs/
│   ├── my-project/
│   │   └── config.fly          # Single Egg: one managed repo/project
│   └── my-group-bucket/
│       └── config.fly          # EggsBucket: multiple repos with shared runner config
├── Jobs/
│   └── {job_name}.fly          # Internal self-management tasks (secret rotation, updates, etc.)
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
