# Implementation Plan: GitOps Runner Orchestration

## Overview

This implementation plan breaks down the GitOps Runner Orchestration system into discrete, manageable tasks. The system will be built incrementally, with each task building on previous work. The implementation follows a true GitOps architecture where the Nest Git repository is the single source of truth, with MotherGoose using OpenTofu (not cloud SDKs) for ALL runner deployment.

**Document Status**: Tasks document refreshed to align with current requirements and design specifications. All task descriptions updated to reflect accurate architecture details and implementation approach.

## Critical Architecture Notes

- **Bootstrap Phase (One-Time)**: Gosling CLI uses Go SDKs to deploy MotherGoose, UglyFox, databases, queues (one-time setup)
- **System Triggers (Internal)**: MotherGoose uses Python SDKs to create cloud triggers for internal operations (Timer Triggers for Yandex Cloud, EventBridge Scheduler for AWS)
- **ALL Runners (Eggs + Jobs)**: MotherGoose uses OpenTofu with Jinja2 templates for ALL runner deployment (both Egg runners and Job runners)
- **Jobs Folder**: Creates GitLab scheduled pipelines + runner tokens + webhooks for Nest repo; when pipeline fires → GitLab webhook (X-Gitlab-Token) → MotherGoose → Celery Task (SQS/YMQ) → OpenTofu → Deploy Runner (same flow as Eggs)
- **Job Runner Constraints**: 10-minute time limit (vs 60 minutes for Egg serverless runners), cannot use Rift servers
- **Gosling CLI Status**: Uses MotherGoose API client, NOT direct database access
- **Development Workflow**: All new features developed in "dev-new-features" worktree ONLY
- **Validation**: All implementations must pass `make mg-tox-all` from "dev-new-features" root before completion
- **Dependency Management**: Use `uv` for Python package management (`uv sync --all-groups`, `uv run pytest`)
- **Test Execution**: Agent must wait for test completion (YDB static port mapping requires serialized execution)

## Tasks

- [x] 1. Project Structure Setup
  - Create Polar-Gosling workspace directory for Gosling CLI
  - Set up Go module with go.mod and go.sum
  - Create Dockerfile for Gosling CLI (serverless runner container)
  - Configure pyproject.toml for MotherGoose (Python 3.10-3.13 compatibility via tox)
  - Configure uv for dependency management
  - _Requirements: 3.1_

