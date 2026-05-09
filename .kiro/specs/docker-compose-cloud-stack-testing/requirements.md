# Requirements Document

## Introduction

This feature introduces a persistent `docker-compose`-based cloud stack for the
`Polar-Gosling-Backends` repository that emulates the full production pipeline
end-to-end on a developer workstation, a demo box, or a CI runner. Unlike the
existing `testcontainers` suite (which spins up short-lived, per-test containers
inside `pytest`), this stack is long-lived and orchestrated as a single unit via
`docker-compose`, so that developers and CI can exercise the complete
MotherGoose → UglyFox flow against a persistent YDB + LocalStack + Celery-broker
environment.

The Cloud_Stack emulates:

- YDB, holding `runners`, `egg_configs`, `sync_history`, `deployment_plans`,
  `audit_logs`, `tofu_versions`, and `gosling_version` tables.
- LocalStack, exposing S3 (artifact / binary cache), SQS (Celery transport and
  cross-service messaging), EventBridge (cloud-trigger emulation), and Secrets
  Manager (secret URI resolution).
- A dedicated Celery broker container that handles task queueing between
  MotherGoose and UglyFox.
- A Nest Git source container that plays the role of the production Nest
  repository.
- MotherGoose as both API and Celery worker containers, built from
  `dev-new-features/Dockerfile.mtg`.
- UglyFox as a Celery worker container, built from
  `dev-new-features/Dockerfile.uf`.
- A cloud-trigger emulator that periodically posts to
  `POST /internal/sync-git`, standing in for the Yandex Cloud Timer or AWS
  EventBridge rule used in production.

All files introduced by this feature MUST live under the `dev-new-features/`
folder (spec metadata under `.kiro/specs/` excepted), per the workspace
steering rules.

## Glossary

- **Cloud_Stack**: The full set of services orchestrated by the feature's
  `docker-compose.yml`, including YDB, LocalStack, the Celery broker, the Nest
  Git source, MotherGoose API and worker, UglyFox worker, the trigger emulator,
  and the seed job.
- **Compose_File**: The `docker-compose.yml` file (plus any
  `docker-compose.*.yml` override files) that defines Cloud_Stack.
- **Stack_Root**: The directory `dev-new-features/compose/` that contains
  Compose_File, initialization scripts, fixtures, the `.env.example` file, and
  the smoke-test script.
- **YDB_Container**: The container running `ydbplatform/local-ydb` (or
  equivalent local YDB image) that exposes the gRPC endpoint on port `2136` and
  the YDB console on port `8765`.
- **LocalStack_Container**: The container running `localstack/localstack` that
  exposes edge port `4566` with the services `s3`, `sqs`, `events`
  (EventBridge), and `secretsmanager` enabled.
- **Celery_Broker_Container**: The container that hosts the Celery broker used
  by MotherGoose and UglyFox.
- **Trigger_Emulator**: A dedicated container in Cloud_Stack that emits periodic
  HTTP `POST` requests to MotherGoose's `/internal/sync-git` endpoint, emulating
  a Yandex Cloud Timer or AWS EventBridge rule.
- **MotherGoose_API_Container**: The container built from
  `dev-new-features/Dockerfile.mtg` that runs the FastAPI application.
- **MotherGoose_Worker_Container**: The container built from
  `dev-new-features/Dockerfile.mtg` that runs the Celery worker on queue
  `mothergoose`.
- **UglyFox_Worker_Container**: The container built from
  `dev-new-features/Dockerfile.uf` that runs the Celery worker on queue
  `uglyfox`.
- **Nest_Git_Container**: A container that serves a sample Nest repository
  (`Eggs/`, `Jobs/`, `UF/`, `MG/`) over HTTP or `git://` so that the MotherGoose
  git-sync task can clone it.
- **Seed_Job**: A one-shot container that runs after infrastructure containers
  are healthy and initializes state: creates YDB tables, creates S3 buckets,
  creates SQS queues, seeds Secrets Manager entries, uploads initial binaries
  (OpenTofu, Gosling) to S3, creates EventBridge rules, and inserts seed rows
  into YDB.
- **Pipeline**: The end-to-end flow from a cloud trigger (or webhook) through
  git sync, `.fly` parsing, runner orchestration, Celery dispatch to UglyFox,
  and audit-log writes to YDB.
- **Stack_Makefile**: The set of Makefile targets added to
  `dev-new-features/Makefile` that start, stop, reset, tail, and smoke-test
  Cloud_Stack.
- **Compose_Profile**: A docker-compose profile used to toggle optional
  components (`full`, `infra-only`, `with-triggers`).
- **Pipeline_Smoke_Test**: The automated Python script that exercises Pipeline
  end-to-end against a running Cloud_Stack.
- **Secret_URI**: A string of the form `yc-lockbox://…`, `aws-sm://…`, or
  `vault://…` that identifies a secret in a production backend. Inside
  Cloud_Stack, Secret_URI instances resolve against LocalStack_Container's
  Secrets Manager service.

