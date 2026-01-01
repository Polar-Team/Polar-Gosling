# Implementation Plan: GitOps Runner Orchestration

## Overview

This implementation plan breaks down the GitOps Runner Orchestration system into discrete, manageable tasks. The system will be built incrementally, with each task building on previous work.

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
  - **Validates: Requirements 3.9, 9.3_

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

- [x] 9.1 Write property test for runner state persistence

  - **Property 11: Runner State Persistence**
  - **Validates: Requirements 4.6, 14.1**

- [x] 9.2 Write property test for database state recovery

  - **Property 31: Database State Recovery**
  - **Validates: Requirements 14.6**

- [x] 9.3 Write property test for database transaction atomicity

  - **Property 32: Database Transaction Atomicity**
  - **Validates: Requirements 14.7**

- [ ] 10. MotherGoose Backend - Celery Task Queue Setup
  - Set up Celery application with YMQ/SQS backend
  - Configure Celery Beat for scheduled tasks
  - Implement task routing and priority queues
  - Set up task result backend (Redis or database)
  - Configure task retry policies and error handling
  - _Requirements: 4.7_

- [ ] 11. MotherGoose Backend - Git Sync Implementation
  - Implement periodic Git sync task (every 5 minutes via Celery Beat)
  - Implement event-driven Git sync on Nest repository push webhooks
  - Create SSH deploy key retrieval from secret storage
  - Implement Git clone/pull operations with deploy key authentication
  - Parse .fly files from Eggs/, Jobs/, UF/ directories
  - Update database cache with parsed configurations
  - Track Git commit hash for each synced configuration
  - Create sync history audit trail
  - _Requirements: 4.1, 4.2, 12.1, 12.2, 12.3, 12.6_

- [ ] 12. MotherGoose Backend - Webhook Handling
  - Implement POST /webhooks/gitlab endpoint in FastAPI
  - Create webhook authentication using per-Egg shared secrets
  - Implement webhook event parsing (push, merge_request, pipeline)
  - Create Celery task for async webhook processing
  - Implement webhook event matching to Eggs
  - Distinguish between Nest repository webhooks (trigger Git sync) and Egg repository webhooks (trigger runner deployment)
  - _Requirements: 4.1, 4.2, 11.2, 16.1_

- [ ]* 12.1 Write property test for webhook event matching
  - **Property 9: Webhook Event Matching**
  - **Validates: Requirements 4.3**

- [ ]* 12.2 Write property test for GitLab webhook event support
  - **Property 26: GitLab Webhook Event Support**
  - **Validates: Requirements 11.2**

- [ ]* 12.3 Write property test for webhook authentication
  - **Property 33: Webhook Authentication**
  - **Validates: Requirements 16.1**

- [ ] 13. MotherGoose Backend - Runner Orchestration
  - Implement runner type determination logic (serverless vs VM)
  - Create Celery tasks for runner deployment (deploy_runner, terminate_runner)
  - Implement runner state tracking in database
  - Create REST API endpoints for runner management (GET /runners, POST /runners, DELETE /runners/{id})
  - Implement runner provisioning workflow
  - _Requirements: 4.4, 4.5, 10.6, 10.7, 11.3_

- [ ]* 13.1 Write property test for runner type determination
  - **Property 10: Runner Type Determination**
  - **Validates: Requirements 4.4**

