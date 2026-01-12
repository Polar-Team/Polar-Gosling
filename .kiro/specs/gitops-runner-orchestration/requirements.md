# Requirements Document

## Introduction

The Polar Gosling GitOps Runner Orchestration system is a comprehensive platform for managing CI/CD runners across multiple cloud providers (Yandex Cloud and AWS) using a GitOps approach. The system consists of multiple backend servers (MotherGoose, UglyFox), a custom configuration language (.fly), a CLI tool for bootstrapping and management, and support for both serverless containers and autoscaling VMs as runner execution environments. The main repository (Nest) manages multiple downstream repositories (Eggs) through declarative configuration files.

## Glossary

- **Nest**: The main GitOps repository that configures and manages runners for downstream repositories
- **Egg**: A single managed repository or GitLab group configured by the Nest
- **EggsBucket**: A group of multiple managed repositories configured together with shared runner configuration
- **MotherGoose**: The primary backend server that handles job orchestration, webhook processing, and runner deployment
- **UglyFox**: The secondary backend server responsible for pruning failed runners and managing runner lifecycle states
- **Apex**: Active running runner VM/containers (with quantity and resource limits)
- **Nadir**: Dormant or sleeping runner VM/containers (with quantity and resource limits)
- **Rift**: A remote context server for Docker/Podman/nerdctl execution and artifact caching
- **Fly_Language**: The custom domain-specific language (.fly extension) for declarative configuration, similar to HCL but more structured and typed
- **Gosling_CLI**: The command-line tool (written in Go) for bootstrapping Nest repositories and converting .fly configurations to cloud/GitLab objects using Go SDKs
- **Runner**: A CI/CD execution environment (VM or serverless container) that executes GitLab jobs
- **Serverless_Runner**: A containerized runner with 60-minute execution limit
- **VM_Runner**: A virtual machine-based runner with persistent agent for long-running jobs
- **Secret_URI**: A URI scheme for referencing secrets stored in cloud secret management services (yc-lockbox://, aws-sm://, vault://)
- **Yandex_Cloud_Lockbox**: Yandex Cloud's secret management service for storing and retrieving sensitive data
- **AWS_Secrets_Manager**: AWS's secret management service for storing and retrieving sensitive data
- **GitLab_Server**: The FQDN of a GitLab instance (e.g., gitlab.com, gitlab.company.com)
- **Project_Level_Runner**: A runner registered to a specific GitLab project
- **Group_Level_Runner**: A runner registered to a GitLab group, available to all projects in that group

## Requirements

### Requirement 1: Nest Repository Structure

**User Story:** As a DevOps engineer, I want a standardized Nest repository structure, so that I can organize managed repositories, jobs, and policies in a predictable way.

#### Acceptance Criteria

1. THE Nest_Repository SHALL contain an "Eggs" directory for managed repository configurations
2. THE Nest_Repository SHALL contain a "Jobs" directory for self-management task definitions
3. THE Nest_Repository SHALL contain a "UF" directory for UglyFox configuration
4. THE Nest_Repository SHALL contain a "MG" directory for MotherGoose infrastructure configuration
5. WHEN an Egg is a single project, THE Eggs_Directory SHALL contain a project folder with a config.fly file
6. WHEN multiple projects are grouped, THE Eggs_Directory SHALL contain an EggsBucket folder with a config.fly file
7. THE EggsBucket_Config SHALL define shared runner configuration for multiple repositories
8. THE EggsBucket_Config SHALL contain a "repositories" block listing all managed repositories
9. THE Jobs_Directory SHALL contain files named with pattern "{job_name}.fly"
10. THE UF_Directory SHALL contain a file named "config.fly" with runner pruning policies
11. THE MG_Directory SHALL contain a file named "config.fly" with MotherGoose infrastructure configuration
12. THE MG_Config SHALL define API Gateway configuration for MotherGoose and UglyFox
13. THE MG_Config SHALL define serverless container configuration for MotherGoose FastAPI application
14. THE MG_Config SHALL define serverless container configuration for Celery workers
15. THE MG_Config SHALL define serverless container configuration for UglyFox workers
16. THE MG_Config SHALL define message queue configuration (YMQ for Yandex Cloud, SQS for AWS)
17. THE MG_Config SHALL define cloud trigger configuration (Timer Triggers for Yandex Cloud, EventBridge for AWS)
18. THE MG_Config SHALL define authentication function configuration for API Gateway endpoints

### Requirement 2: Fly Language Parser and Interpreter

**User Story:** As a platform developer, I want a custom configuration language parser, so that users can define infrastructure and policies in a type-safe, structured format.

#### Acceptance Criteria

1. THE Fly_Parser SHALL parse .fly files with HCL-like syntax
2. THE Fly_Language SHALL enforce stronger typing than HCL
3. THE Fly_Language SHALL provide structured validation of configuration blocks
4. WHEN parsing a .fly file, THE Fly_Parser SHALL validate syntax against the grammar specification
5. WHEN a .fly file contains type errors, THE Fly_Parser SHALL return descriptive error messages
6. THE Fly_Parser SHALL support nested block structures for complex configurations
7. THE Fly_Parser SHALL support variable interpolation and references
8. THE Fly_Parser SHALL support both "egg" and "eggsbucket" block types
9. THE Fly_Parser SHALL parse secret references using URI schemes (yc-lockbox://, aws-sm://, vault://)

### Requirement 3: Gosling CLI Tool for Nest Bootstrapping and Deployment

**User Story:** As a DevOps engineer, I want a Go-based CLI tool to bootstrap and manage Nest repositories, so that I can quickly set up new GitOps environments and deploy resources using native cloud SDKs.

#### Acceptance Criteria

1. THE Gosling_CLI SHALL be written in Go
2. THE Gosling_CLI SHALL provide a command to initialize a new Nest repository structure
3. WHEN initializing a Nest, THE Gosling_CLI SHALL create the Eggs, Jobs, and UF directories
4. THE Gosling_CLI SHALL provide commands to add new Egg configurations
5. THE Gosling_CLI SHALL provide commands to add new Job definitions
6. THE Gosling_CLI SHALL validate .fly configuration files before committing
7. THE Gosling_CLI SHALL support both interactive and non-interactive modes
8. THE Gosling_CLI SHALL be usable within runner deployment processes
9. THE Gosling_CLI SHALL convert .fly configurations to cloud provider objects using Go SDKs
10. THE Gosling_CLI SHALL convert .fly configurations to GitLab objects using GitLab Go SDK
11. WHEN deploying resources from Nest, THE Gosling_CLI SHALL use native Go SDKs for Yandex Cloud and AWS

### Requirement 4: MotherGoose Backend Server

**User Story:** As a system administrator, I want a central orchestration server, so that I can manage job distribution and runner deployment across multiple clouds.

**IMPORTANT NOTE**: MotherGoose deploys runners using OpenTofu with Jinja2 templates, NOT cloud SDKs directly. Cloud SDKs are only used for bootstrap infrastructure (one-time setup) and cloud-native services (Timer Triggers, EventBridge, etc.).

#### Acceptance Criteria

1. THE MotherGoose_Server SHALL receive webhooks from GitLab repositories
2. WHEN a webhook is received, THE MotherGoose_Server SHALL parse the event payload
3. THE MotherGoose_Server SHALL match webhook events to configured Eggs
4. THE MotherGoose_Server SHALL determine the appropriate runner type (serverless or VM) based on job requirements
5. THE MotherGoose_Server SHALL deploy runners using OpenTofu configurations
6. THE MotherGoose_Server SHALL track runner states in a database (YDB or DynamoDB)
7. THE MotherGoose_Server SHALL provide REST API endpoints for runner management
8. THE MotherGoose_Server SHALL support deployment to both Yandex Cloud and AWS

### Requirement 5: Serverless Container Runner Deployment

**User Story:** As a cost-conscious operator, I want serverless container runners for short jobs, so that I can minimize infrastructure costs for ephemeral workloads.

#### Acceptance Criteria

1. THE MotherGoose_Server SHALL deploy serverless containers to Yandex Cloud Functions or AWS Lambda
2. THE Serverless_Runner SHALL have a maximum execution time of 60 minutes
3. WHEN deploying a serverless runner, THE MotherGoose_Server SHALL inject the Gosling CLI tool
4. THE Serverless_Runner SHALL execute the Gosling CLI tool to set up the GitLab runner agent
5. THE Serverless_Runner SHALL register with GitLab and execute the assigned job
6. WHEN a serverless runner completes or times out, THE MotherGoose_Server SHALL clean up resources
7. THE Serverless_Runner SHALL report execution status back to MotherGoose

### Requirement 6: VM-Based Runner Deployment with Autoscaling

**User Story:** As a platform operator, I want autoscaling VM-based runners, so that I can handle long-running jobs and maintain a pool of ready runners.

#### Acceptance Criteria

1. THE MotherGoose_Server SHALL deploy VM runners using the Compute Module
2. THE VM_Runner SHALL run a persistent GitLab runner agent
3. THE MotherGoose_Server SHALL maintain a pool of Apex (active) runners
4. THE MotherGoose_Server SHALL maintain a pool of Nadir (dormant) runners
5. WHEN job demand increases, THE MotherGoose_Server SHALL promote Nadir runners to Apex state
6. WHEN job demand decreases, THE MotherGoose_Server SHALL demote Apex runners to Nadir state
7. THE MotherGoose_Server SHALL enforce quantity and resource limits for Apex and Nadir pools
8. THE VM_Runner SHALL update itself and handle runner workflow orchestration

### Requirement 7: UglyFox Runner Lifecycle Management

**User Story:** As a reliability engineer, I want automated cleanup of failed runners, so that I can maintain system health without manual intervention.

#### Acceptance Criteria

1. THE UglyFox_Server SHALL monitor runner health status
2. WHEN a runner fails, THE UglyFox_Server SHALL evaluate pruning policies from UF/config.fly
3. THE UglyFox_Server SHALL terminate runners that exceed failure thresholds
4. THE UglyFox_Server SHALL transition runners between Apex and Nadir states based on policies
5. THE UglyFox_Server SHALL enforce maximum runner age policies
6. THE UglyFox_Server SHALL report pruning actions to MotherGoose
7. THE UglyFox_Server SHALL maintain audit logs of all lifecycle actions

### Requirement 8: Rift Remote Context Server

**User Story:** As a developer, I want a remote Docker context server, so that I can cache artifacts and share build contexts across runners.

#### Acceptance Criteria

1. THE Rift_Server SHALL provide remote context for Docker, Podman, and nerdctl
2. THE Rift_Server SHALL cache build artifacts to reduce redundant downloads
3. THE Rift_Server SHALL be accessible from both VM and serverless runners
4. WHEN a runner requests an artifact, THE Rift_Server SHALL serve from cache if available
5. THE Rift_Server SHALL implement cache eviction policies based on age and size
6. THE Rift_Server SHALL support authentication and authorization for runner access
7. THE Rift_Server SHALL be optional (runners can operate without Rift)

### Requirement 9: Multi-Cloud Infrastructure Provisioning via Go SDKs

**User Story:** As a cloud architect, I want infrastructure provisioned across multiple clouds using native Go SDKs, so that I can leverage the best features of each provider with type-safe, idiomatic code.

**IMPORTANT NOTE**: Go SDKs are used ONLY for bootstrap infrastructure deployment (one-time setup of MotherGoose, UglyFox, databases, queues). Runtime runner deployment uses OpenTofu, not Go SDKs.

#### Acceptance Criteria

1. THE System SHALL support deployment to Yandex Cloud using Yandex Cloud Go SDK
2. THE System SHALL support deployment to AWS using AWS SDK for Go
3. THE Gosling_CLI SHALL convert .fly configurations to native SDK calls
4. THE Compute_Module SHALL provision VMs on both Yandex Cloud and AWS
5. THE Compute_Module SHALL provision serverless containers on both platforms
6. WHEN provisioning infrastructure, THE Gosling_CLI SHALL use Go SDK methods for resource creation
7. THE System SHALL support cloud-specific features (Yandex Cloud Functions, AWS Lambda) through their respective SDKs
8. THE System SHALL maintain consistent runner behavior across cloud providers
9. THE Gosling_CLI SHALL handle SDK authentication using cloud provider credentials

### Requirement 10: Nest Repository Deployment via Gosling CLI

**User Story:** As a platform operator, I want all Nest repository resources deployed using the Gosling CLI, so that I have a consistent deployment mechanism across all environments.

#### Acceptance Criteria

1. THE Gosling_CLI SHALL read .fly configuration files from the Nest repository
2. THE Gosling_CLI SHALL parse Egg configurations and convert them to cloud resources
3. THE Gosling_CLI SHALL use Yandex Cloud Go SDK for Yandex Cloud deployments
4. THE Gosling_CLI SHALL use AWS SDK for Go for AWS deployments
5. THE Gosling_CLI SHALL use GitLab Go SDK for GitLab runner registration
6. WHEN deploying from Nest, THE Gosling_CLI SHALL create all required infrastructure resources
7. WHEN deploying from Nest, THE Gosling_CLI SHALL configure GitLab webhooks using GitLab SDK
8. THE Gosling_CLI SHALL support dry-run mode to preview changes before applying
9. THE Gosling_CLI SHALL provide rollback capabilities for failed deployments

### Requirement 11: GitLab Integration

**User Story:** As a GitLab user, I want seamless integration with GitLab CI/CD, so that my pipelines can use dynamically provisioned runners.

#### Acceptance Criteria

1. THE System SHALL register runners with GitLab using the GitLab Go SDK
2. THE System SHALL support GitLab webhook events (push, merge request, pipeline)
3. WHEN a GitLab job is queued, THE System SHALL provision an appropriate runner
4. THE Runner SHALL authenticate with GitLab using runner tokens
5. THE Runner SHALL execute GitLab CI/CD jobs according to .gitlab-ci.yml
6. THE Runner SHALL report job status and logs back to GitLab
7. THE System SHALL support GitLab runner tags for job routing
8. THE System SHALL support multiple GitLab instances (GitLab.com, self-hosted, GitLab Dedicated)
9. THE Egg_Config SHALL specify GitLab server FQDN for API communication
10. THE System SHALL support both project-level and group-level runner registration
11. WHEN an Egg specifies a group_id, THE System SHALL discover all projects in that group
12. WHEN an Egg specifies a group_id, THE System SHALL configure webhooks for all projects in the group
13. THE System SHALL register group-level runners that are available to all projects in the group
14. THE Gosling_CLI SHALL require API token with appropriate permissions for group-level operations

### Requirement 12: Configuration Management for Eggs

**User Story:** As a repository maintainer, I want to configure runner requirements per repository, so that each project gets appropriate compute resources.

#### Acceptance Criteria

1. THE Egg_Config SHALL specify runner type (serverless or VM)
2. THE Egg_Config SHALL specify resource requirements (CPU, memory, disk)
3. THE Egg_Config SHALL specify maximum concurrent runners
4. THE Egg_Config SHALL specify runner tags for GitLab job matching
5. THE Egg_Config SHALL specify cloud provider preferences
6. WHEN an Egg_Config is updated in Nest, THE MotherGoose_Server SHALL apply changes to future runners
7. THE Egg_Config SHALL support environment variable injection into runners

### Requirement 13: Self-Management Jobs

**User Story:** As a system operator, I want automated self-management tasks, so that the Nest repository can maintain itself without manual intervention.

#### Acceptance Criteria

1. THE Jobs_Directory SHALL contain .fly files defining self-management tasks
2. THE System SHALL support jobs for secret rotation
3. THE System SHALL support jobs for Nest repository updates
4. THE System SHALL support jobs for runner image updates
5. WHEN a self-management job is triggered, THE MotherGoose_Server SHALL execute it on a dedicated runner
6. THE Self_Management_Job SHALL have elevated permissions for Nest modifications
7. THE System SHALL schedule recurring self-management jobs based on cron expressions

### Requirement 14: Database State Management

**User Story:** As a backend developer, I want persistent state storage, so that the system can recover from failures and maintain consistency.

#### Acceptance Criteria

1. THE System SHALL store runner state in YDB (Yandex Cloud) or DynamoDB (AWS)
2. THE Database SHALL track runner lifecycle (provisioning, active, dormant, terminated)
3. THE Database SHALL store Egg configurations
4. THE Database SHALL store job execution history
5. THE Database SHALL store UglyFox pruning policies
6. WHEN a backend server restarts, THE System SHALL restore state from the database
7. THE Database SHALL support transactional updates for state consistency

### Requirement 15: Monitoring and Observability

**User Story:** As a site reliability engineer, I want comprehensive monitoring, so that I can troubleshoot issues and optimize performance.

#### Acceptance Criteria

1. THE System SHALL emit metrics for runner provisioning time
2. THE System SHALL emit metrics for job execution duration
3. THE System SHALL emit metrics for runner pool sizes (Apex and Nadir)
4. THE System SHALL log all webhook events
5. THE System SHALL log all runner lifecycle transitions
6. THE System SHALL provide health check endpoints for all backend servers
7. THE System SHALL integrate with standard observability tools (Prometheus, Grafana)

### Requirement 16: Security and Authentication

**User Story:** As a security engineer, I want secure authentication and authorization, so that only authorized systems can manage runners.

#### Acceptance Criteria

1. THE System SHALL authenticate webhook requests from GitLab using shared secrets
2. THE System SHALL use IAM roles for cloud provider authentication
3. THE Runner SHALL authenticate with GitLab using runner tokens
4. THE System SHALL encrypt sensitive data at rest in the database
5. THE System SHALL encrypt runner communication with backend servers
6. THE System SHALL implement least-privilege access for all components
7. THE System SHALL support secret injection into runners from Yandex Cloud Lockbox, AWS Secrets Manager, and HashiCorp Vault
8. THE System SHALL use URI schemes to reference secrets: yc-lockbox://{secret-id}/{key}, aws-sm://{secret-name}/{key}, vault://{path}/{key}
9. WHEN secrets are referenced in .fly files, THE System SHALL mask secret values in logs and outputs
10. THE System SHALL retrieve secrets at runtime from the appropriate secret storage backend
11. THE System SHALL cache retrieved secrets with configurable TTL to minimize API calls
12. WHEN a secret reference is invalid or inaccessible, THE System SHALL fail deployment with a descriptive error message

### Requirement 17: Secret Storage and Management

**User Story:** As a security engineer, I want centralized secret management, so that sensitive credentials are never stored in configuration files or version control.

#### Acceptance Criteria

1. THE System SHALL store GitLab runner tokens in Yandex Cloud Lockbox (for Yandex Cloud deployments)
2. THE System SHALL store GitLab runner tokens in AWS Secrets Manager (for AWS deployments)
3. THE System SHALL support HashiCorp Vault as an optional secret backend
4. WHEN storing secrets, THE System SHALL use cloud-native encryption at rest
5. THE System SHALL support automatic secret rotation for GitLab runner tokens
6. WHEN a secret is rotated, THE System SHALL update all active runners with the new secret
7. THE System SHALL log secret access events for audit purposes (without logging secret values)
8. THE System SHALL enforce secret access policies based on IAM roles and service accounts
9. WHEN parsing .fly files, THE System SHALL validate secret URI syntax before deployment
10. THE System SHALL support secret versioning to enable rollback of secret changes

### Requirement 18: Development Workflow for Backend Servers

**User Story:** As a developer working on backend servers (MotherGoose, UglyFox), I want a standardized development workflow using Git worktrees, so that new features are isolated from production code until they are fully tested and approved.

#### Acceptance Criteria

1. THE Polar-Gosling-Backend-Servers_Repository (MotherGoose) SHALL use Git worktrees for feature development
2. ALL new feature development SHALL be performed in the "dev-new-features" worktree directory ONLY
3. THE main worktree SHALL remain untouched during feature development
4. WHEN implementing new features, developers SHALL work exclusively in "dev-new-features" directory
5. THE "dev-new-features" worktree SHALL contain a complete copy of the backend server codebase
6. ALL tests for new features SHALL be executed within the "dev-new-features" worktree
7. WHEN a feature is complete and tested, THE changes SHALL be committed in the "dev-new-features" worktree
8. ONLY after feature approval, THE changes SHALL be merged into the main worktree
9. THE development workflow SHALL prevent accidental modifications to production code in the main worktree
10. ALL CI/CD agents and automation tools SHALL respect the worktree isolation and work only in "dev-new-features" when developing new features

### Requirement 19: Backend Server Implementation Validation

**User Story:** As a developer working on backend servers (MotherGoose, UglyFox), I want automated validation of all code changes, so that I can ensure code quality, formatting, and correctness before committing changes.

#### Acceptance Criteria

1. ALL implementations in the "dev-new-features" worktree SHALL pass the full tox test suite before being considered complete
2. THE validation command `make mg-tox-all` SHALL be executed from the "dev-new-features" root directory
3. WHEN `make mg-tox-all` succeeds, THE implementation SHALL be considered successful
4. WHEN `make mg-tox-all` fails, THE developer SHALL fix all issues before proceeding
5. THE tox test suite SHALL include unit tests on Python 3.10, 3.11, 3.12, and 3.13
6. THE tox test suite SHALL include code formatting checks (black, isort)
7. THE tox test suite SHALL include style checks (flake8, pylint)
8. THE tox test suite SHALL include type checks (mypy)
9. ALL code SHALL achieve a pylint rating of 10/10
10. ALL code SHALL pass formatting checks with max line length of 120 characters
11. THE validation SHALL be performed before any feature is merged from "dev-new-features" to main worktree
12. THE System SHALL use uv for Python dependency management and package installation
13. WHEN installing dependencies, THE System SHALL use `uv sync --all-groups` to install all dependency groups
14. WHEN running tests, THE System SHALL use `uv run pytest` to execute tests in the uv-managed environment
15. THE System SHALL use uv for fast, reliable dependency resolution and installation

### Requirement 20: Code Comment Standards for Future Work

**User Story:** As a developer, I want a standardized format for marking future work in code comments, so that task references are concise, consistent, and easy to identify without cluttering the codebase.

#### Acceptance Criteria

1. ALL code comments referencing future work SHALL use the format "# Task N: Brief description"
2. THE task comment format SHALL NOT use "TODO" keyword
3. THE task comment SHALL reference the task number from tasks.md
4. THE task comment SHALL include a brief description of what needs to be done
5. THE total length of a task comment SHALL NOT exceed 42 characters
6. WHEN a task has subtasks, THE comment format SHALL be "# Task N.M: Brief description"
7. THE brief description SHALL be concise and action-oriented
8. THE task comment SHALL appear on a single line
9. WHEN task comments are added, developers SHALL ensure they reference valid task numbers from the implementation plan
10. THE System SHALL reject code with verbose or non-standard task comment formats during code review

### Requirement 21: Test Execution with YDB Static Port Mapping

**User Story:** As a developer running tests with YDB, I want the test execution agent to wait for test completion, so that parallel test execution does not fail due to YDB's static port mapping conflicts.

#### Acceptance Criteria

1. WHEN running tests using `tox`, THE Agent SHALL wait until all tests pass or fail before proceeding
2. WHEN running tests using `uv run pytest`, THE Agent SHALL wait until all tests pass or fail before proceeding
3. THE Agent SHALL NOT terminate test execution prematurely
4. THE Agent SHALL handle YDB static port mapping conflicts by preventing parallel test execution
5. WHEN YDB port conflicts occur, THE Agent SHALL serialize test execution to avoid port collisions
6. THE Agent SHALL report test results only after complete test suite execution
7. WHEN tests are running, THE Agent SHALL monitor test process completion status
8. THE Agent SHALL NOT assume test success until explicit pass/fail status is received