## Requirements

### Requirement 1: Docker Compose Stack Definition

**User Story:** As a backend developer, I want a single `docker-compose.yml` that launches the full Cloud_Stack, so that I can bring up MotherGoose, UglyFox, YDB, LocalStack, the Celery broker, the Nest Git source, and the trigger emulator with one command.

#### Acceptance Criteria

1. THE Stack_Root SHALL contain a file named `docker-compose.yml` at path `dev-new-features/compose/docker-compose.yml`, and THE Compose_File SHALL be parseable by `docker compose config` without errors.
2. THE Compose_File SHALL declare exactly the following nine services as top-level entries under the `services:` key, each with a non-empty configuration: `ydb`, `localstack`, `celery-broker`, `nest-git`, `mothergoose-api`, `mothergoose-worker`, `uglyfox-worker`, `trigger-emulator`, and `seed`.
3. THE Compose_File SHALL declare a user-defined bridge network named `pg-stack-net`, and every service listed in criterion 2 SHALL attach to `pg-stack-net` via its `networks:` key.
4. WHEN a developer runs `docker compose -f dev-new-features/compose/docker-compose.yml up -d` from Stack_Root, THE Cloud_Stack SHALL start every service whose assigned Compose_Profile is either unset (default/happy-path) or listed in the `COMPOSE_PROFILES` environment variable, and SHALL return exit code `0` within `180` seconds on a host with the required images already pulled.
5. IF any image reference in THE Compose_File omits an explicit tag or uses the tag `latest`, THEN THE Compose_File SHALL be considered invalid and SHALL fail a CI validation check that greps every `image:` entry for a `:` separator and rejects the literal suffix `:latest`.
6. THE Compose_File SHALL build MotherGoose_API_Container and MotherGoose_Worker_Container from Dockerfile `dev-new-features/Dockerfile.mtg`, and SHALL build UglyFox_Worker_Container from Dockerfile `dev-new-features/Dockerfile.uf`, with the `build.context` for all three services set to `dev-new-features/` (expressed relative to the compose file).
7. WHERE a service is not part of the happy-path pipeline (defined as the set: `ydb`, `localstack`, `celery-broker`, `nest-git`, `mothergoose-api`, `mothergoose-worker`, `uglyfox-worker`), THE Compose_File SHALL assign that service to at least one named Compose_Profile drawn from the closed set `{seed, triggers, debug}`, and that service SHALL NOT start under the default profile.
8. IF any declared service fails to reach a healthy or running state within `120` seconds of `docker compose up`, THEN THE Cloud_Stack SHALL surface the failure via non-zero exit code from `docker compose up --wait` and SHALL preserve container logs for inspection without removing the failed container.

### Requirement 2: YDB Database Container

**User Story:** As a backend developer, I want a YDB container in Cloud_Stack, so that both MotherGoose and UglyFox can read and write runner, egg, sync-history, deployment-plan, audit, and binary-version records.

#### Acceptance Criteria

1. THE YDB_Container SHALL use image `ydbplatform/local-ydb` pinned to an explicit semantic version tag (format `MAJOR.MINOR.PATCH` where each component is an integer between `0` and `9999`) defined in `.env.example`, and SHALL fail container startup with a non-zero exit code if the tag is absent, empty, or set to `latest`.
2. THE YDB_Container SHALL expose the gRPC endpoint on host port `2136` and the YDB console on host port `8765`, and IF either host port is already bound at stack startup, THEN THE Compose_File SHALL cause stack startup to fail with a non-zero exit code and an error message identifying the conflicting port.
3. THE YDB_Container SHALL set environment variable `YDB_USE_IN_MEMORY_PDISKS=true` so that all stored data is discarded when the container is stopped or removed, with no data persisted to host volumes.
4. THE Compose_File SHALL declare a Docker `healthcheck` for YDB_Container that probes the gRPC endpoint on port `2136`, runs every `10s` with a per-probe timeout of `5s`, allows a `120s` start period, and retries up to `12` times before marking the container `unhealthy`.
5. WHEN YDB_Container reports health status `healthy`, THE Compose_File SHALL start MotherGoose_API_Container, MotherGoose_Worker_Container, and UglyFox_Worker_Container, each declared with `depends_on` condition `service_healthy` on YDB_Container.
6. IF YDB_Container does not reach health status `healthy` within `120s` start period plus `12` retries at `10s` intervals (maximum `240s` total), THEN THE Compose_File SHALL mark YDB_Container `unhealthy`, prevent MotherGoose_API_Container, MotherGoose_Worker_Container, and UglyFox_Worker_Container from starting, and cause stack startup to exit with a non-zero status code.
7. WHEN YDB_Container reaches health status `healthy`, THE Seed_Job SHALL create the YDB tables `runners`, `egg_configs`, `sync_history`, `deployment_plans`, `audit_logs`, `tofu_versions`, and `gosling_version`, and SHALL exit with status code `0` within `60s` of the first table creation attempt.
8. IF the Seed_Job fails to create any of the seven required tables, THEN THE Seed_Job SHALL exit with a non-zero status code, emit an error message identifying the failed table name and the underlying cause, and leave any already-created tables in place without rollback.