- [x] 2. Fly Language Parser and Core Infrastructure
  - Implement HCL-based parser with stronger typing
  - Create AST representation for .fly files
  - Implement validation engine
  - Support both "egg" and "eggsbucket" block types
  - Support secret URI parsing (yc-lockbox://, aws-sm://, vault://)
  - _Requirements: 1.6, 1.7, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9_

- [x] 2.1 Write property test for Fly parser round-trip
  - **Property 1: Fly Parser Round-Trip Consistency**
  - **Validates: Requirements 2.1, 2.4**

- [x] 2.2 Write property test for type error detection
  - **Property 2: Fly Parser Type Error Detection**
  - **Validates: Requirements 2.5**

- [x] 2.3 Write property test for nested block support
  - **Property 3: Fly Parser Nested Block Support**
  - **Validates: Requirements 2.6**


- [x] 2.4 Write property test for variable interpolation
  - **Property 4: Fly Parser Variable Interpolation**
  - **Validates: Requirements 2.7**

- [x] 2.5 Write property test for EggsBucket support
  - **Property 4a: Fly Parser EggsBucket Support**
  - **Validates: Requirements 1.6, 1.7, 2.8**

- [x] 3. Gosling CLI - Core Commands
  - Implement CLI framework using cobra
  - Create `init` command for Nest repository initialization
  - Create `add egg` and `add job` commands
  - Implement `validate` command for .fly files
  - _Requirements: 3.2, 3.3, 3.4, 3.5, 3.6_

- [x] 3.1 Write property test for Nest initialization structure
  - **Property 5: Nest Initialization Structure**
  - **Validates: Requirements 3.3**

- [x] 3.2 Write property test for configuration validation
  - **Property 6: Egg Configuration Validation**
  - **Validates: Requirements 3.6**

- [x] 3.3 Write property test for CLI mode equivalence
  - **Property 7: CLI Mode Equivalence**
  - **Validates: Requirements 3.7**

- [x] 4. Gosling CLI - Cloud SDK Integration
  - Integrate Yandex Cloud Go SDK
  - Integrate AWS SDK for Go v2
  - Integrate GitLab Go SDK
  - Implement .fly to SDK conversion logic
  - _Requirements: 3.9, 3.10, 9.1, 9.2, 9.3_

- [x] 4.1 Write property test for Fly to Cloud SDK conversion
  - **Property 8: Fly to Cloud SDK Conversion**
  - **Validates: Requirements 3.9, 9.3**

- [x] 5. Gosling CLI - Deployment Commands
  - Implement `deploy` command with dry-run support
  - Implement `rollback` command
  - Implement `status` command (calls MotherGoose API, NOT direct database access)
  - Implement MotherGoose API client for status queries
  - _Requirements: 10.1, 10.2, 10.6, 10.7, 10.8, 10.9_

- [x] 5.1 Implement MotherGoose API client in Gosling CLI
  - Create internal/mothergoose/client.go with MotherGooseClient interface
  - Implement HTTP client for MotherGoose API endpoints
  - Add methods: GetEggStatus, ListEggs, CreateOrUpdateEgg, GetDeploymentPlan, ListDeploymentPlans
  - Include proper error handling and retry logic
  - _Requirements: 10.6, 10.7_

- [x] 5.2 Refactor Gosling CLI commands to use MotherGoose API
  - Update status.go to use MotherGoose API client instead of PlanStore
  - Update deploy.go to call MotherGoose API for storing Egg configurations
  - Update rollback.go to call MotherGoose API for plan queries
  - Remove direct database access from all CLI commands
  - Remove PlanStore interface and implementations (dynamodb_store.go, ydb_store.go) from internal/storage
  - _Requirements: 10.6, 10.7_

- [x] 5.3 Write property test for dry-run non-modification
  - **Property 24: Dry-Run Non-Modification**
  - **Validates: Requirements 10.8**

- [x] 5.4 Write property test for deployment rollback
  - **Property 25: Deployment Rollback**
  - **Validates: Requirements 10.9**


- [x] 6. Checkpoint - Gosling CLI Core Functionality
  - All Gosling CLI tests pass
  - .fly parsing and validation works
  - Deployment and rollback commands implemented
  - MotherGoose API client ready for integration

- [x] 7. MotherGoose Backend - FastAPI Application Setup
  - Set up FastAPI application structure in mothergoose/src/app/main.py
  - Create API router structure in mothergoose/src/app/routers/
  - Implement health check endpoint (GET /health)
  - Configure CORS and middleware
  - Set up OpenAPI documentation
  - _Requirements: 4.7_

- [x] 8. MotherGoose Backend - API Endpoints for Gosling CLI
  - Implement GET /eggs/{name}/status endpoint (returns EggStatusResponse)
  - Implement GET /eggs/{name}/plans endpoint (lists deployment plans)
  - Implement GET /eggs/{name}/plans/{id} endpoint (gets specific plan)
  - Implement POST /eggs endpoint (creates or updates Egg configuration)
  - Implement GET /eggs endpoint (lists all Eggs)
  - Create Pydantic models for request/response schemas
  - _Requirements: 10.6, 10.7_

- [x] 9. MotherGoose Backend - Database Layer
  - Implement async database operations for runners table (YDB/DynamoDB)
  - Implement async database operations for egg_configs table
  - Implement async database operations for sync_history table
  - Implement async database operations for deployment_plans table
  - Implement async database operations for jobs table
  - Implement async database operations for audit_logs table
  - Implement async database operations for runner_metrics table
  - Implement async database operations for tofu_versions table
  - Create database schema initialization scripts
  - Implement connection pooling for YDB and DynamoDB
  - _Requirements: 4.6, 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7_

- [x] 9.1 Complete deployment_plans table implementation
  - Create DeploymentPlan Pydantic model in app/model/runners_models.py
  - Create DeploymentPlansTableYDB dataclass with proper schema
  - Add deployment_plans table to RunnerModelYDB.tables list
  - Implement actual database operations in DeploymentPlanService
  - Update deployment_plan_service.py to use real database queries
  - Add to_storage_dict() method for DeploymentPlan model
  - _Requirements: 14.3_

- [x] 9.2 Implement runner queries by egg_name
  - Add list_runners_by_egg(egg_name: str) method to RunnerService
  - Query runners table filtering by egg_name column
  - Return list of Runner objects for the specified egg
  - Update GET /eggs/{name}/status endpoint to populate active_runners field
  - Add unit tests for runner queries by egg_name
  - _Requirements: 4.6, 14.1_

- [x] 9.3 Configure database schema injection
  - Create get_ydb_schema() function in app/core/config.py
  - Read YDB configuration from environment variables
  - Initialize YDB connection on application startup
  - Register schema as FastAPI dependency in main.py
  - Update get_ydb_schema() in app/routers/eggs.py to use injected schema
  - Add configuration validation and error handling
  - _Requirements: 14.1_

- [x] 9.4 Add Git commit tracking to egg configurations
  - Update create_or_update_egg endpoint to accept git_commit parameter
  - Modify EggConfigRequest schema to include optional git_commit field
  - Update Git sync task to pass actual commit hash when upserting eggs
  - Change default from "unknown" to actual Git commit hash
  - Add validation to ensure git_commit is a valid SHA hash
  - _Requirements: 12.6, 14.3_


- [x] 9.5 Write property test for runner state persistence
  - **Property 11: Runner State Persistence**
  - **Validates: Requirements 4.6, 14.1**

- [x] 9.6 Write property test for database state recovery
  - **Property 31: Database State Recovery**
  - **Validates: Requirements 14.6**

- [x] 9.7 Write property test for database transaction atomicity
  - **Property 32: Database Transaction Atomicity**
  - **Validates: Requirements 14.7**

- [x] 10. MotherGoose Backend - Celery Task Queue Setup
  - Set up Celery application with YMQ/SQS backend
  - Remove Celery Beat (not compatible with serverless)
  - Implement task routing and priority queues
  - Set up task result backend (SQS/YMQ for production, Redis for development)
  - Configure task retry policies and error handling
  - _Requirements: 4.7_

- [x] 11. MotherGoose Backend - Cloud Trigger Setup
  - Configure Yandex Cloud Timer Triggers for periodic tasks (Git sync every 5 minutes, health checks every 10 minutes)
  - Configure AWS EventBridge Scheduler for periodic tasks (Git sync every 5 minutes, health checks every 10 minutes)
  - Create internal API endpoints for cloud triggers (/internal/sync-git, /internal/health-check)
  - Implement trigger authentication using secret tokens
  - Configure API Gateway to allow internal endpoints from cloud triggers only
  - _Requirements: 4.7, 13.7_

- [x] 12. MotherGoose Backend - Git Sync Implementation
  - Implement periodic Git sync task (triggered by cloud triggers every 5 minutes)
  - Implement event-driven Git sync on Nest repository push webhooks
  - Create SSH deploy key retrieval from secret storage
  - Implement Git clone/pull operations with deploy key authentication
  - Parse .fly files from Eggs/, Jobs/, UF/ directories
  - Update database cache with parsed configurations
  - Track Git commit hash for each synced configuration
  - Create sync history audit trail
  - _Requirements: 4.1, 4.2, 12.1, 12.2, 12.3, 12.6_

- [x] 13. MotherGoose Backend - Webhook Handling
  - Implement POST /webhooks/gitlab endpoint in FastAPI (create app/routers/webhooks.py)
  - Create webhook authentication using X-Gitlab-Token header (per-Egg shared secrets)
  - Implement webhook event parsing (push, merge_request, pipeline)
  - Create Celery task for async webhook processing (app/tasks/webhooks.py already exists)
  - Implement webhook event matching to Eggs
  - Distinguish between Nest repository webhooks (trigger Git sync) and Egg repository webhooks (trigger runner deployment via OpenTofu)
  - Note: Jobs folder creates GitLab scheduled pipelines + runner tokens + webhooks for Nest repo
  - When Nest pipeline fires → GitLab webhook (X-Gitlab-Token) → MotherGoose → Celery Task (SQS/YMQ) → OpenTofu → Deploy Runner
  - Include webhook router in main.py (currently missing)
  - _Requirements: 4.1, 4.2, 11.2, 16.1_

- [x] 13.1 Write property test for webhook event matching
  - **Property 9: Webhook Event Matching**
  - **Validates: Requirements 4.3**

- [x] 13.2 Write property test for GitLab webhook event support
  - **Property 26: GitLab Webhook Event Support**
  - **Validates: Requirements 11.2**

- [x] 13.3 Write property test for webhook authentication
  - **Property 33: Webhook Authentication**
  - **Validates: Requirements 16.1**


- [x] 14. MotherGoose Backend - Runner Orchestration
  - Implement runner type determination logic (serverless vs VM)
  - Create Celery tasks for runner deployment (deploy_runner, terminate_runner)
  - Implement runner state tracking in database
  - Create REST API endpoints for runner management (GET /runners, POST /runners, DELETE /runners/{id})
  - Implement runner provisioning workflow
  - _Requirements: 4.4, 4.5, 10.6, 10.7, 11.3_

- [x] 14.1 Write property test for runner type determination
  - **Property 10: Runner Type Determination**
  - **Validates: Requirements 4.4**

- [x] 15. MotherGoose Backend - Secret Management Integration
  - Implement SecretReference parser for URI schemes (yc-lockbox://, aws-sm://, vault://)
  - Implement YandexLockboxManager for Yandex Cloud Lockbox
  - Implement AWSSecretsManager for AWS Secrets Manager
  - Implement VaultManager for HashiCorp Vault (optional)
  - Implement SecretCache with TTL for caching retrieved secrets
  - Implement SecretMasker utility for masking secrets in logs
  - Integrate secret retrieval into Egg configuration processing
  - _Requirements: 16.7, 16.8, 16.9, 16.10, 16.11, 16.12, 17.1, 17.2, 17.3_

- [x] 15.1 Write property test for secret URI parsing
  - **Property 4b: Secret URI Parsing**
  - **Validates: Requirements 2.9, 16.8**

- [x] 15.2 Write property test for secret masking in logs
  - **Property 4c: Secret Masking in Logs**
  - **Validates: Requirements 16.9**

- [x] 15.3 Write property test for secret retrieval from Yandex Cloud Lockbox
  - **Property 36: Secret Retrieval from Yandex Cloud Lockbox**
  - **Validates: Requirements 16.7, 17.1**

- [x] 15.4 Write property test for secret retrieval from AWS Secrets Manager
  - **Property 37: Secret Retrieval from AWS Secrets Manager**
  - **Validates: Requirements 16.7, 17.2**

- [x] 15.5 Write property test for secret cache TTL
  - **Property 38: Secret Cache TTL**
  - **Validates: Requirements 16.11**

- [x] 15.6 Write property test for invalid secret reference error
  - **Property 39: Invalid Secret Reference Error**
  - **Validates: Requirements 16.12**

- [x] 15.7 Write property test for secret rotation propagation
  - **Property 40: Secret Rotation Propagation**
  - **Validates: Requirements 17.6**

- [x] 16. MotherGoose Backend - OpenTofu Integration for Runner Deployment
  - Verify existing OpenTofu binary management (already implemented in app/services/opentofu_binary.py)
  - Verify existing Jinja2 template rendering (already implemented in app/services/opentofu_configuration.py)
  - Implement S3 artifact caching logic for provider plugins and modules
  - Implement health checks template rendering
  - Generate cloud-init scripts for VMs using templates
  - Note: OpenTofu is used for ALL runner deployment (both Egg runners and Job runners)
  - _Requirements: 4.5, 5.3, 6.1_

- [x] 17. MotherGoose Backend - Serverless Runner Deployment
  - Verify existing OpenTofu template rendering for serverless runners
  - Implement serverless container deployment to Yandex Cloud Serverless Containers
  - Implement serverless container deployment to AWS Lambda (container image support)
  - Create container image build process with pre-installed binaries (Gosling CLI, GitLab Runner Agent)
  - Implement 60-minute timeout enforcement for serverless runners
  - Implement resource cleanup after completion or timeout
  - Note: Serverless runners use OpenTofu deployment (same as VM runners), not direct cloud SDK calls
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_


- [x] 17.1 Implement OpenTofu template rendering for runner deployment
  - Extend `OpenTofuConfiguration.__create_tofu_configuration_from_templates()` to render additional templates
  - Implement `tofu_providers_tf.j2` → `providers.tf` rendering
    - Render provider blocks with settings from Egg configuration
    - Support Yandex Cloud provider configuration
    - Support AWS provider configuration
  - Implement `tofu_resources_tf.j2` → `resources.tf` rendering
    - Render TLS private key resource for SSH access
    - Render local_file resource for SSH key storage
    - Render worker module block with dynamic source (git or registry)
    - Render rift module block (conditional, based on tofu_rift_required)
    - Support for_each iteration over worker/rift instances
    - Support cloud-init injection for VM chassis
  - Implement `tofu_variables_tf.j2` → `variables.tf` rendering
    - Render tofu_worker_instances variable definition
    - Render tofu_rift_instances variable definition (conditional)
  - Implement `tofu_data_tf.j2` → `data.tf` rendering (currently empty, prepare for future use)
  - Implement `tofu_rc.j2` → `.terraformrc` or `.tofurc` rendering
    - Configure provider installation methods
    - Configure plugin cache directory
  - Update `TofuSetting` dataclass to include new configuration options:
    - `worker_module_source` (git URL or registry URL with version)
    - `rift_module_source` (optional, for Rift server deployment)
    - `worker_instances` (map of instance configurations)
    - `rift_instances` (optional map of Rift instance configurations)
    - `vm_key_algorithm` and `vm_key_rsa_bits` for SSH key generation
  - _Requirements: 5.1, 5.3, 6.1_

- [x] 17.2 Write property test for serverless runner timeout

  - **Property 12: Serverless Runner Timeout Enforcement**
  - **Validates: Requirements 5.2**

- [x] 17.3 Write property test for serverless runner cleanup
  - **Property 13: Serverless Runner Cleanup**
  - **Validates: Requirements 5.6**

- [ ] 18. MotherGoose Backend - VM Runner Deployment
  - Verify existing OpenTofu template rendering for VM runners (already implemented in task 17.1)
  - Implement VM deployment using OpenTofu with Compute Module
  - Create Apex and Nadir pool management logic
  - Implement pool size limit enforcement (max_count, min_count)
  - Implement runner promotion logic (Nadir → Apex when demand increases)
  - Implement runner demotion logic (Apex → Nadir when idle beyond timeout)
  - Generate cloud-init scripts for VM setup using Jinja2 templates
  - Note: VM runners use OpenTofu deployment (same as serverless runners), not direct cloud SDK calls
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8_

- [ ] 18.1 Write property test for Apex pool size limits
  - **Property 14: Apex Pool Size Limits**
  - **Validates: Requirements 6.7**

- [ ] 18.2 Write property test for Nadir to Apex promotion
  - **Property 15: Nadir to Apex Promotion**
  - **Validates: Requirements 6.5**

- [ ] 18.3 Write property test for Apex to Nadir demotion
  - **Property 16: Apex to Nadir Demotion**
  - **Validates: Requirements 6.6**

- [ ] 19. Checkpoint - MotherGoose Core Functionality
  - Ensure all MotherGoose tests pass
  - Verify webhook processing works
  - Test runner deployment to both clouds
  - Run `make mg-tox-all` from dev-new-features root
  - Ask the user if questions arise

- [ ] 20. UglyFox Backend - Setup and Database Integration
  - Set up Celery worker structure in new UglyFox project
  - Implement async database operations
  - Configure cloud triggers for scheduled tasks (no Celery Beat in serverless)
  - _Requirements: 7.1_


- [ ] 21. UglyFox Backend - Policy Engine
  - Implement policy evaluation engine
  - Parse UF/config.fly for pruning policies
  - Create policy condition evaluator
  - _Requirements: 7.2, 7.4_

- [ ] 22. UglyFox Backend - Runner Lifecycle Management
  - Implement runner health monitoring
  - Create failure threshold termination logic
  - Implement age-based termination
  - Create Apex/Nadir transition logic
  - Implement audit logging
  - _Requirements: 7.1, 7.3, 7.4, 7.5, 7.6, 7.7_

- [ ] 22.1 Write property test for failure threshold termination
  - **Property 17: UglyFox Failure Threshold Termination**
  - **Validates: Requirements 7.3**

- [ ] 22.2 Write property test for age-based termination
  - **Property 18: UglyFox Age-Based Termination**
  - **Validates: Requirements 7.5**

- [ ] 22.3 Write property test for audit logging
  - **Property 19: UglyFox Audit Logging**
  - **Validates: Requirements 7.7**

- [ ] 23. Gosling CLI - Runner Mode Implementation
  - Implement `gosling runner` command
  - Create GitLab Runner Agent manager
  - Implement version synchronization with Egg config
  - Create metrics reporter for database
  - Implement signal handlers (SIGTERM, SIGHUP, SIGINT)
  - _Requirements: 6.8, 11.4, 11.5, 11.6_

- [ ] 23.1 Write property test for runner tag-based routing
  - **Property 27: Runner Tag-Based Routing**
  - **Validates: Requirements 11.7**

- [ ]* 23.2 Write property test for environment variable injection
  - **Property 29: Environment Variable Injection**
  - **Validates: Requirements 12.7**

- [ ] 24. Gosling CLI - Runner Mode Metrics
  - Implement periodic metrics collection (CPU, memory, disk)
  - Create metrics reporting to runner_metrics table
  - Implement heartbeat mechanism
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

- [ ] 25. Rift Server - Core Implementation
  - Implement Docker API proxy
  - Create artifact caching system
  - Implement LRU cache eviction
  - Add authentication for runner access
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

- [ ] 25.1 Write property test for Rift cache hit behavior
  - **Property 20: Rift Cache Hit Behavior**
  - **Validates: Requirements 8.4**

- [ ] 25.2 Write property test for Rift authentication
  - **Property 21: Rift Authentication Enforcement**
  - **Validates: Requirements 8.6**

- [ ] 25.3 Write property test for Rift optional dependency
  - **Property 22: Rift Optional Dependency**
  - **Validates: Requirements 8.7**

- [ ] 26. Configuration Management
  - Implement Egg configuration storage and retrieval
  - Create configuration update propagation
  - Implement environment variable injection
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 12.7_

- [ ] 26.1 Write property test for Egg config update propagation
  - **Property 28: Egg Config Update Propagation**
  - **Validates: Requirements 12.6**


- [ ] 27. Self-Management Jobs
  - Implement job scheduling using GitLab scheduled pipelines (not Celery Beat)
  - Create GitLab scheduled pipeline configuration for Nest repository
  - Create GitLab runner tokens for Job runners
  - Configure GitLab webhooks for Nest repository (X-Gitlab-Token authentication)
  - Create secret rotation job (rotates webhook secrets, runner tokens, deploy keys)
  - Create Nest repository update job (pulls latest changes, validates configurations)
  - Create runner image update job (updates container images for serverless runners)
  - Note: Jobs folder creates GitLab scheduled pipelines + runner tokens + webhooks for Nest repo
  - When Nest pipeline fires → GitLab webhook (X-Gitlab-Token) → MotherGoose → Celery Task (SQS/YMQ) → OpenTofu → Deploy Runner
  - Job runners use OpenTofu deployment (same as Egg runners), NOT a separate "self-runner" system
  - Job runner constraints: 10-minute time limit (vs 60 minutes for Egg serverless runners), cannot use Rift servers
  - Job runners are for lightweight self-management tasks only (secret rotation, config updates)
  - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [ ] 27.1 Write property test for cron job scheduling
  - **Property 30: Cron Job Scheduling**
  - **Validates: Requirements 13.7**

- [ ] 28. Multi-Cloud Consistency
  - Implement cloud-agnostic runner behavior
  - Test deployment to both Yandex Cloud and AWS
  - Verify equivalent behavior across clouds
  - _Requirements: 9.7, 9.8_

- [ ] 28.1 Write property test for multi-cloud deployment consistency
  - **Property 23: Multi-Cloud Deployment Consistency**
  - **Validates: Requirements 9.8**

- [ ] 29. Security Implementation
  - Implement data encryption at rest
  - Implement TLS for runner communication
  - Set up IAM roles for cloud authentication
  - Implement secret injection from secure storage
  - _Requirements: 16.2, 16.3, 16.4, 16.5, 16.7_

- [ ] 29.1 Write property test for data encryption at rest
  - **Property 34: Data Encryption at Rest**
  - **Validates: Requirements 16.4**

- [ ] 29.2 Write property test for communication encryption
  - **Property 35: Communication Encryption**
  - **Validates: Requirements 16.5**

- [ ] 30. Monitoring and Observability
  - Implement metrics emission for runner provisioning
  - Implement metrics emission for job execution
  - Implement metrics emission for pool sizes
  - Create health check endpoints
  - Integrate with Prometheus/Grafana
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7_

- [ ] 31. Integration Testing with Testcontainers
  - Set up YDB testcontainer fixtures (already implemented in conftest.py)
  - Set up LocalStack testcontainer for AWS services
  - Create end-to-end test scenarios
  - Test cross-component interactions
  - _Requirements: All_

- [ ] 32. Documentation and Deployment
  - Create deployment guides for Yandex Cloud and AWS
  - Document API Gateway configuration
  - Create runbooks for common operations
  - Document troubleshooting procedures

- [ ] 33. Final Checkpoint - System Integration
  - Run full test suite (unit + property tests)
  - Verify all components work together
  - Test failover scenarios
  - Perform load testing
  - Run `make mg-tox-all` from dev-new-features root
  - Ask the user if questions arise


## Additional Properties for Comprehensive Coverage

- [ ] 34. Write property test for EggsBucket repository isolation
  - **Property 41: EggsBucket Repository Isolation**
  - **Validates: Requirements 1.6, 1.7**

- [ ] 35. Write property test for EggsBucket shared configuration
  - **Property 42: EggsBucket Shared Configuration**
  - **Validates: Requirements 1.6**

## Notes

- Tasks marked with `*` are optional test tasks and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Integration tests use Testcontainers (YDB, DynamoDB via LocalStack, S3 via LocalStack)
- All Python tests use pytest with async support
- All Go tests use gopter for property-based testing
- **CRITICAL**: All implementations in "dev-new-features" worktree must pass `make mg-tox-all` before completion
- **CRITICAL**: Use `uv` for Python dependency management (`uv sync --all-groups`, `uv run pytest`)
- **CRITICAL**: Agent must wait for test completion (YDB static port mapping requires serialized execution)
- **CRITICAL**: Gosling CLI status command uses MotherGoose API client, NOT direct database access
- **CRITICAL**: OpenTofu is used for ALL runner deployment (Eggs + Jobs), NOT cloud SDKs
- **CRITICAL**: Jobs folder creates GitLab scheduled pipelines that trigger webhooks → MotherGoose → Celery → OpenTofu (same as Eggs)

## Development Workflow Reminders

1. **Worktree Isolation**: ALL new feature development happens in "dev-new-features" worktree ONLY
2. **Validation**: Run `make mg-tox-all` from "dev-new-features" root before considering any task complete
3. **Tox Configuration**: Defined in `pyproject.toml` under `[tool.tox]` section (not separate tox.ini)
4. **Python Versions**: Tests run on Python 3.10, 3.11, 3.12, and 3.13
5. **Code Quality**: All code must achieve pylint rating of 10/10
6. **Line Length**: Maximum 120 characters
7. **Task Comments**: Use format "# Task N: Brief description" (max 42 chars, no "TODO" keyword)
8. **Test Execution**: Agent waits for complete test suite execution (no premature termination)

## Architecture Reminders

### Deployment Mechanisms (CRITICAL)

1. **Bootstrap Phase (One-Time)**: Gosling CLI + Go SDKs → Deploy MotherGoose, UglyFox, databases, queues
2. **System Triggers (Internal)**: MotherGoose + Python SDKs → Create Timer Triggers (YC) / EventBridge (AWS)
3. **ALL Runners (Eggs + Jobs)**: MotherGoose + OpenTofu + Jinja2 → Deploy ALL runners

### Jobs Folder Flow

Jobs folder creates:
- GitLab scheduled pipelines for Nest repo
- GitLab runner tokens for Nest repo
- GitLab webhooks for Nest repo

When Nest pipeline fires:
1. GitLab webhook (X-Gitlab-Token) → MotherGoose
2. MotherGoose → Celery Task (SQS/YMQ)
3. Celery → OpenTofu (with Jinja2 templates)
4. OpenTofu → Deploy Runner

**Same flow as Eggs!** No separate "self-runner" system.

### Job Runner Constraints

- **Time Limit**: 10 minutes (vs 60 minutes for Egg serverless runners)
- **No Rift**: Cannot use Rift servers for caching
- **Purpose**: Lightweight self-management tasks only (secret rotation, config updates)

### GitOps Architecture

- **Source of Truth**: Nest Git repository (Eggs/, Jobs/, UF/)
- **Database**: Cache only (runtime state + parsed configs from Git)
- **Secrets**: NEVER in Git or database, only in Yandex Cloud Lockbox / AWS Secrets Manager
- **Git Sync**: Every 5 minutes via cloud triggers + immediate on push events
- **Deploy Keys**: Three SSH keys (mothergoose, uglyfox, selfjobs) stored in secret manager

