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
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7_

- [ ]* 2.1 Write property test for Fly parser round-trip
  - **Property 1: Fly Parser Round-Trip Consistency**
  - **Validates: Requirements 2.1, 2.4**

- [ ]* 2.2 Write property test for type error detection
  - **Property 2: Fly Parser Type Error Detection**
  - **Validates: Requirements 2.5**

- [ ]* 2.3 Write property test for nested block support
  - **Property 3: Fly Parser Nested Block Support**
  - **Validates: Requirements 2.6**

- [ ]* 2.4 Write property test for variable interpolation
  - **Property 4: Fly Parser Variable Interpolation**
  - **Validates: Requirements 2.7**

- [ ] 3. Gosling CLI - Core Commands
  - Implement CLI framework using cobra
  - Create `init` command for Nest repository initialization
  - Create `add egg` and `add job` commands
  - Implement `validate` command for .fly files
  - _Requirements: 3.2, 3.3, 3.4, 3.5, 3.6_

- [ ]* 2.1 Write property test for Nest initialization structure
  - **Property 5: Nest Initialization Structure**
  - **Validates: Requirements 3.3**

- [ ]* 2.2 Write property test for configuration validation
  - **Property 6: Egg Configuration Validation**
  - **Validates: Requirements 3.6**

- [ ]* 2.3 Write property test for CLI mode equivalence
  - **Property 7: CLI Mode Equivalence**
  - **Validates: Requirements 3.7**

- [ ] 3. Gosling CLI - Cloud SDK Integration
  - Integrate Yandex Cloud Go SDK
  - Integrate AWS SDK for Go v2
  - Integrate GitLab Go SDK
  - Implement .fly to SDK conversion logic
  - _Requirements: 3.9, 3.10, 9.1, 9.2, 9.3_

- [ ]* 3.1 Write property test for Fly to Cloud SDK conversion
  - **Property 8: Fly to Cloud SDK Conversion**
  - **Validates: Requirements 3.9, 9.3**

- [ ] 4. Gosling CLI - Deployment Commands
  - Implement `deploy` command with dry-run support
  - Implement `rollback` command
  - Implement `status` command
  - Create deployment plan storage in DynamoDB/YDB
  - _Requirements: 10.1, 10.2, 10.6, 10.7, 10.8, 10.9_

- [ ]* 4.1 Write property test for dry-run non-modification
  - **Property 24: Dry-Run Non-Modification**
  - **Validates: Requirements 10.8**

- [ ]* 4.2 Write property test for deployment rollback
  - **Property 25: Deployment Rollback**
  - **Validates: Requirements 10.9**

- [ ] 5. Checkpoint - Gosling CLI Core Functionality
  - Ensure all Gosling CLI tests pass
  - Verify .fly parsing and validation works
  - Test deployment and rollback commands
  - Ask the user if questions arise

- [ ] 6. MotherGoose Backend - FastAPI Setup
  - Set up FastAPI application structure
  - Configure API Gateway integration (OpenAPI spec)
  - Implement authentication middleware
  - Set up Celery task queue with YMQ/SQS
  - _Requirements: 4.7_

- [ ] 7. MotherGoose Backend - Database Layer
  - Implement async YDB operations with prepared queries
  - Implement async DynamoDB operations with aioboto3
  - Create database schema for runners, eggs, jobs, deployment plans
  - Implement connection pooling
  - _Requirements: 4.6, 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7_

- [ ]* 7.1 Write property test for runner state persistence
  - **Property 11: Runner State Persistence**
  - **Validates: Requirements 4.6, 14.1**

- [ ]* 7.2 Write property test for database state recovery
  - **Property 31: Database State Recovery**
  - **Validates: Requirements 14.6**

- [ ]* 7.3 Write property test for database transaction atomicity
  - **Property 32: Database Transaction Atomicity**
  - **Validates: Requirements 14.7**

- [ ] 8. MotherGoose Backend - Webhook Handling
  - Implement GitLab webhook endpoint
  - Create webhook authentication using shared secrets
  - Implement webhook event parsing
  - Create Celery task for webhook processing
  - _Requirements: 4.1, 4.2, 11.2, 16.1_