### Requirement 3: LocalStack AWS Service Emulation

**User Story:** As a backend developer, I want LocalStack to emulate S3, SQS, EventBridge, and Secrets Manager in Cloud_Stack, so that MotherGoose and UglyFox exercise the real AWS SDK code paths against local endpoints.

#### Acceptance Criteria

1. THE LocalStack_Container SHALL use image `localstack/localstack` pinned to an explicit semantic version tag (format `MAJOR.MINOR.PATCH` or `MAJOR.MINOR`, not `latest` and not empty) defined in `.env.example`.
2. THE LocalStack_Container SHALL set environment variable `SERVICES` to exactly the value `s3,sqs,events,secretsmanager` with no additional or missing services.
3. THE LocalStack_Container SHALL expose edge port `4566` on the host bound to `127.0.0.1` and reachable via TCP within `60` seconds of container start.
4. THE Compose_File SHALL declare a Docker `healthcheck` for LocalStack_Container that issues an HTTP GET to `http://localhost:4566/_localstack/health`, runs every `5s` with a per-probe timeout of `5s`, allows a `60s` start period, and retries up to `12` times before the container is reported unhealthy.
5. IF the LocalStack_Container healthcheck endpoint does not return HTTP `200` within the start period and retry budget defined in criterion 4, THEN THE Compose_File SHALL cause dependent services (MotherGoose_API_Container, MotherGoose_Worker_Container, UglyFox_Worker_Container, Seed_Job) to not start and SHALL surface the unhealthy state via `docker compose ps`.
6. WHEN the LocalStack_Container healthcheck reports healthy, THE Seed_Job SHALL create the S3 bucket named exactly `polar-gosling-artifacts` in LocalStack_Container in region `us-east-1`, and SHALL treat an existing bucket with the same name as success (idempotent).
7. WHEN the LocalStack_Container healthcheck reports healthy, THE Seed_Job SHALL create the SQS queues named exactly `mothergoose` and `uglyfox` in LocalStack_Container in region `us-east-1`, and SHALL treat existing queues with the same names as success (idempotent).
8. WHEN the LocalStack_Container healthcheck reports healthy, THE Seed_Job SHALL create one Secrets Manager entry per distinct `aws-sm://` Secret_URI referenced in the seeded `egg_configs` rows, using the secret name parsed from the URI, and SHALL treat existing secrets with the same names as success (idempotent).
9. IF the Seed_Job fails to create any required S3 bucket, SQS queue, or Secrets Manager entry after `3` retry attempts spaced `2` seconds apart, THEN THE Seed_Job SHALL exit with a non-zero status code and SHALL emit an error message indicating the resource name and AWS service that failed, without leaving partially created resources undocumented.
10. THE Compose_File SHALL set `AWS_ENDPOINT_URL=http://localstack:4566`, `AWS_ACCESS_KEY_ID=test`, `AWS_SECRET_ACCESS_KEY=test`, and `AWS_DEFAULT_REGION=us-east-1` as environment variables for MotherGoose_API_Container, MotherGoose_Worker_Container, and UglyFox_Worker_Container, and these variables SHALL be present in each container's environment before the container process starts.

### Requirement 4: Celery Broker Container

**User Story:** As a backend developer, I want a dedicated Celery broker container in Cloud_Stack, so that MotherGoose can dispatch tasks to UglyFox and to its own worker through a production-shaped transport.

#### Acceptance Criteria

1. THE Celery_Broker_Container SHALL use an image pinned to an explicit semantic version tag (format `MAJOR.MINOR.PATCH`, no `latest` or floating tags) whose value is defined as a variable in `.env.example`.
2. THE Celery_Broker_Container SHALL expose its broker listener on a single TCP port, reachable by other containers on the `pg-stack-net` network and not published to the host by default.
3. THE Compose_File SHALL declare a Docker `healthcheck` for Celery_Broker_Container that verifies the broker accepts TCP connections on its broker port and responds to a broker-native liveness probe, runs every `5s` with a per-check `timeout` of `5s`, allows a `start_period` of `30s`, and retries up to `6` times before reporting `unhealthy`.
4. WHEN Celery_Broker_Container is started, THE Compose_File SHALL set `MOTHERGOOSE_BROKER_URL` and `MOTHERGOOSE_RESULT_BACKEND_URL` on MotherGoose_API_Container and MotherGoose_Worker_Container to non-empty URL values whose host resolves to the Celery_Broker_Container service name on `pg-stack-net` and whose port matches the broker listener port declared in criterion 2.
5. WHEN Celery_Broker_Container is started, THE Compose_File SHALL set `UGLYFOX_BROKER_URL` and `UGLYFOX_RESULT_BACKEND_URL` on UglyFox_Worker_Container to non-empty URL values whose host resolves to the Celery_Broker_Container service name on `pg-stack-net` and whose port matches the broker listener port declared in criterion 2.
6. THE Compose_File SHALL declare that MotherGoose_API_Container, MotherGoose_Worker_Container, and UglyFox_Worker_Container depend on Celery_Broker_Container with condition `service_healthy`, so that none of these containers start before Celery_Broker_Container reports `healthy` per criterion 3.
7. IF Celery_Broker_Container fails its healthcheck for more than `6` consecutive retries or exits with a non-zero status, THEN THE Compose_File SHALL mark Celery_Broker_Container as `unhealthy`, prevent MotherGoose_API_Container, MotherGoose_Worker_Container, and UglyFox_Worker_Container from starting, and preserve the declared broker volume (if any) so no queued messages are deleted by the failure itself.

