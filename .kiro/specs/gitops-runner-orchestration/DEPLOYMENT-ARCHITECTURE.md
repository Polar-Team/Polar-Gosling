# Deployment Architecture Quick Reference

## Critical Distinction: Bootstrap vs Runtime

The Polar Gosling system uses **two different deployment mechanisms** for different phases:

### 1. Bootstrap Phase (One-Time Setup)

**Tool**: Gosling CLI  
**Mechanism**: Native Go SDKs (Yandex Cloud Go SDK, AWS SDK for Go v2)  
**Purpose**: Deploy core infrastructure  
**Frequency**: Once during initial setup, rarely updated  
**Command**: `gosling deploy --cloud yandex`

**What Gets Deployed**:
- MotherGoose backend (FastAPI + Celery)
- UglyFox backend (Celery workers)
- Databases (YDB/DynamoDB)
- Message queues (YMQ/SQS)
- API Gateway
- S3 buckets for state storage

**Why Go SDKs?**
- Type-safe, idiomatic code
- Complex one-time infrastructure setup
- Direct cloud provider integration
- Better error handling for bootstrap operations

---

### 2. Runtime Phase (Continuous Operations)

**Tool**: MotherGoose Backend  
**Mechanism**: OpenTofu with Jinja2 templates  
**Purpose**: Deploy individual runners (VMs and serverless containers)  
**Frequency**: Continuously, triggered by GitLab webhooks  
**Process**: Webhook → MotherGoose → Render Jinja2 Templates → Execute OpenTofu → Deploy Runner

**What Gets Deployed**:
- VM runners (persistent GitLab Runner Agents)
- Serverless container runners (ephemeral, 60-minute limit)
- Network configurations
- Security groups
- Cloud-init scripts

**Why OpenTofu?**
- Declarative infrastructure as code
- State management in S3
- Idempotent deployments
- Rollback capability via deployment plans
- Template-based configuration generation

---

## Deployment Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    BOOTSTRAP PHASE (One-Time)               │
│                                                             │
│  Gosling CLI → Go SDKs → Bootstrap Infrastructure          │
│                           (MotherGoose, UglyFox, etc.)     │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ Enables
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   RUNTIME PHASE (Continuous)                │
│                                                             │
│  GitLab Webhook → MotherGoose → Jinja2 Templates →        │
│  OpenTofu → Runners (VMs & Containers)                     │
└─────────────────────────────────────────────────────────────┘
```

---

## Common Misconceptions

### ❌ Misconception 1: "Cloud SDKs are used for runner deployment"
**✅ Reality**: Cloud SDKs are ONLY used for:
- One-time bootstrap infrastructure deployment
- Cloud-native services (Timer Triggers, EventBridge Scheduler)
- IAM authentication and secret retrieval

**Runner deployment uses OpenTofu**, not cloud SDKs directly.

---

### ❌ Misconception 2: "Gosling CLI deploys runners"
**✅ Reality**: Gosling CLI has two distinct modes:
- **Bootstrap mode** (`gosling deploy`): One-time infrastructure setup using Go SDKs
- **Runner mode** (`gosling runner`): Manages GitLab Runner Agent lifecycle on deployed runners

**MotherGoose deploys runners**, not Gosling CLI.

---

### ❌ Misconception 3: "OpenTofu is only for state management"
**✅ Reality**: OpenTofu is the PRIMARY deployment mechanism for runners. It:
- Generates infrastructure configurations from Jinja2 templates
- Provisions VMs and serverless containers
- Manages infrastructure state in S3
- Enables declarative, idempotent deployments

---

### ❌ Misconception 4: "The system uses Terraform"
**✅ Reality**: The system uses **OpenTofu**, not Terraform. This avoids license issues with HashiCorp's BSL license change. Always refer to "OpenTofu" in documentation and code.

---

## When to Use What

| Scenario | Tool | Mechanism |
|----------|------|-----------|
| Initial infrastructure setup | Gosling CLI | Go SDKs |
| Deploy MotherGoose/UglyFox | Gosling CLI | Go SDKs |
| Configure Timer Triggers | Gosling CLI | Go SDKs |
| Deploy a runner for a GitLab job | MotherGoose | OpenTofu |
| Scale runner pool | MotherGoose | OpenTofu |
| Terminate a runner | MotherGoose | OpenTofu |
| Update runner configuration | MotherGoose | OpenTofu |

---

## Code References

### Bootstrap (Go SDKs)
- Location: `Polar-Gosling/internal/deployer/`
- Uses: `github.com/yandex-cloud/go-sdk`, `github.com/aws/aws-sdk-go-v2`

### Runtime (OpenTofu)
- Location: `Polar-Gosling-MotherGoose/dev-new-features/mothergoose/src/app/services/opentofu_binary.py`
- Templates: `Polar-Gosling-MotherGoose/dev-new-features/mothergoose/src/app/templates/tofu_*.j2`
- Task: `Polar-Gosling-MotherGoose/dev-new-features/mothergoose/src/app/tasks/runners.py`

---

## Summary

**Remember**: 
- **Gosling CLI + Go SDKs** = Bootstrap infrastructure (once)
- **MotherGoose + OpenTofu** = Deploy runners (continuously)
- **Cloud SDKs ≠ Runner deployment**