- [ ]* 8.1 Write property test for webhook event matching
  - **Property 9: Webhook Event Matching**
  - **Validates: Requirements 4.3**

- [ ]* 8.2 Write property test for GitLab webhook event support
  - **Property 26: GitLab Webhook Event Support**
  - **Validates: Requirements 11.2**

- [ ]* 8.3 Write property test for webhook authentication
  - **Property 33: Webhook Authentication**
  - **Validates: Requirements 16.1**

- [ ] 9. MotherGoose Backend - Runner Orchestration
  - Implement runner type determination logic
  - Create Celery tasks for runner deployment
  - Implement runner state tracking
  - Create REST API endpoints for runner management
  - _Requirements: 4.4, 4.5, 11.3_

- [ ]* 9.1 Write property test for runner type determination
  - **Property 10: Runner Type Determination**
  - **Validates: Requirements 4.4**

- [ ] 10. MotherGoose Backend - OpenTofu Integration
  - Integrate existing OpenTofu binary management
  - Implement Jinja2 template rendering for OpenTofu configs
  - Create S3 artifact caching logic
  - Implement health checks template
  - Generate cloud-init scripts for VMs
  - _Requirements: 4.5, 5.3, 6.1_

- [ ] 11. MotherGoose Backend - Serverless Runner Deployment
  - Implement serverless container deployment to Yandex Cloud Functions
  - Implement serverless container deployment to AWS Lambda
  - Create container image build process with pre-installed binaries
  - Implement 60-minute timeout enforcement
  - Implement resource cleanup after completion
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_

- [ ]* 11.1 Write property test for serverless runner timeout
  - **Property 12: Serverless Runner Timeout Enforcement**
  - **Validates: Requirements 5.2**

- [ ]* 11.2 Write property test for serverless runner cleanup
  - **Property 13: Serverless Runner Cleanup**
  - **Validates: Requirements 5.6**

- [ ] 12. MotherGoose Backend - VM Runner Deployment
  - Implement VM deployment using Compute Module
  - Create Apex and Nadir pool management
  - Implement pool size limit enforcement
  - Implement runner promotion/demotion logic
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7_

- [ ]* 12.1 Write property test for Apex pool size limits
  - **Property 14: Apex Pool Size Limits**
  - **Validates: Requirements 6.7**

- [ ]* 12.2 Write property test for Nadir to Apex promotion
  - **Property 15: Nadir to Apex Promotion**
  - **Validates: Requirements 6.5**

- [ ]* 12.3 Write property test for Apex to Nadir demotion
  - **Property 16: Apex to Nadir Demotion**
  - **Validates: Requirements 6.6**

- [ ] 13. Checkpoint - MotherGoose Core Functionality
  - Ensure all MotherGoose tests pass
  - Verify webhook processing works
  - Test runner deployment to both clouds
  - Ask the user if questions arise

- [ ] 14. UglyFox Backend - Setup and Database Integration
  - Set up Celery worker structure
  - Implement async database operations
  - Create Celery Beat for scheduled tasks
  - _Requirements: 7.1_

- [ ] 15. UglyFox Backend - Policy Engine
  - Implement policy evaluation engine
  - Parse UF/config.fly for pruning policies
  - Create policy condition evaluator
  - _Requirements: 7.2, 7.4_

- [ ] 16. UglyFox Backend - Runner Lifecycle Management
  - Implement runner health monitoring
  - Create failure threshold termination logic
  - Implement age-based termination
  - Create Apex/Nadir transition logic
  - Implement audit logging
  - _Requirements: 7.1, 7.3, 7.4, 7.5, 7.6, 7.7_

- [ ]* 16.1 Write property test for failure threshold termination
  - **Property 17: UglyFox Failure Threshold Termination**
  - **Validates: Requirements 7.3**

- [ ]* 16.2 Write property test for age-based termination
  - **Property 18: UglyFox Age-Based Termination**
  - **Validates: Requirements 7.5**