### Requirement 5: Cloud Trigger Emulation

**User Story:** As a backend developer, I want a trigger emulator that periodically invokes MotherGoose's internal sync endpoint, so that Cloud_Stack reproduces the cloud timer or EventBridge rule that drives git sync in production.

#### Acceptance Criteria

1. WHILE Trigger_Emulator is running, THE Trigger_Emulator SHALL issue an HTTP `POST` to `http://mothergoose-api:8000/internal/sync-git` once per configured interval with a request timeout of `10` seconds.
2. THE Trigger_Emulator SHALL read the interval in whole seconds from environment variable `TRIGGER_SYNC_INTERVAL_SECONDS`, SHALL accept integer values from `5` to `3600` inclusive, and SHALL default to `60` when the variable is unset or empty.
3. IF `TRIGGER_SYNC_INTERVAL_SECONDS` is set to a non-integer value or a value outside the range `5` to `3600` inclusive, THEN THE Trigger_Emulator SHALL log an error indicating the invalid interval and SHALL fall back to the default of `60` seconds.
4. THE Trigger_Emulator SHALL attach a request header named `X-Internal-Token` whose value is read from environment variable `INTERNAL_SYNC_TOKEN` on each invocation.
5. IF `INTERNAL_SYNC_TOKEN` is unset or empty at startup, THEN THE Trigger_Emulator SHALL log an error indicating the missing token and SHALL exit with a non-zero status code without issuing any HTTP `POST`.
6. IF the HTTP `POST` to MotherGoose_API_Container fails due to connection refused, DNS resolution failure, timeout, or any HTTP response status code outside the range `200` to `299` inclusive, THEN THE Trigger_Emulator SHALL log the failure at level `WARN` with the failure reason and SHALL retry on the next scheduled interval without crashing the process.
7. WHERE Compose_Profile `with-triggers` is disabled, THE Compose_File SHALL exclude Trigger_Emulator from the set of started services.
8. WHEN Trigger_Emulator completes an invocation attempt, THE Trigger_Emulator SHALL emit a single structured log line containing an ISO-8601 UTC timestamp, the HTTP status code (or the failure reason when no response was received), and the response duration in milliseconds measured from request send to response received or failure detection.

### Requirement 6: MotherGoose Service Containers

**User Story:** As a backend developer, I want MotherGoose to run as both an API container and a worker container in Cloud_Stack, so that the full FastAPI plus Celery topology is exercised end-to-end.

#### Acceptance Criteria

1. WHEN Cloud_Stack is started, THE MotherGoose_API_Container SHALL run the command `uvicorn app.main:app --host 0.0.0.0 --port 8000` and SHALL bind and expose TCP port `8000` on the host within `60s` of container start.
2. WHEN Cloud_Stack is started, THE MotherGoose_Worker_Container SHALL run the command `celery -A app.celery_worker worker --loglevel=info -Q mothergoose` and SHALL consume tasks only from the `mothergoose` queue.
3. THE Compose_File SHALL set the environment variables `MOTHERGOOSE_DATABASE_TYPE=ydb`, `MOTHERGOOSE_YDB_ENDPOINT=grpc://ydb:2136`, and `MOTHERGOOSE_YDB_DATABASE=/local` with identical values on both MotherGoose_API_Container and MotherGoose_Worker_Container.
4. THE Compose_File SHALL set `MOTHERGOOSE_NEST_REPO_URL` on both MotherGoose_API_Container and MotherGoose_Worker_Container to the in-network URL of Nest_Git_Container reachable over the shared Compose network using the Nest_Git_Container service name as hostname.
5. THE Compose_File SHALL set `MOTHERGOOSE_INTERNAL_SYNC_TOKEN` on MotherGoose_API_Container to a non-empty string with length between `16` and `128` characters that is byte-for-byte identical to `INTERNAL_SYNC_TOKEN` on Trigger_Emulator.
6. THE Compose_File SHALL declare a Docker `healthcheck` for MotherGoose_API_Container that issues a `GET` request to `http://localhost:8000/health` with a per-probe timeout of `5s`, runs every `10s`, allows a `60s` start period during which failing probes do not count toward the retry limit, and retries up to `5` times before reporting unhealthy.
7. WHEN the MotherGoose_API_Container `/health` probe returns an HTTP `2xx` status within the `5s` probe timeout, THE Compose_File SHALL mark MotherGoose_API_Container as `healthy`.
8. IF the MotherGoose_API_Container `/health` probe fails `5` consecutive times after the `60s` start period (due to non-`2xx` status, connection refusal, or probe timeout), THEN THE Compose_File SHALL mark MotherGoose_API_Container as `unhealthy` and SHALL preserve the container in a running state without automatic restart.

