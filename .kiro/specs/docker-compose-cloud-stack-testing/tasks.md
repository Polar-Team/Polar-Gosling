# Implementation Plan: Docker Compose Cloud Stack Testing

## Overview

This plan turns the `docker-compose-cloud-stack-testing` design into discrete,
incremental coding tasks. The target implementation directory is
`Polar-Gosling-Backends/dev-new-features/compose/` (a new top-level directory
under the existing `dev-new-features/` tree, per the workspace rule that "all
new code must be located under `dev-new-features/`").

**Implementation language:** Python 3.13 for all scripts (`seed.py`,
`trigger.py`, `smoke_test.py`, static checks, and `pytest` + `hypothesis`
property tests). Shell / GNU make for the Makefile targets. YAML for the
Compose file and the CI workflow. Dockerfiles for the custom images.

**Key decisions baked in from the design:**

- Nine Compose services on a single user-defined bridge network `pg-stack-net`.
- Redis chosen as the Celery broker (see design §3 — "Choice of broker").
- Seed job reuses the MotherGoose image as its base so it inherits `ydb` and
  `boto3` dependencies without re-building them.
- Trigger emulator and seed job ship as tiny Python entrypoints in their own
  Dockerfiles under `compose/trigger/` and `compose/seed/`.
- Every container, volume, and network name is prefixed `pg-stack-` so the
  stack cannot collide with the existing `testcontainers` suite.
- Property tests use `hypothesis` (already in `tech.md`) and live under
  `compose/tests/`; they can be run stand-alone from the stack.

**Structure conventions:**

- Top-level tasks group related work; sub-tasks are the actual commits.
- Sub-tasks marked `*` are optional (tests). They MUST still be listed in the
  dependency graph so they can be scheduled.
- Every sub-task ends with `_Requirements: X.Y_` tracing back to the
  requirements document; property-test sub-tasks also reference the
  correctness property number from the design document.

## Tasks

- [x] 1. Scaffold `compose/` directory, `.env.example`, and CI workflow skeleton
  - [x] 1.1 Create the `dev-new-features/compose/` directory tree
    - Create subdirectories `compose/`, `compose/nest/{Eggs/sample-egg,Jobs,UF,MG}/`, `compose/seed/fixtures/`, `compose/seed/artifacts/`, `compose/trigger/`, `compose/nest-git/`, `compose/scripts/`, `compose/tests/`.
    - Add a `.gitkeep` in any empty subdirectory so the structure is committed.
    - _Requirements: 1.1_

  - [x] 1.2 Author `compose/.env.example` with the full variable schema
    - Document every variable from the design's "Environment variable wiring" section: `YDB_IMAGE_TAG`, `LOCALSTACK_IMAGE_TAG`, `CELERY_BROKER_IMAGE_TAG`, `NEST_GIT_IMAGE_TAG`, `MG_IMAGE_TAG`, `UF_IMAGE_TAG`, `INTERNAL_SYNC_TOKEN`, `TRIGGER_SYNC_INTERVAL_SECONDS`, `AWS_DEFAULT_REGION`, `MOTHERGOOSE_API_HOST_PORT`.
    - Precede each variable with a one-line comment documenting its purpose and accepted range or format.
    - Mark variables without a concrete default with a trailing `# REQUIRED` comment; give the rest a non-empty default value.
    - Exactly one `KEY=value` declaration per line; no trailing whitespace.
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

  - [x] 1.3 Create an empty CI workflow placeholder at `dev-new-features/.github/workflows/compose-smoke.yml`
    - File exists with just a `name: compose-smoke` header and a TODO comment so later tasks can extend it without creating the file twice.
    - _Requirements: 16.1_

- [x] 2. Define `docker-compose.yml` infrastructure-tier services
  - [x] 2.1 Create `compose/docker-compose.yml` skeleton with networks and volumes
    - Declare the top-level `name: pg-stack` project name.
    - Declare top-level `networks.pg-stack-net` with explicit `name: pg-stack-net` and driver `bridge`.
    - Declare top-level `volumes.pg-stack-localstack-data` and `volumes.pg-stack-celery-broker-data`, each with an explicit `name:` matching the key.
    - Add `services: {}` placeholder so subsequent sub-tasks can append without editing an empty file.
    - _Requirements: 1.1, 1.3, 13.2, 13.3_

  - [x] 2.2 Add the `ydb` service to `docker-compose.yml`
    - `container_name: pg-stack-ydb`, image `ydbplatform/local-ydb:${YDB_IMAGE_TAG:?set YDB_IMAGE_TAG in .env}`, `YDB_USE_IN_MEMORY_PDISKS=true`.
    - Publish `127.0.0.1:2136:2136` (gRPC) and `127.0.0.1:8765:8765` (console), no volumes.
    - Attach to `pg-stack-net`, healthcheck probes gRPC on port 2136 with `interval=10s`, `timeout=5s`, `start_period=120s`, `retries=12`.
    - _Requirements: 1.2, 1.3, 2.1, 2.2, 2.3, 2.4, 13.1, 13.4_

  - [x] 2.3 Add the `localstack` service to `docker-compose.yml`
    - `container_name: pg-stack-localstack`, image `localstack/localstack:${LOCALSTACK_IMAGE_TAG:?set LOCALSTACK_IMAGE_TAG in .env}`, env `SERVICES=s3,sqs,events,secretsmanager`, `DEFAULT_REGION=us-east-1`, `LS_LOG=warn`, `PERSISTENCE=0`.
    - Publish `127.0.0.1:4566:4566`; mount `pg-stack-localstack-data:/var/lib/localstack`.
    - Healthcheck `curl -fsS http://localhost:4566/_localstack/health` with `interval=5s`, `timeout=5s`, `start_period=60s`, `retries=12`.
    - _Requirements: 1.2, 3.1, 3.2, 3.3, 3.4, 13.4_

  - [x] 2.4 Add the `celery-broker` service to `docker-compose.yml`
    - `container_name: pg-stack-celery-broker`, image `redis:${CELERY_BROKER_IMAGE_TAG:?set CELERY_BROKER_IMAGE_TAG in .env}`, no published host ports.
    - Mount `pg-stack-celery-broker-data:/data` (per design — preserved on `compose-down`).
    - Healthcheck `["CMD", "redis-cli", "ping"]` with `interval=5s`, `timeout=5s`, `start_period=30s`, `retries=6`.
    - _Requirements: 1.2, 4.1, 4.2, 4.3, 13.1, 13.2, 13.4_

  - [x] 2.5 Write property test `test_compose_properties.py::test_network_attachment`
    - **Property 1: Every Cloud_Stack service attaches to `pg-stack-net`**
    - **Validates: Requirements 1.3, 13.3**
    - Use `python-dotenv` + `pyyaml` (or `docker compose config --format json` via subprocess) in a `hypothesis`-driven test that substitutes arbitrary values for the env vars that have defaults and asserts that every resolved service attaches to `pg-stack-net` and that no service declares a network absent from the top-level `networks` map.

  - [x] 2.6 Write property test `test_compose_properties.py::test_image_pins`
    - **Property 2: No Compose `image:` reference is floating or unpinned**
    - **Validates: Requirements 1.5, 2.1, 3.1, 4.1**
    - Hypothesis strategy generates legal `MAJOR.MINOR.PATCH` and `MAJOR.MINOR` pins for image-tag env vars, and asserts each resolved `image:` contains a `:`, does not end with `:latest`, is non-empty, and — for env-driven tags — matches the regex `^\d{1,4}\.\d{1,4}(\.\d{1,4})?$`.

  - [x] 2.7 Write property test `test_compose_properties.py::test_internal_only_services_publish_no_ports`
    - **Property 19: Internal-only services publish no host ports**
    - **Validates: Requirement 13.4**
    - Assert that services `celery-broker`, `mothergoose-worker`, `uglyfox-worker`, `trigger-emulator`, `seed`, and `nest-git` declare no `ports:` key or an empty list.

  - [x] 2.8 Write property test `test_compose_properties.py::test_resource_name_prefixes`
    - **Property 18: All stack-owned resources carry the `pg-stack-` prefix**
    - **Validates: Requirements 13.1, 13.2, 13.3**
    - Assert every `container_name`, every top-level volume key and its `name`, and every top-level network key and its `name` begins with `pg-stack-`.

- [x] 3. Author the sample Nest repository content under `compose/nest/`
  - [x] 3.1 Create `compose/nest/Eggs/sample-egg/config.fly`
    - Define a minimal `egg "sample-egg" { … }` block matching the design's sample, referencing `aws-sm://pg-stack/gitlab_token` and `aws-sm://pg-stack/webhook_secret` as secret URIs and `aws` / `us-east-1` / `serverless` as runner wiring.
    - _Requirements: 8.2_

  - [x] 3.2 Create `compose/nest/Jobs/rotate-secrets.fly`
    - Define a minimal `job "rotate-secrets" { … }` block with a cron `schedule`, AWS serverless runner, and a trivial heredoc `script`.
    - _Requirements: 8.2_

  - [x] 3.3 Create `compose/nest/UF/config.fly`
    - Define a minimal `uglyfox { pruning { … } apex_pool = { … } nadir_pool = { … } }` block per the design sample.
    - _Requirements: 8.2_

  - [x] 3.4 Create `compose/nest/MG/config.fly`
    - Define a minimal `mothergoose { sync_interval_seconds = 300 api_gateway = { path_prefix = "/api/v1" } }` block.
    - _Requirements: 8.2_

  - [x] 3.5 Write property test `test_nest_parseability.py::test_every_nest_directory_has_parseable_fly`
    - **Property 12: Every Nest directory contains a parseable `.fly` file**
    - **Validates: Requirement 8.2**
    - For each directory in `{Eggs, Jobs, UF, MG}` under `compose/nest/`, assert at least one `.fly` file exists and that `gosling parse <path> --type=<inferred>` exits with status 0 (skip the test with a clear marker if the `gosling` binary is not on PATH so local developers without the Go build can still run the rest of the suite).

- [x] 4. Build the `nest-git` container image and wire it into Compose
  - [x] 4.1 Create `compose/nest-git/Dockerfile`
    - Base on `alpine:3.20` (pinned), install `git` and `git-daemon` and `lighttpd` (or `git-http-backend` via busybox) to serve the repo.
    - At image build time, `COPY ../nest /srv/nest-src`, then run `git init --bare /srv/git/nest.git`, initialise a working tree under `/tmp/work`, copy `/srv/nest-src/*` in, `git add -A && git commit -m 'seed' && git push /srv/git/nest.git main`.
    - Expose port `80`; configure the HTTP server to route `/nest.git/*` to `git-http-backend` via CGI.
    - Drop to a non-root user.
    - _Requirements: 8.1, 8.2_

  - [x] 4.2 Add the `nest-git` service to `docker-compose.yml`
    - `container_name: pg-stack-nest-git`, `build: { context: ./nest-git, dockerfile: Dockerfile }`, image tag `pg-stack/nest-git:${NEST_GIT_IMAGE_TAG:-0.1.0}`.
    - No published ports (reachable only as `http://nest-git/nest.git` inside `pg-stack-net`).
    - Healthcheck `curl -fsS http://localhost/nest.git/info/refs?service=git-upload-pack` with `interval=5s`, `timeout=5s`, `start_period=30s`, `retries=6`.
    - _Requirements: 1.2, 8.1, 8.3, 13.1, 13.4_

- [x] 5. Implement the seed job (image + orchestrator + fixtures)
  - [x] 5.1 Create seed fixtures under `compose/seed/fixtures/`
    - `egg_configs.json` containing the single `sample-egg` row from the design, with `git_repo_url_secret = "aws-sm://pg-stack/nest_git_url"`.
    - `secrets.json` containing the three `aws-sm://` URIs from the design (nest git URL, GitLab token, webhook secret).
    - `buckets.json` → `["polar-gosling-artifacts"]`, `queues.json` → `["mothergoose", "uglyfox"]`.
    - `eventbridge_rules.json` containing the single `pg-stack-sync-git` rule with `rate(1 minute)` targeting `http://mothergoose-api:8000/internal/sync-git`.
    - _Requirements: 2.7, 3.6, 3.7, 3.8, 8.4, 9.2, 9.3_

  - [x] 5.2 Create `compose/seed/Dockerfile`
    - Base on `pg-stack/mothergoose:${MG_IMAGE_TAG:-dev}` (built earlier by Compose) to inherit `ydb`, `boto3`, and Python 3.13.
    - `COPY seed.py /app/seed.py` and `COPY fixtures /app/fixtures` and (optionally) `COPY artifacts /app/artifacts`.
    - `ENTRYPOINT ["python", "/app/seed.py"]`, run as the non-root `mothergoose` user inherited from the base image, add `LABEL pg-stack=true` for the `compose-clean` prune rule.
    - _Requirements: 9.1_

  - [x] 5.3 Implement `compose/seed/seed.py` — YDB schema creation step
    - Load env (`MOTHERGOOSE_YDB_ENDPOINT`, `MOTHERGOOSE_YDB_DATABASE`), connect via `ydb` SDK.
    - Create tables `runners`, `egg_configs`, `sync_history`, `deployment_plans`, `audit_logs`, `tofu_versions`, `gosling_version` using the existing MotherGoose schema module (imported from the installed package).
    - Treat `TableAlreadyExists` / 409 as success; wrap the step in a `retry(attempts=3, delay=2s)` helper.
    - Log `INFO "creating <table>"` and `INFO "exists, skipping"` per the design error matrix.
    - _Requirements: 2.7, 2.8, 9.2, 14.4, 14.5_

  - [x] 5.4 Implement `compose/seed/seed.py` — LocalStack seeding steps
    - S3 bucket creation from `fixtures/buckets.json` in region `us-east-1` against `AWS_ENDPOINT_URL`.
    - SQS queue creation from `fixtures/queues.json`.
    - Secrets Manager entry creation from `fixtures/secrets.json` plus any `aws-sm://` URI referenced by an `egg_configs` row but absent from `secrets.json` (auto-filled with a deterministic `"<uri>:dev-placeholder"` value).
    - Binary artifact upload: walk `/app/artifacts` (empty-dir-safe) and `put_object` each file to `s3://polar-gosling-artifacts/<relative-path>`.
    - Each step uses the same `retry(attempts=3, delay=2s)` helper; every failure after 3 attempts logs `ERROR` and marks the step failed without aborting remaining independent steps.
    - _Requirements: 3.6, 3.7, 3.8, 3.9, 9.2, 9.4, 14.5, 14.6_

  - [x] 5.5 Implement `compose/seed/seed.py` — YDB row seeding and exit contract
    - Upsert `egg_configs.json` rows into the `egg_configs` table with `synced_at = created_at = updated_at = now_utc_iso()`.
    - If `COMPOSE_PROFILES` (read from env) contains `with-triggers`, additionally create every EventBridge rule from `eventbridge_rules.json`; otherwise skip that step entirely.
    - After all steps, exit `0` iff no step failed; otherwise exit `1` with an accumulated summary log line.
    - Pydantic v2 models validate each fixture at startup before any side-effecting step runs.
    - _Requirements: 8.4, 9.2, 9.3, 9.4, 9.5, 14.4, 14.5, 14.6_

  - [x] 5.6 Add the `seed` service to `docker-compose.yml`
    - `container_name: pg-stack-seed`, `build: { context: ./seed, dockerfile: Dockerfile }`, image tag `pg-stack/seed:${MG_IMAGE_TAG:-dev}`.
    - `restart: no`, `profiles: [seed]`, `depends_on: { ydb: service_healthy, localstack: service_healthy, nest-git: service_healthy }`.
    - Env: full MG/AWS env block plus `SEED_DATA_DIR=/app/fixtures` and `SEED_PROFILE=${COMPOSE_PROFILES:-}`.
    - Mount fixtures read-only.
    - _Requirements: 1.7, 9.1, 9.6_

  - [x] 5.7 Write property test `test_seed_idempotency.py::test_seed_is_idempotent`
    - **Property 5: Seed operations are idempotent**
    - **Validates: Requirements 3.6, 3.7, 3.8, 9.5, 14.4, 14.5**
    - Using `testcontainers` to launch ephemeral LocalStack + YDB containers, parameterise with `hypothesis` over the set of pre-existing resources (subsets of the fixture set) and assert `seed(state)` yields the full inventory and exits `0`, and `seed(seed(state)) == seed(state)`. Use `max_examples=20` per the design.

  - [x] 5.8 Write property test `test_seed_idempotency.py::test_eventbridge_profile_gated`
    - **Property 13: EventBridge rules created iff `with-triggers` profile is active**
    - **Validates: Requirement 9.3**
    - Hypothesis strategy generates arbitrary comma-separated `COMPOSE_PROFILES` values; assert the EventBridge rule count equals `len(eventbridge_rules.json)` iff `with-triggers` is a member, else 0.

  - [x] 5.9 Write unit tests `test_seed_errors.py` for the retry/error paths
    - Use `unittest.mock` to simulate transient and permanent failures of each AWS/YDB client call.
    - Verify: retry count is 3 with 2s spacing; transient failures recover; permanent failures log `ERROR` with resource name + cause; exit code is non-zero on any permanent failure; independent steps continue to run after a failure.
    - _Requirements: 2.8, 3.9, 9.4, 14.5, 14.6_

- [x] 6. Implement the trigger emulator (script + Dockerfile + Compose wiring)
  - [x] 6.1 Create `compose/trigger/Dockerfile`
    - Base on `python:3.13-slim-trixie` (pinned), install only `httpx` via `pip` in a single `RUN` layer inside a venv.
    - `COPY trigger.py /app/trigger.py`; non-root user; `ENTRYPOINT ["python", "/app/trigger.py"]`; `LABEL pg-stack=true`.
    - _Requirements: 1.2, 1.6_

  - [x] 6.2 Implement `compose/trigger/trigger.py::parse_interval`
    - Pure function matching the design sketch: accepts `str | None`, returns an `int` in `[5, 3600]`, defaults to `60` on any invalid input, and returns `n` unchanged for any valid in-range integer string.
    - Emit a structured `ERROR` log line on invalid input before falling back.
    - _Requirements: 5.2, 5.3_

  - [x] 6.3 Implement `compose/trigger/trigger.py::main` loop
    - Read `INTERNAL_SYNC_TOKEN` (exit `1` with error log if empty/unset).
    - Resolve interval via `parse_interval(os.environ.get("TRIGGER_SYNC_INTERVAL_SECONDS"))`.
    - Every `interval` seconds POST to `${MOTHERGOOSE_API_URL}/internal/sync-git` with header `X-Internal-Token`, `httpx.Client(timeout=10)`.
    - On any `httpx.RequestError` or non-2xx response, log one `WARN` line with reason and duration, do not crash, continue.
    - On every outcome emit one structured line containing ISO-8601 UTC timestamp, status token, and `duration_ms=<int>`.
    - _Requirements: 5.1, 5.4, 5.5, 5.6, 5.8_

  - [x] 6.4 Add the `trigger-emulator` service to `docker-compose.yml`
    - `container_name: pg-stack-trigger-emulator`, `build: { context: ./trigger, dockerfile: Dockerfile }`, image tag `pg-stack/trigger-emulator:${MG_IMAGE_TAG:-dev}`.
    - Env: `TRIGGER_SYNC_INTERVAL_SECONDS=${TRIGGER_SYNC_INTERVAL_SECONDS:-60}`, `INTERNAL_SYNC_TOKEN=${INTERNAL_SYNC_TOKEN:?set INTERNAL_SYNC_TOKEN in .env}`, `MOTHERGOOSE_API_URL=http://mothergoose-api:8000`.
    - `profiles: [with-triggers]`, `depends_on: { mothergoose-api: service_healthy, seed: service_completed_successfully }`.
    - Healthcheck `pgrep -af trigger.py`.
    - _Requirements: 1.7, 5.7, 9.6, 13.1, 13.4_

  - [x] 6.5 Write property test `test_trigger_emulator.py::test_parse_interval_total_and_bounded`
    - **Property 7: `parse_interval` is total, bounded, and identity-on-valid**
    - **Validates: Requirements 5.2, 5.3**
    - Hypothesis strategy generates arbitrary text (including empty, `None`, non-integer, negative, fractional, out-of-range, and in-range integer strings); assert the result is in `[5, 3600]`, equals `n` for in-range integers, and defaults to `60` otherwise.

  - [x] 6.6 Write property test `test_trigger_emulator.py::test_loop_tolerates_all_failure_modes`
    - **Property 8: Trigger loop tolerates all failure modes**
    - **Validates: Requirement 5.6**
    - Use `respx` (or a hand-rolled `httpx.MockTransport`) driven by a hypothesis strategy that yields per-iteration outcomes (HTTP status code or `RequestError` subclass); drive the loop for `N` iterations and assert the loop issues exactly `N` POSTs, does not raise, emits one `WARNING` per failure, and one `INFO` per 2xx.

  - [x] 6.7 Write property test `test_trigger_emulator.py::test_loop_emits_structured_log_line`
    - **Property 9: Each trigger invocation emits one well-formed structured log line**
    - **Validates: Requirement 5.8**
    - Capture logs via `caplog`; assert each iteration emits exactly one line containing an ISO-8601 UTC timestamp, a status token, and a `duration_ms=<non-negative-int>` token.

  - [x] 6.8 Write unit tests for the startup `INTERNAL_SYNC_TOKEN` gate
    - Verify that a missing, empty, or whitespace-only token causes the process to exit `1` with an error log, without performing any POST.
    - _Requirements: 5.4, 5.5_

- [x] 7. Wire the MotherGoose and UglyFox services into `docker-compose.yml`
  - [x] 7.1 Add `mothergoose-api` service
    - `container_name: pg-stack-mothergoose-api`, `build: { context: ../, dockerfile: Dockerfile.mtg }`, image tag `pg-stack/mothergoose:${MG_IMAGE_TAG:-dev}`.
    - `command: uvicorn app.main:app --host 0.0.0.0 --port 8000`, ports `127.0.0.1:${MOTHERGOOSE_API_HOST_PORT:-8000}:8000`.
    - Env block from the design — MG YDB trio, `MOTHERGOOSE_BROKER_URL` / `MOTHERGOOSE_RESULT_BACKEND_URL` pointing at `redis://celery-broker:6379/{0,1}`, `MOTHERGOOSE_NEST_REPO_URL=http://nest-git/nest.git`, `MOTHERGOOSE_INTERNAL_SYNC_TOKEN=${INTERNAL_SYNC_TOKEN:?…}`, four `AWS_*` vars.
    - Healthcheck `curl -fsS http://localhost:8000/health` (`interval=10s`, `timeout=5s`, `start_period=60s`, `retries=5`), `restart: no`.
    - `depends_on`: `ydb`, `localstack`, `celery-broker`, `nest-git` all `service_healthy`; `seed` `service_completed_successfully`.
    - _Requirements: 1.2, 1.6, 3.10, 4.4, 4.6, 6.1, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 9.6_

  - [x] 7.2 Add `mothergoose-worker` service
    - Same image and env as `mothergoose-api`; command `celery -A app.celery_worker worker --loglevel=info -Q mothergoose`; no published ports.
    - Healthcheck `celery -A app.celery_worker inspect ping -d celery@$$(hostname)` (`interval=15s`, `timeout=10s`, `start_period=45s`, `retries=4`).
    - Same `depends_on` set as the API.
    - _Requirements: 1.2, 1.6, 3.10, 4.4, 4.6, 6.2, 6.3, 6.4, 9.6, 13.4_

  - [x] 7.3 Add `uglyfox-worker` service
    - `container_name: pg-stack-uglyfox-worker`, `build: { context: ../, dockerfile: Dockerfile.uf }`, image tag `pg-stack/uglyfox:${UF_IMAGE_TAG:-dev}`.
    - `command: celery -A app.celery_worker worker --loglevel=info -Q uglyfox`; no ports.
    - Env: exactly `UGLYFOX_DATABASE_TYPE=ydb`, `UGLYFOX_YDB_ENDPOINT=grpc://ydb:2136`, `UGLYFOX_YDB_DATABASE=/local`, `UGLYFOX_BROKER_URL`, `UGLYFOX_RESULT_BACKEND_URL`, and the four `AWS_*` vars — no other `UGLYFOX_*` keys.
    - Healthcheck `pgrep -af 'celery.*worker'` (`interval=15s`, `timeout=10s`, `start_period=45s`, `retries=4`).
    - `depends_on`: `ydb`, `localstack`, `celery-broker` `service_healthy`; `seed` `service_completed_successfully`.
    - _Requirements: 1.2, 1.6, 3.10, 4.5, 4.6, 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 9.6, 13.4_

  - [x] 7.4 Write property test `test_compose_properties.py::test_depends_on_graph`
    - **Property 4: `depends_on` graph matches the expected dependency map**
    - **Validates: Requirements 2.5, 2.6, 4.6, 7.5, 7.6, 9.6**
    - Parse the resolved Compose file and assert every expected `(dependent, dependency, condition)` edge is present; additionally assert `uglyfox-worker.depends_on` equals exactly the four-edge set in the design.

  - [x] 7.5 Write property test `test_compose_properties.py::test_profile_assignment`
    - **Property 3: Non-happy-path services use only allowed Compose profiles**
    - **Validates: Requirement 1.7**
    - Assert happy-path services have no/empty `profiles`; `seed` is in `{seed}`; `trigger-emulator` is in `{triggers, with-triggers}`; no service uses a profile outside `{seed, triggers, with-triggers, debug}`.

  - [x] 7.6 Write property test `test_compose_properties.py::test_service_env_vars`
    - **Property 6: App containers receive the expected environment values**
    - **Validates: Requirements 3.10, 4.4, 4.5, 6.3, 6.4, 7.2**
    - For each `(container, variable, expected_value)` triple from the design's cross-service env map, assert the resolved env contains the variable with the expected value.

  - [x] 7.7 Write property test `test_compose_properties.py::test_uglyfox_env_allowlist`
    - **Property 11: UglyFox container env vars are a subset of the allow-list**
    - **Validates: Requirement 7.2**
    - Assert every key starting with `UGLYFOX_` on `uglyfox-worker` is in `{UGLYFOX_DATABASE_TYPE, UGLYFOX_YDB_ENDPOINT, UGLYFOX_YDB_DATABASE, UGLYFOX_BROKER_URL, UGLYFOX_RESULT_BACKEND_URL}`.

  - [x] 7.8 Write property test `test_compose_properties.py::test_internal_sync_token_shared`
    - **Property 10: Internal sync token is shared and well-formed**
    - **Validates: Requirement 6.5**
    - Hypothesis generates arbitrary 16–128 char tokens; assert `mothergoose-api.MOTHERGOOSE_INTERNAL_SYNC_TOKEN` equals `trigger-emulator.INTERNAL_SYNC_TOKEN` byte-for-byte and satisfies the length constraint.

- [x] 8. Checkpoint — Ensure all compose and seed artifacts pass local validation
  - Run `docker compose -f compose/docker-compose.yml config -q` locally and in CI.
  - Run every `test_compose_properties.py` and `test_seed_*.py` test and ensure they pass.
  - Ask the user if questions arise.

- [x] 9. Implement static compose and env validation scripts
  - [x] 9.1 Create `compose/scripts/check_compose.py`
    - Parse `docker-compose.yml` via `docker compose config --format json` (subprocess), then validate:
      - every `image:` contains `:`, is non-empty, does not end with `:latest`;
      - every env-driven tag matches `^\d{1,4}\.\d{1,4}(\.\d{1,4})?$`;
      - every `container_name`, every top-level volume key + `name`, every network key + `name` begins with `pg-stack-`.
    - Exit `0` on all green; exit `1` with a line listing every violation.
    - _Requirements: 1.5, 13.1, 13.2, 13.3_

  - [x] 9.2 Create `compose/scripts/check_env.py`
    - Parse `.env.example` for `KEY=value` lines (tracking which keys are marked `REQUIRED` vs have a default) and the preceding comment lines.
    - Parse `docker-compose.yml` (raw text) for `${VAR}`, `${VAR:-default}`, and `${VAR:?message}` references.
    - Assert: `set(declared) == set(referenced)`; every REQUIRED var uses `:?error` syntax; every var with a default uses `:-default` syntax with a matching default; every non-blank non-comment `.env.example` line matches `^[A-Z_][A-Z0-9_]*=.*$`; every declaration is preceded by either another declaration or a `#`-comment describing it.
    - Exit `0` / `1` with actionable diffs.
    - _Requirements: 11.2, 11.3, 11.4, 11.5, 11.6, 11.7_

  - [x] 9.3 Write property test `test_env_example_contract.py::test_env_example_shape`
    - **Property 14: `.env.example` has canonical variable-declaration shape**
    - **Validates: Requirements 11.2, 11.3, 11.4**
    - Hypothesis-generated perturbations of `.env.example` (add/remove comment lines, inject blank lines, swap formats) confirm `check_env.py` accepts well-formed inputs and rejects malformed ones.

  - [x] 9.4 Write property test `test_env_example_contract.py::test_compose_reference_syntax_matches`
    - **Property 15: Compose reference syntax matches the `.env.example` REQUIRED/default declaration**
    - **Validates: Requirements 11.5, 11.6**
    - For each declared variable assert the `${V…}` references use the correct `:?` vs `:-` syntax with the right default.

  - [x] 9.5 Write property test `test_env_example_contract.py::test_declared_equals_referenced`
    - **Property 16: Declared and referenced env-var sets are equal**
    - **Validates: Requirement 11.7**
    - `set(declared_vars) == set(referenced_vars)`.

- [x] 10. Implement the end-to-end smoke test script
  - [x] 10.1 Create `compose/scripts/smoke_test.py` — driver and step framework
    - Define `SmokeStep` dataclass (`id`, `description`, `timeout_s`, `poll_interval_s`) and `StepResult` (`ok`, `elapsed_ms`, `detail`).
    - Implement `run_step(step, fn)` with a per-step timeout (asyncio or `threading.Timer`), catching `TimeoutError` and `AssertionError` into failing `StepResult`s with the step id and reason.
    - Parse env (`MOTHERGOOSE_API_URL`, `INTERNAL_SYNC_TOKEN`, YDB endpoint, `SMOKE_TEST_VERBOSE`).
    - _Requirements: 12.1, 12.3, 12.4_

  - [x] 10.2 Implement steps (a)–(f) in `smoke_test.py`
    - Step (a): `GET /health` → 200 within 10 s.
    - Step (b): `POST /internal/sync-git` with `X-Internal-Token` → 202 within 10 s.
    - Step (c): poll `sync_history` in YDB every 2 s for up to 60 s until a row with `status == SUCCESS` appears.
    - Step (d): query `egg_configs` in YDB; assert ≥ 1 row.
    - Step (e): send a mock GitLab webhook (POST `/webhooks/gitlab` with `X-Gitlab-Token` and a realistic push-event body) → 202 within 10 s.
    - Step (f): poll `audit_logs` in YDB every 2 s for up to 60 s until ≥ 1 row exists.
    - On the first failing step, emit a single `ERROR` line to stderr naming the step id and observed failure, and exit `1`.
    - When all steps pass, exit `0`; when `SMOKE_TEST_VERBOSE=1`, also emit one `step=<id> … duration_ms=<int>` line per executed step to stdout.
    - _Requirements: 12.2, 12.3, 12.4, 12.5, 12.6_

  - [x] 10.3 Write unit tests `test_smoke_driver.py` for the driver
    - Mock MG HTTP endpoints and YDB polling; simulate timeout, assertion failure, and happy paths; assert correct exit codes, log lines, and step identifiers.
    - _Requirements: 12.3, 12.4, 12.5_

  - [x] 10.4 Write property test `test_smoke_driver.py::test_verbose_output_shape`
    - **Property 17: Verbose smoke test emits one well-formed line per executed step**
    - **Validates: Requirement 12.6**
    - Hypothesis over arbitrary subsets `S ⊆ {a…f}` of steps that execute to completion; capture stdout; assert exactly `|S|` lines matching `^step=[a-f]\s.+\sduration_ms=\d+$`, each step id appearing exactly once.

- [x] 11. Extend `dev-new-features/Makefile` with the `compose-*` targets
  - [x] 11.1 Add the shared `_preflight` define plus `COMPOSE` and `PROFILES_UP` variables
    - `_preflight` guards `docker` and `docker compose version`; exits `1` with a naming error within 5 s on missing deps.
    - `COMPOSE := docker compose -f compose/docker-compose.yml`, `PROFILES_UP := --profile seed`.
    - _Requirements: 10.9_

  - [x] 11.2 Add `compose-up`, `compose-down`, `compose-reset`, `compose-logs` targets
    - `compose-up`: `$(COMPOSE) $(PROFILES_UP) up -d --wait --wait-timeout 180` with fallback error message naming `mothergoose-api` last health.
    - `compose-down`: `$(COMPOSE) down` (keep volumes).
    - `compose-reset`: `$(COMPOSE) down -v` → `$(MAKE) compose-up`.
    - `compose-logs`: `$(COMPOSE) logs -f`.
    - Each target starts with `$(_preflight)`.
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 14.1, 14.2, 14.3_

  - [x] 11.3 Add `compose-smoke`, `compose-clean`, `compose-config`, `compose-check` targets
    - `compose-smoke`: preflight → assert stack is running via `$(COMPOSE) ps --services --filter status=running` (error-log naming the condition and exit `1` on empty) → run `uv run python compose/scripts/smoke_test.py` from `dev-new-features/`.
    - `compose-clean`: `$(COMPOSE) down -v` → `docker image prune --filter "label=pg-stack=true" -f`.
    - `compose-config`: `$(COMPOSE) config -q`.
    - `compose-check`: run `check_compose.py`, then `check_env.py`, then `$(COMPOSE) config -q`.
    - _Requirements: 10.6, 10.7, 10.8, 14.7, 14.8_

  - [x] 11.4 Write unit tests `test_makefile_preflight.py`
    - Drive the `_preflight` shell via `subprocess` with `PATH` manipulated to hide `docker` and/or `docker compose`; assert exit code is non-zero within 5 s and stderr names the missing tool.
    - _Requirements: 10.9_

- [x] 12. Author the CI workflow `dev-new-features/.github/workflows/compose-smoke.yml`
  - [x] 12.1 Replace the placeholder from task 1.3 with the full workflow
    - Trigger: `pull_request` with `types: [opened, synchronize, reopened]` on branches `[dev-new-features]`.
    - Job `smoke` on `ubuntu-latest` with `timeout-minutes: 15`, `defaults.run.working-directory: dev-new-features`.
    - Steps: checkout, `docker/setup-buildx-action@v3`, `astral-sh/setup-uv@v3` (Python 3.13), render CI `.env` (copy example, sed-inject pinned tags and a 16+ char `INTERNAL_SYNC_TOKEN`), `make compose-check`, `make compose-up` (step id `up`), `make compose-smoke` (step id `smoke`), dump per-service logs on failure, upload `compose-logs` artifact with `retention-days: 7` on failure, run `make compose-down` with `if: always()`.
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_

  - [x] 12.2 Write example-based test `test_ci_workflow_shape.py`
    - Parse `compose-smoke.yml`; assert trigger events, branch filter, step order, artifact name and retention, and `timeout-minutes` match the requirements.
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_

- [x] 13. Author `compose/README.md` and run final validation
  - [x] 13.1 Write `compose/README.md`
    - Section (a) purpose of Cloud_Stack.
    - Section (b) enumerated list of every service with one-line descriptions.
    - Section (c) enumerated list of every Makefile target with its effect.
    - Section (d) enumerated list of every env var with name, purpose, default, and REQUIRED flag.
    - Section (e) the exact `make compose-smoke` command and expected output.
    - Section (f) troubleshooting with at least three distinct failure scenarios (missing `INTERNAL_SYNC_TOKEN`, port `8000`/`2136`/`4566` already bound, `docker compose` plugin missing) and remediation steps.
    - Worked end-to-end example from `make compose-up` through `make compose-smoke` with expected output per command.
    - State minimum Docker Engine and `docker compose` plugin versions (concrete `MAJOR.MINOR.PATCH`) and the exact commands to verify them; add a Windows-specific subsection documenting Docker Desktop minimum, WSL2 kernel minimum, and required file-sharing settings.
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

  - [x]* 13.2 Write example-based test `test_readme_shape.py`
    - Parse `compose/README.md`; assert the six required sections are present as headings, the Windows subsection exists, the worked example includes both `make compose-up` and `make compose-smoke` with expected-output blocks, and the troubleshooting section lists ≥ 3 distinct scenarios.
    - _Requirements: 15.1, 15.2, 15.4, 15.5_

  - [x] 13.3 Final checkpoint — Ensure all tests pass
    - Run `uv run pytest -v compose/tests` from `dev-new-features/` and ensure every non-optional test passes; invoke `make compose-check` and confirm zero violations; run `make compose-up` + `make compose-smoke` + `make compose-down` end-to-end once against a local Docker engine.
    - Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional (tests) and can be skipped for a faster MVP, but the dependency graph still schedules them correctly.
- Every non-optional task references its governing requirements, and every property-test sub-task additionally references the property number from the design document.
- Core implementation tasks (scaffolding, Compose services, seed/trigger/smoke scripts, Makefile, CI workflow, README) are never marked optional.
- Checkpoints (tasks 8 and 13.3) provide natural validation breaks.
- Property-based tests use `hypothesis` and live under `compose/tests/`; they can be executed without the stack running (the seed idempotency test uses `testcontainers` to spin ephemeral LocalStack + YDB containers just for itself).
- All new files land under `Polar-Gosling-Backends/dev-new-features/compose/`, `dev-new-features/.github/workflows/`, and `dev-new-features/Makefile` (extended, not replaced), per the workspace rule that "all new code must be located under `dev-new-features/`".

## Task Dependency Graph

Tasks that write to the same file (`docker-compose.yml`, `Makefile`,
`seed.py`, `test_compose_properties.py`, `test_env_example_contract.py`,
`test_trigger_emulator.py`, `test_seed_idempotency.py`, `test_smoke_driver.py`)
are placed in separate waves to avoid write conflicts. Independent-file tasks
are parallelised within each wave.

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "1.3", "3.1", "3.2", "3.3", "3.4", "5.1", "6.1", "6.2"] },
    { "id": 2, "tasks": ["2.1", "4.1", "5.2", "6.3", "10.1", "11.1"] },
    { "id": 3, "tasks": ["2.2", "5.3", "3.5", "11.2", "6.5"] },
    { "id": 4, "tasks": ["2.3", "5.4", "11.3", "9.1", "9.2", "6.6", "10.2", "11.4"] },
    { "id": 5, "tasks": ["2.4", "5.5", "6.7", "10.3", "9.3"] },
    { "id": 6, "tasks": ["4.2", "5.9", "10.4", "9.4", "6.8"] },
    { "id": 7, "tasks": ["5.6", "5.7", "9.5"] },
    { "id": 8, "tasks": ["5.8", "7.1"] },
    { "id": 9, "tasks": ["7.3"] },
    { "id": 10, "tasks": ["7.2"] },
    { "id": 11, "tasks": ["6.4"] },
    { "id": 12, "tasks": ["2.5", "12.1", "13.1"] },
    { "id": 13, "tasks": ["2.6"] },
    { "id": 14, "tasks": ["2.7"] },
    { "id": 15, "tasks": ["2.8"] },
    { "id": 16, "tasks": ["7.4"] },
    { "id": 17, "tasks": ["7.5"] },
    { "id": 18, "tasks": ["7.6"] },
    { "id": 19, "tasks": ["7.7"] },
    { "id": 20, "tasks": ["7.8"] },
    { "id": 21, "tasks": ["12.2", "13.2"] }
  ]
}
```

## Workflow Completion

This workflow has produced the three planning artefacts — `requirements.md`,
`design.md`, and this `tasks.md` — for the `docker-compose-cloud-stack-testing`
feature. No implementation has been performed.

To begin executing the plan, open
`.kiro/specs/docker-compose-cloud-stack-testing/tasks.md` and click "Start
task" next to the first incomplete item. Tasks can be executed in dependency
order (see the graph above) or in listed order.