- [ ] 14. MotherGoose Backend - Secret Management Integration
  - Implement SecretReference parser for URI schemes (yc-lockbox://, aws-sm://, vault://)
  - Implement YandexLockboxManager for Yandex Cloud Lockbox
  - Implement AWSSecretsManager for AWS Secrets Manager
  - Implement VaultManager for HashiCorp Vault (optional)
  - Implement SecretCache with TTL for caching retrieved secrets
  - Implement SecretMasker utility for masking secrets in logs
  - Integrate secret retrieval into Egg configuration processing
  - _Requirements: 16.7, 16.8, 16.9, 16.10, 16.11, 16.12, 17.1, 17.2, 17.3_

- [ ]* 14.1 Write property test for secret URI parsing
  - **Property 4b: Secret URI Parsing**
  - **Validates: Requirements 2.9, 16.8**

- [ ]* 14.2 Write property test for secret masking in logs
  - **Property 4c: Secret Masking in Logs**
  - **Validates: Requirements 16.9**

- [ ]* 14.3 Write property test for secret retrieval from Yandex Cloud Lockbox
  - **Property 36: Secret Retrieval from Yandex Cloud Lockbox**
  - **Validates: Requirements 16.7, 17.1**

- [ ]* 14.4 Write property test for secret retrieval from AWS Secrets Manager
  - **Property 37: Secret Retrieval from AWS Secrets Manager**
  - **Validates: Requirements 16.7, 17.2**

- [ ]* 14.5 Write property test for secret cache TTL
  - **Property 38: Secret Cache TTL**
  - **Validates: Requirements 16.11**

- [ ]* 14.6 Write property test for invalid secret reference error
  - **Property 39: Invalid Secret Reference Error**
  - **Validates: Requirements 16.12**

- [ ]* 14.7 Write property test for secret rotation propagation
  - **Property 40: Secret Rotation Propagation**
  - **Validates: Requirements 17.6**

- [ ] 15. MotherGoose Backend - OpenTofu Integration
  - Verify existing OpenTofu binary management (already implemented in app/services/opentofu_binary.py)
  - Verify existing Jinja2 template rendering (already implemented in app/services/opentofu_configuration.py)
  - Implement S3 artifact caching logic for provider plugins and modules
  - Implement health checks template rendering
  - Generate cloud-init scripts for VMs using templates
  - _Requirements: 4.5, 5.3, 6.1_

- [ ] 16. MotherGoose Backend - Serverless Runner Deployment
  - Implement serverless container deployment to Yandex Cloud Functions
  - Implement serverless container deployment to AWS Lambda
  - Create container image build process with pre-installed binaries
  - Implement 60-minute timeout enforcement
  - Implement resource cleanup after completion
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_

- [ ]* 16.1 Write property test for serverless runner timeout
  - **Property 12: Serverless Runner Timeout Enforcement**
  - **Validates: Requirements 5.2**

- [ ]* 16.2 Write property test for serverless runner cleanup
  - **Property 13: Serverless Runner Cleanup**
  - **Validates: Requirements 5.6**

- [ ] 17. MotherGoose Backend - VM Runner Deployment
  - Implement VM deployment using Compute Module
  - Create Apex and Nadir pool management
  - Implement pool size limit enforcement
  - Implement runner promotion/demotion logic
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7_

- [ ]* 17.1 Write property test for Apex pool size limits
  - **Property 14: Apex pool Size Limits**
  - **Validates: Requirements 6.7**

- [ ]* 17.2 Write property test for Nadir to Apex promotion
  - **Property 15: Nadir to Apex Promotion**
  - **Validates: Requirements 6.5**

- [ ]* 17.3 Write property test for Apex to Nadir demotion
  - **Property 16: Apex to Nadir Demotion**
  - **Validates: Requirements 6.6**

- [ ] 18. Checkpoint - MotherGoose Core Functionality
  - Ensure all MotherGoose tests pass
  - Verify webhook processing works
  - Test runner deployment to both clouds
  - Ask the user if questions arise

- [ ] 19. UglyFox Backend - Setup and Database Integration
  - Set up Celery worker structure in new UglyFox project
  - Implement async database operations
  - Create Celery Beat for scheduled tasks
  - _Requirements: 7.1_

- [ ] 20. UglyFox Backend - Policy Engine
  - Implement policy evaluation engine
  - Parse UF/config.fly for pruning policies
  - Create policy condition evaluator
  - _Requirements: 7.2, 7.4_

- [ ] 21. UglyFox Backend - Runner Lifecycle Management
  - Implement runner health monitoring
  - Create failure threshold termination logic
  - Implement age-based termination
  - Create Apex/Nadir transition logic
  - Implement audit logging
  - _Requirements: 7.1, 7.3, 7.4, 7.5, 7.6, 7.7_

- [ ]* 21.1 Write property test for failure threshold termination
  - **Property 17: UglyFox Failure Threshold Termination**
  - **Validates: Requirements 7.3**

- [ ]* 21.2 Write property test for age-based termination
  - **Property 18: UglyFox Age-Based Termination**
  - **Validates: Requirements 7.5**

- [ ]* 21.3 Write property test for audit logging
  - **Property 19: UglyFox Audit Logging**
  - **Validates: Requirements 7.7**

- [ ] 22. Gosling CLI - Runner Mode Implementation
  - Implement `gosling runner` command
  - Create GitLab Runner Agent manager
  - Implement version synchronization with Egg config
  - Create metrics reporter for database
  - Implement signal handlers (SIGTERM, SIGHUP, SIGINT)
  - _Requirements: 6.8, 11.4, 11.5, 11.6_

- [ ]* 22.1 Write property test for runner tag-based routing
  - **Property 27: Runner Tag-Based Routing**
  - **Validates: Requirements 11.7**

- [ ]* 22.2 Write property test for environment variable injection
  - **Property 29: Environment Variable Injection**
  - **Validates: Requirements 12.7**

- [ ] 23. Gosling CLI - Runner Mode Metrics
  - Implement periodic metrics collection (CPU, memory, disk)
  - Create metrics reporting to runner_metrics table
  - Implement heartbeat mechanism
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

- [ ] 24. Rift Server - Core Implementation
  - Implement Docker API proxy
  - Create artifact caching system
  - Implement LRU cache eviction
  - Add authentication for runner access
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

- [ ]* 24.1 Write property test for Rift cache hit behavior
  - **Property 20: Rift Cache Hit Behavior**
  - **Validates: Requirements 8.4**

- [ ]* 24.2 Write property test for Rift authentication
  - **Property 21: Rift Authentication Enforcement**
  - **Validates: Requirements 8.6**

- [ ]* 24.3 Write property test for Rift optional dependency
  - **Property 22: Rift Optional Dependency**
  - **Validates: Requirements 8.7**

- [ ] 25. Configuration Management
  - Implement Egg configuration storage and retrieval
  - Create configuration update propagation
  - Implement environment variable injection
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 12.7_

- [ ]* 25.1 Write property test for Egg config update propagation
  - **Property 28: Egg Config Update Propagation**
  - **Validates: Requirements 12.6**

- [ ] 26. Self-Management Jobs
  - Implement job scheduling with cron expressions
  - Create secret rotation job
  - Create Nest repository update job
  - Create runner image update job
  - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [ ]* 26.1 Write property test for cron job scheduling
  - **Property 30: Cron Job Scheduling**
  - **Validates: Requirements 13.7**

- [ ] 27. Multi-Cloud Consistency
  - Implement cloud-agnostic runner behavior
  - Test deployment to both Yandex Cloud and AWS
  - Verify equivalent behavior across clouds
  - _Requirements: 9.7, 9.8_

- [ ]* 27.1 Write property test for multi-cloud deployment consistency
  - **Property 23: Multi-Cloud Deployment Consistency**
  - **Validates: Requirements 9.8**

- [ ] 28. Security Implementation
  - Implement data encryption at rest
  - Implement TLS for runner communication
  - Set up IAM roles for cloud authentication
  - Implement secret injection from secure storage
  - _Requirements: 16.2, 16.3, 16.4, 16.5, 16.7_

- [ ]* 28.1 Write property test for data encryption at rest
  - **Property 34: Data Encryption at Rest**
  - **Validates: Requirements 16.4**

- [ ]* 28.2 Write property test for communication encryption
  - **Property 35: Communication Encryption**
  - **Validates: Requirements 16.5**

- [ ] 29. Monitoring and Observability
  - Implement metrics emission for runner provisioning
  - Implement metrics emission for job execution
  - Implement metrics emission for pool sizes
  - Create health check endpoints
  - Integrate with Prometheus/Grafana
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7_

- [ ] 30. Integration Testing with Testcontainers
  - Set up YDB testcontainer fixtures (already implemented in conftest.py)
  - Set up LocalStack testcontainer for AWS services
  - Create end-to-end test scenarios
  - Test cross-component interactions
  - _Requirements: All_

- [ ] 31. Documentation and Deployment
  - Create deployment guides for Yandex Cloud and AWS
  - Document API Gateway configuration
  - Create runbooks for common operations
  - Document troubleshooting procedures

- [ ] 32. Final Checkpoint - System Integration
  - Run full test suite (unit + property tests)
  - Verify all components work together
  - Test failover scenarios
  - Perform load testing
  - Ask the user if questions arise

## Notes

- Tasks marked with `*` are optional test tasks and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Integration tests use Testcontainers (YDB, DynamoDB via LocalStack, S3 via LocalStack)
- All Python tests use pytest with async support
- All Go tests use gopter for property-based testing