### Requirement 7: UglyFox Service Container

**User Story:** As a backend developer, I want UglyFox to run as a worker container in Cloud_Stack, so that lifecycle Celery tasks dispatched by MotherGoose are actually consumed and processed.

#### Acceptance Criteria

1. WHEN Cloud_Stack is started, THE UglyFox_Worker_Container SHALL execute the command `celery -A app.celery_worker worker --loglevel=info -Q uglyfox` as its entrypoint, bind exclusively to the `uglyfox` queue, and begin consuming tasks within `45` seconds of container start.
2. THE Compose_File SHALL set the environment variables `UGLYFOX_DATABASE_TYPE=ydb`, `UGLYFOX_YDB_ENDPOINT=grpc://ydb:2136`, and `UGLYFOX_YDB_DATABASE=/local` on UglyFox_Worker_Container, with no additional `UGLYFOX_*` variables injected by the compose definition.
3. THE Compose_File SHALL declare a Docker `healthcheck` for UglyFox_Worker_Container that reports healthy only when at least one process whose command line contains `celery` and `worker` is running, executes every `15` seconds, allows a start period of `45` seconds during which failures do not mark the container unhealthy, and marks the container unhealthy after `4` consecutive failed checks.
4. IF any probe of the UglyFox_Worker_Container healthcheck does not complete within `10` seconds, THEN THE Compose_File SHALL cause that probe to be recorded as a failed attempt counting toward the `4`-retry limit defined in criterion 3.
5. THE Compose_File SHALL declare that UglyFox_Worker_Container depends on YDB_Container, LocalStack_Container, and Celery_Broker_Container with condition `service_healthy`, and on Seed_Job with condition `service_completed_successfully`, and SHALL not start UglyFox_Worker_Container until all four dependencies have reached their required state.
6. IF any dependency listed in criterion 5 fails to reach its required state, THEN THE Compose_File SHALL prevent UglyFox_Worker_Container from starting and SHALL surface a startup error indicating which dependency failed.

### Requirement 8: Nest Git Source Container

**User Story:** As a backend developer, I want a Git source inside Cloud_Stack containing a sample Nest repository, so that MotherGoose can clone it and exercise the real git-sync plus `.fly` parsing path.

#### Acceptance Criteria

1. THE Nest_Git_Container SHALL expose a Git repository named `nest.git` over HTTP on a port declared in Compose_File, reachable from other Cloud_Stack containers via the in-network hostname `nest-git` and support unauthenticated anonymous `git clone` operations.
2. THE Nest_Git_Container SHALL populate `nest.git` with a top-level directory structure containing exactly the directories `Eggs/`, `Jobs/`, `UF/`, and `MG/`, where each directory contains at least one `.fly` file that parses successfully via the Gosling parser without errors and is committed on the default branch `main` with at least one commit whose SHA is resolvable via `git rev-parse HEAD`.
3. IF the Nest_Git_Container fails to start, fails its readiness probe within `60` seconds of container start, or returns a non-2xx response to an unauthenticated `git clone` request, THEN THE Cloud_Stack SHALL mark the container as unhealthy, prevent dependent containers from starting, and surface an error indicating Nest_Git_Container startup failure.
4. WHEN the Seed_Job executes during Cloud_Stack initialization, THE Seed_Job SHALL insert at least one `egg_configs` row into YDB_Container whose `git_repo_url_secret` field, when resolved via LocalStack_Container Secrets Manager, returns the in-network clone URL of Nest_Git_Container pointing to `nest.git`.
5. WHEN MotherGoose_Worker_Container processes a git-sync task targeting Nest_Git_Container, THE MotherGoose_Worker_Container SHALL write exactly one `sync_history` row with status `SUCCESS` to YDB_Container within `60` seconds of task dispatch.
6. IF a git-sync task targeting Nest_Git_Container fails to clone the repository, fails to parse any `.fly` file, or does not complete within `60` seconds of task dispatch, THEN THE MotherGoose_Worker_Container SHALL write exactly one `sync_history` row with status `FAILED` to YDB_Container containing an error message indicating the failure cause and preserve any previously persisted `egg_configs` rows unchanged.

### Requirement 9: Stack Seed and Initialization

