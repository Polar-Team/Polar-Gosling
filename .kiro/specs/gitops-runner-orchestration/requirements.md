# Requirements Document

## Introduction

The Polar Gosling GitOps Runner Orchestration system is a comprehensive platform for managing CI/CD runners across multiple cloud providers (Yandex Cloud and AWS) using a GitOps approach. The system consists of multiple backend servers (MotherGoose, UglyFox), a custom configuration language (.fly), a CLI tool for bootstrapping and management, and support for both serverless containers and autoscaling VMs as runner execution environments. The main repository (Nest) manages multiple downstream repositories (Eggs) through declarative configuration files.

## Glossary

- **Nest**: The main GitOps repository that configures and manages runners for downstream repositories
- **Egg**: A single managed repository configured by the Nest
- **EggsBucket**: A group of multiple managed repositories configured together
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

## Requirements

### Requirement 1: Nest Repository Structure

**User Story:** As a DevOps engineer, I want a standardized Nest repository structure, so that I can organize managed repositories, jobs, and policies in a predictable way.

#### Acceptance Criteria

1. THE Nest_Repository SHALL contain an "Eggs" directory for managed repository configurations
2. THE Nest_Repository SHALL contain a "Jobs" directory for self-management task definitions
3. THE Nest_Repository SHALL contain a "UF" directory for UglyFox configuration
4. WHEN an Egg is a single project, THE Eggs_Directory SHALL contain a project folder with a config.fly file
5. WHEN multiple projects are grouped, THE Eggs_Directory SHALL contain an EggsBucket folder with a config.fly file
6. THE Jobs_Directory SHALL contain files named with pattern "{job_name}.fly"
7. THE UF_Directory SHALL contain a file named "config.fly" with runner pruning policies

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

#### Acceptance Criteria

1. THE MotherGoose_Server SHALL receive webhooks from GitLab repositories
2. WHEN a webhook is received, THE MotherGoose_Server SHALL parse the event payload
3. THE MotherGoose_Server SHALL match webhook events to configured Eggs
4. THE MotherGoose_Server SHALL determine the appropriate runner type (serverless or VM) based on job requirements
5. THE MotherGoose_Server SHALL deploy runners using OpenTofu/Terraform configurations
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
7. THE System SHALL support secret injection into runners from secure storage (Vault, AWS Secrets Manager)