- [ ]* 16.3 Write property test for audit logging
  - **Property 19: UglyFox Audit Logging**
  - **Validates: Requirements 7.7**

- [ ] 17. Gosling CLI - Runner Mode Implementation
  - Implement `gosling runner` command
  - Create GitLab Runner Agent manager
  - Implement version synchronization with Egg config
  - Create metrics reporter for database
  - Implement signal handlers (SIGTERM, SIGHUP, SIGINT)
  - _Requirements: 6.8, 11.4, 11.5, 11.6_

- [ ]* 17.1 Write property test for runner tag-based routing
  - **Property 27: Runner Tag-Based Routing**
  - **Validates: Requirements 11.7**

- [ ]* 17.2 Write property test for environment variable injection
  - **Property 29: Environment Variable Injection**
  - **Validates: Requirements 12.7**

- [ ] 18. Gosling CLI - Runner Mode Metrics
  - Implement periodic metrics collection (CPU, memory, disk)
  - Create metrics reporting to runner_metrics table
  - Implement heartbeat mechanism
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

- [ ] 19. Rift Server - Core Implementation
  - Implement Docker API proxy
  - Create artifact caching system
  - Implement LRU cache eviction
  - Add authentication for runner access
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

- [ ]* 19.1 Write property test for Rift cache hit behavior
  - **Property 20: Rift Cache Hit Behavior**
  - **Validates: Requirements 8.4**

- [ ]* 19.2 Write property test for Rift authentication
  - **Property 21: Rift Authentication Enforcement**
  - **Validates: Requirements 8.6**

- [ ]* 19.3 Write property test for Rift optional dependency
  - **Property 22: Rift Optional Dependency**
  - **Validates: Requirements 8.7**

- [ ] 20. Configuration Management
  - Implement Egg configuration storage and retrieval
  - Create configuration update propagation
  - Implement environment variable injection
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6, 12.7_

- [ ]* 20.1 Write property test for Egg config update propagation
  - **Property 28: Egg Config Update Propagation**
  - **Validates: Requirements 12.6**

- [ ] 21. Self-Management Jobs
  - Implement job scheduling with cron expressions
  - Create secret rotation job
  - Create Nest repository update job
  - Create runner image update job
  - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [ ]* 21.1 Write property test for cron job scheduling
  - **Property 30: Cron Job Scheduling**
  - **Validates: Requirements 13.7**

- [ ] 22. Multi-Cloud Consistency
  - Implement cloud-agnostic runner behavior
  - Test deployment to both Yandex Cloud and AWS
  - Verify equivalent behavior across clouds
  - _Requirements: 9.7, 9.8_

- [ ]* 22.1 Write property test for multi-cloud deployment consistency
  - **Property 23: Multi-Cloud Deployment Consistency**
  - **Validates: Requirements 9.8**

- [ ] 23. Security Implementation
  - Implement data encryption at rest
  - Implement TLS for runner communication
  - Set up IAM roles for cloud authentication
  - Implement secret injection from secure storage
  - _Requirements: 16.2, 16.3, 16.4, 16.5, 16.7_

- [ ]* 23.1 Write property test for data encryption at rest
  - **Property 34: Data Encryption at Rest**
  - **Validates: Requirements 16.4**

- [ ]* 23.2 Write property test for communication encryption
  - **Property 35: Communication Encryption**
  - **Validates: Requirements 16.5**

- [ ] 24. Monitoring and Observability
  - Implement metrics emission for runner provisioning
  - Implement metrics emission for job execution
  - Implement metrics emission for pool sizes
  - Create health check endpoints
  - Integrate with Prometheus/Grafana
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7_

- [ ] 25. Integration Testing with Testcontainers
  - Set up YDB testcontainer fixtures
  - Set up LocalStack testcontainer for AWS services
  - Create end-to-end test scenarios
  - Test cross-component interactions
  - _Requirements: All_

- [ ] 26. Documentation and Deployment
  - Create deployment guides for Yandex Cloud and AWS
  - Document API Gateway configuration
  - Create runbooks for common operations
  - Document troubleshooting procedures

- [ ] 27. Final Checkpoint - System Integration
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