**User Story:** As a backend developer, I want a one-shot seed container that sets up YDB tables, S3 buckets, SQS queues, Secrets Manager entries, and EventBridge rules, so that Cloud_Stack is ready for tests immediately after startup.

#### Acceptance Criteria

1. THE Seed_Job SHALL run as a Compose service with `restart: no` and SHALL complete or fail within `120` seconds of container start.
2. WHEN Seed_Job is executed, THE Seed_Job SHALL perform each of the following operations and SHALL exit with status code `0` only after all of them complete successfully: YDB schema creation for every table defined in the MotherGoose and UglyFox schema modules, S3 bucket creation for every bucket name listed in the Seed_Job configuration, SQS queue creation for every queue name listed in the Seed_Job configuration, Secrets Manager entry creation for every secret name listed in the Seed_Job configuration, binary artifact upload to S3 for every artifact path listed in the Seed_Job configuration, and seed-row insertion into YDB for every row defined in the Seed_Job configuration.
3. WHERE Compose_Profile `with-triggers` is enabled, THE Seed_Job SHALL additionally create every EventBridge rule listed in the Seed_Job configuration before exiting with status code `0`.
4. IF any Seed_Job operation fails after exhausting `3` retry attempts with a delay of `2` seconds between attempts, THEN THE Seed_Job SHALL exit with a non-zero status code, SHALL emit an error log line containing the name of the failed operation and the target resource identifier, and SHALL leave resources created by prior successful operations in place without rollback.
5. WHEN Seed_Job runs against a stack where one or more target resources already exist with matching identifiers, THE Seed_Job SHALL treat each pre-existing resource as a successful operation without modification, SHALL produce the same final resource inventory as a run against an empty stack, and SHALL exit with status code `0`.
6. THE Compose_File SHALL declare that MotherGoose_API_Container, MotherGoose_Worker_Container, UglyFox_Worker_Container, and Trigger_Emulator depend on Seed_Job with condition `service_completed_successfully`.

### Requirement 10: Stack Control via Makefile

**User Story:** As a backend developer, I want Makefile targets to control Cloud_Stack, so that I have a consistent entry point for starting, stopping, and resetting the test environment.

#### Acceptance Criteria

1. WHEN the user invokes `make compose-up`, THE Stack_Makefile SHALL start Cloud_Stack in detached mode and SHALL block until MotherGoose_API_Container reports health status `healthy` or until `180` seconds elapse.
2. IF MotherGoose_API_Container does not reach health status `healthy` within `180` seconds during `make compose-up`, THEN THE Stack_Makefile SHALL exit with a non-zero status code and SHALL emit an error log line naming MotherGoose_API_Container and its last observed health status.
3. WHEN the user invokes `make compose-down`, THE Stack_Makefile SHALL stop and remove every Cloud_Stack container and SHALL preserve every Cloud_Stack named volume.
4. WHEN the user invokes `make compose-reset`, THE Stack_Makefile SHALL stop Cloud_Stack, remove every Cloud_Stack named volume, and start Cloud_Stack again in detached mode, blocking until MotherGoose_API_Container reports health status `healthy` or until `180` seconds elapse.
5. WHEN the user invokes `make compose-logs`, THE Stack_Makefile SHALL stream the combined stdout and stderr logs of every Cloud_Stack service to the terminal until the user interrupts the command.
6. WHEN the user invokes `make compose-smoke` against a running Cloud_Stack, THE Stack_Makefile SHALL execute Pipeline_Smoke_Test and SHALL exit with status code `0` on success and a non-zero status code on failure.
7. IF Cloud_Stack is not running when the user invokes `make compose-smoke`, THEN THE Stack_Makefile SHALL exit with a non-zero status code and SHALL emit an error log line indicating that Cloud_Stack is not running.
8. WHEN the user invokes `make compose-clean`, THE Stack_Makefile SHALL remove every Cloud_Stack container, every Cloud_Stack named volume, and every dangling Cloud_Stack image.
9. IF the `docker` CLI or `docker compose` plugin is not available on the user's PATH when the user invokes any Cloud_Stack Makefile target, THEN THE Stack_Makefile SHALL exit with a non-zero status code within `5` seconds and SHALL emit an error log line naming the missing dependency (`docker` or `docker compose`).

### Requirement 11: Environment Configuration File

**User Story:** As a backend developer, I want a single `.env.example` file that documents every environment variable the stack reads, so that I can copy it to `.env` and run Cloud_Stack without guessing.

#### Acceptance Criteria

1. THE Stack_Root SHALL contain a file named `.env.example` at the top level of the repository, readable by the developer's user account.
2. THE `.env.example` file SHALL document every environment variable consumed by Compose_File, including at minimum `TRIGGER_SYNC_INTERVAL_SECONDS`, `INTERNAL_SYNC_TOKEN`, `LOCALSTACK_IMAGE_TAG`, `YDB_IMAGE_TAG`, `CELERY_BROKER_IMAGE_TAG`, and `AWS_DEFAULT_REGION`, with exactly one variable declared per line in `KEY=value` form.
3. THE `.env.example` file SHALL declare, for each documented variable, either a concrete default value or an empty value followed by an inline comment containing the token `REQUIRED` to mark it as mandatory.
4. THE `.env.example` file SHALL precede each documented variable with a comment line describing the variable's purpose and its accepted value range or format.
5. THE Compose_File SHALL reference every environment variable listed in `.env.example` using the `${VAR:-default}` syntax whenever that variable has a documented default in `.env.example`, and using the `${VAR:?error message}` syntax whenever the variable is marked `REQUIRED`.
6. IF a variable marked `REQUIRED` in `.env.example` is unset or empty at stack-up time, THEN THE Compose_File SHALL abort the `up` operation with a non-zero exit code and emit an error message that names the missing variable and references `.env.example`.
7. IF the set of variables declared in `.env.example` differs from the set of variables referenced by Compose_File, THEN THE Stack_Root validation SHALL fail with an error message listing each variable that is declared but unreferenced and each variable that is referenced but undeclared.

### Requirement 12: End-to-End Pipeline Smoke Test

**User Story:** As a backend developer, I want an end-to-end smoke test that drives a full pipeline run against Cloud_Stack, so that I can prove every component is wired correctly before running deeper tests.

#### Acceptance Criteria

1. THE Pipeline_Smoke_Test SHALL be implemented as a Python script located at `dev-new-features/compose/scripts/smoke_test.py`.
2. WHEN Pipeline_Smoke_Test runs against a healthy Cloud_Stack, THE Pipeline_Smoke_Test SHALL execute the following steps in the listed order, proceeding to the next step only after the current step succeeds: (a) issue `GET /health` on MotherGoose_API_Container and verify the response status is HTTP `200` within `10` seconds; (b) issue `POST /internal/sync-git` on MotherGoose_API_Container and verify the response status is HTTP `202` within `10` seconds; (c) poll the `sync_history` table in YDB_Container at an interval of `2` seconds until a row with status equal to `SUCCESS` is observed or `60` seconds elapse; (d) query the `egg_configs` table in YDB_Container and verify that at least `1` row is present; (e) send a mock GitLab webhook request to MotherGoose_API_Container that enqueues a runner-deployment task and verify the response status is HTTP `202` within `10` seconds; (f) poll the `audit_logs` table in YDB_Container at an interval of `2` seconds until at least `1` row is observed or `60` seconds elapse.
3. IF any individual step defined in criterion 2 does not complete successfully within `120` seconds of that step starting, THEN THE Pipeline_Smoke_Test SHALL stop execution, SHALL emit a single error log line to standard error that names the timed-out step identifier (a through f), and SHALL exit with status code `1`.
4. IF any step defined in criterion 2 completes but fails its verification (non-matching HTTP status, missing row, or status not equal to `SUCCESS`), THEN THE Pipeline_Smoke_Test SHALL stop execution, SHALL emit a single error log line to standard error naming the failed step identifier and the observed failure reason, and SHALL exit with status code `1`.
5. WHEN every step defined in criterion 2 completes successfully, THE Pipeline_Smoke_Test SHALL exit with status code `0`.
6. WHERE environment variable `SMOKE_TEST_VERBOSE` is set to the string `"1"`, THE Pipeline_Smoke_Test SHALL emit to standard output one log line per step containing the step identifier and its elapsed duration in milliseconds as a non-negative integer.

### Requirement 13: Isolation From Existing Testcontainers Suite

**User Story:** As a backend developer, I want Cloud_Stack to remain isolated from the existing `pytest` testcontainers suite, so that running one does not interfere with the other.

#### Acceptance Criteria

1. THE Compose_File SHALL assign every Cloud_Stack service a `container_name` attribute whose value begins with the literal prefix `pg-stack-` and is unique within the Compose_File.
2. THE Compose_File SHALL declare every Cloud_Stack named volume with a top-level `name` attribute whose value begins with the literal prefix `pg-stack-`.
3. THE Compose_File SHALL declare every Cloud_Stack network with a `name` attribute whose value begins with the literal prefix `pg-stack-`, and Cloud_Stack services SHALL NOT attach to any network not declared inside the Compose_File.
4. WHERE a Cloud_Stack service does not declare an explicit host-side port in its `ports` mapping, THE Compose_File SHALL NOT publish any host port for that service, and the service SHALL be reachable from other Cloud_Stack services only over the Cloud_Stack internal network.
5. WHEN Cloud_Stack is running and occupying any host port that a testcontainers container requires, and the developer invokes `make mg-tox-all` or `make uf-tox-all`, THE testcontainers suite SHALL exit with a non-zero status within `120` seconds of suite startup and SHALL emit an error message that names the conflicting host port number.

### Requirement 14: Data Reset and Tear-down

**User Story:** As a backend developer, I want a deterministic way to reset Cloud_Stack state between runs, so that tests do not depend on residual data.

#### Acceptance Criteria

1. WHEN the developer runs `make compose-reset`, THE Stack_Makefile SHALL remove every named volume owned by Cloud_Stack and SHALL restart the stack within `120` seconds, returning exit code `0` on success.
2. IF `make compose-reset` fails to remove any Cloud_Stack named volume, THEN THE Stack_Makefile SHALL abort the restart, SHALL emit an error message naming the volume that failed removal, and SHALL exit with a non-zero exit code.
3. WHEN the developer runs `make compose-down`, THE Stack_Makefile SHALL stop every Cloud_Stack container within `60` seconds and SHALL preserve every Cloud_Stack named volume with its data intact.
4. WHEN Seed_Job starts a seed operation, THE Seed_Job SHALL check for each pre-defined seeded resource before creating it.
5. IF Seed_Job detects an existing seeded resource on re-run, THEN Seed_Job SHALL emit an informational log line naming the existing resource, SHALL leave the existing resource unmodified, and SHALL continue executing the remaining seed operations without aborting.
6. IF a seed operation fails for a reason other than the resource already existing, THEN Seed_Job SHALL emit an error log line naming the failing resource and the failure reason, and SHALL exit with a non-zero exit code after attempting all remaining independent seed operations.
7. WHEN the developer runs `make compose-clean`, THE Stack_Makefile SHALL remove every Cloud_Stack container, SHALL remove every Cloud_Stack named volume, and SHALL remove every dangling Cloud_Stack image, returning exit code `0` on success.
8. IF `make compose-clean` cannot remove any Cloud_Stack container, named volume, or dangling image, THEN THE Stack_Makefile SHALL emit an error message naming the resource that failed removal and SHALL exit with a non-zero exit code.

### Requirement 15: Documentation

**User Story:** As a new contributor, I want a README that explains how to run Cloud_Stack, so that I can get a working integration environment on my first attempt.

#### Acceptance Criteria

1. THE Stack_Root SHALL contain a file named `README.md` whose body includes all of the following sections, each introduced by a Markdown heading: (a) purpose of Cloud_Stack, (b) enumerated list of every service started by Cloud_Stack with a one-line description per service, (c) enumerated list of every supported Stack_Makefile target with its effect, (d) enumerated list of every environment variable consumed by Cloud_Stack with name, purpose, default value, and whether it is required, (e) the exact smoke-test command, and (f) a troubleshooting section listing at least three distinct failure scenarios with diagnosis steps and remediation.
2. THE `README.md` SHALL contain a worked end-to-end example for Pipeline that lists, in order, every command a contributor must execute starting with `make compose-up` and ending with `make compose-smoke`, and SHALL show the expected terminal output or exit code for each command so that a reader can verify success without running additional commands.
3. THE `README.md` SHALL state the minimum required Docker Engine version as a specific semantic version (major.minor.patch) and the minimum required `docker compose` plugin version as a specific semantic version (major.minor.patch), and SHALL provide the exact commands a contributor runs to print the installed versions for verification.
4. WHERE the host platform is Windows, THE `README.md` SHALL document the required Docker Desktop minimum version, the required WSL2 distribution and minimum kernel version, the required file-sharing or resource settings, and the exact steps to validate the configuration before running `make compose-up`.
5. IF any command listed in the worked example in criterion 2 exits with a non-zero exit code on a host that meets the versions stated in criterion 3 and, on Windows, the configuration in criterion 4, THEN the troubleshooting section required by criterion 1 SHALL contain an entry that names that command, describes the expected failure indication, and lists the remediation steps.

### Requirement 16: CI Integration

**User Story:** As a repository maintainer, I want the Cloud_Stack smoke test to run in CI, so that regressions in the cross-service pipeline are caught before they land on `main`.

#### Acceptance Criteria

1. THE Stack_Root SHALL contain a GitHub Actions workflow file at `dev-new-features/.github/workflows/compose-smoke.yml` that triggers on pull request events of type `opened`, `synchronize`, and `reopened` targeting the `dev-new-features` branch.
2. WHEN the compose-smoke workflow runs, THE workflow SHALL execute `make compose-up`, then `make compose-smoke`, then `make compose-down` in that order, where each step only proceeds if the previous step exited with status code `0`.
3. IF `make compose-up` or `make compose-smoke` exits with a non-zero status code, THEN the compose-smoke workflow SHALL still execute `make compose-down` as a cleanup step and SHALL mark the workflow run as failed.
4. IF `make compose-smoke` exits with a non-zero status code, THEN the compose-smoke workflow SHALL upload the combined Cloud_Stack logs as a workflow artifact named `compose-logs` with a retention period of `7` days, and the artifact SHALL be available for download from the workflow run page.
5. THE compose-smoke workflow SHALL reach a terminal state (success, failure, or cancelled) within `15` minutes of wall-clock time measured from workflow start to workflow end, and IF the `15` minute limit is exceeded, THEN the workflow SHALL be cancelled and marked as failed.
