# Implementation Plan: OpenTofu Integration Testing

## Overview

This implementation plan breaks down the OpenTofu integration testing feature into discrete, actionable tasks. Each task builds on previous work to create a comprehensive integration test suite.

**KEY PRINCIPLES:**
- **REUSE EXISTING FIXTURES**: Session-scoped `ydb_container`, `localstack_container`, `aws_credentials` from `tests/conftest.py`
- **USE EXISTING CLASSES**: `OpenTofuUpdateGithub`, `OpenTofuConfiguration`, and schema classes from MotherGoose backend
- **EXISTING DIRECTORY**: Tests go in `tests/integration/`, not a new subdirectory
- **MINIMAL NEW FIXTURES**: Only add 4 new fixtures to `tests/conftest.py`

## Tasks

- [ ] 1. Set up integration test infrastructure
  - Configure pytest markers for integration tests in `pyproject.toml`
  - Add required dependencies to `pyproject.toml` (python-hcl2, etc.)
  - Tests will be added to existing `tests/integration/` directory
  - _Requirements: 10.1, 10.2_

- [ ] 2. Add new integration test fixtures to tests/conftest.py
  - **REUSE EXISTING FIXTURES**: `ydb_container`, `localstack_container`, `aws_credentials` (already session-scoped)
  
  - [ ] 2.1 Implement OpenTofu updater fixture
    - Create session-scoped fixture using `OpenTofuUpdateGithub`
    - Pass `test_ydb_schema` to constructor
    - Call `start_update(rb=3)` to download and verify binary
    - Binary automatically cached in `/mnt/tofu_binary/{version}/`
    - Return updater instance with `c_version` property
    - _Requirements: 1.1, 1.2, 1.3, 1.5_

  - [ ] 2.2 Implement S3 backend configuration fixture
    - Function-scoped fixture depending on `aws_credentials` and `sqs_client`
    - Create S3 bucket in LocalStack using boto3
    - Create DynamoDB table for state locking using boto3
    - Return `TofuBackendS3Options` instance with LocalStack endpoint
    - Implement cleanup logic (delete bucket and table)
    - _Requirements: 3.2, 3.3_

  - [ ] 2.3 Implement OpenTofu workspace fixture
    - Function-scoped fixture
    - Create temporary directory for workspace using `tempfile.mkdtemp()`
    - Yield workspace path
    - Cleanup: Remove directory with `shutil.rmtree()`
    - _Requirements: 4.3_

  - [ ] 2.4 Implement OpenTofu configuration service fixture
    - Function-scoped fixture depending on `opentofu_updater`, `s3_backend_config`, `opentofu_workspace`
    - Create `TofuSetting` with test providers and S3 backend config
    - Create `OpenTofuConfiguration` instance with updater and settings
    - Call `setup_topfu_configuration()` to generate templates
    - Return configured service ready for testing
    - _Requirements: 8.1_

- [ ] 3. Create binary download and validation tests (test_opentofu_binary_integration.py)
  - [ ] 3.1 Test binary download using OpenTofuUpdateGithub
    - Use `opentofu_updater` fixture (already downloaded)
    - Verify `c_version` property is tuple of (version_id, version, sha256_hash)
    - Verify version_id is not "dummy_id"
    - Verify version string matches expected format (e.g., "1.6.0")
    - _Requirements: 1.1_

  - [ ] 3.2 Test binary checksum verification
    - Access `opentofu_updater.c_version[2]` for SHA256 hash
    - Verify hash is not "dummy_hash"
    - Verify hash matches expected format (64 hex characters)
    - Verify `OpenTofuDownloadGithub.get_packages_sha256_hash` contains version
    - _Requirements: 1.2_

  - [ ] 3.3 Test binary version check
    - Get binary path from updater: `/mnt/tofu_binary/{version}/tofu`
    - Execute binary with `--version` flag using subprocess
    - Parse stdout and verify version string matches `c_version[1]`
    - _Requirements: 1.4_

  - [ ] 3.4 Test binary caching
    - Verify binary exists at `/mnt/tofu_binary/{version}/tofu`
    - Verify file size > 0
    - Create second updater instance with same schema
    - Verify it reuses cached binary (no download)
    - _Requirements: 1.5_

  - [ ] 3.5 Test binary executable
    - Get binary path from updater
    - Verify file has executable permissions (os.access with os.X_OK)
    - Execute binary with `--help` to verify it runs
    - _Requirements: 1.3_

- [ ] 4. Create template generation tests (test_opentofu_templates_integration.py)
  - [ ] 4.1 Test versions.tf generation (tofu_versions_tf.j2)
    - Use `opentofu_config_service` fixture (already called setup_topfu_configuration)
    - Read generated `versions.tf` from workspace
    - Parse HCL using `python-hcl2` library
    - Verify terraform block exists with required_version
    - Verify backend "s3" block with LocalStack endpoint
    - Verify required_providers block includes: local, tls, random, and test providers
    - Verify all provider sources and versions are correct
    - _Requirements: 2.1, 2.2, 2.3_

  - [ ] 4.2 Test health checks generation (tofu_checks_tf.j2)
    - Configure `TofuSetting` with health_checks list
    - Create service and call `setup_topfu_configuration()`
    - Read generated `checks.tf` from workspace
    - Parse HCL and verify check blocks exist
    - Verify check URLs match configuration
    - Verify check names and conditions are correct
    - _Requirements: 2.4_

  - [ ] 4.3 Test cloud-init script generation (cloud-init-runner.tpl.j2)
    - Use `opentofu_config_service.generate_cloud_init_script()`
    - Pass test parameters: runner_id, egg_name, mothergoose_api_url, admin_ssh_key, gosling_binary_url
    - Verify returned script is valid YAML
    - Parse YAML and verify all required sections exist
    - Verify runner_id, egg_name, and URLs are correctly embedded
    - Verify SSH key is properly formatted
    - _Requirements: 2.1_

  - [ ] 4.4 Test providers.tf generation (tofu_providers_tf.j2)
    - Configure `TofuSetting` with provider settings (Yandex Cloud and AWS)
    - Create service and call extended template rendering method
    - Read generated `providers.tf` from workspace
    - Parse HCL and verify provider blocks exist
    - Verify provider configurations include correct settings
    - Verify both yandex and aws providers are configured
    - _Requirements: 2.1, 2.3_

  - [ ] 4.5 Test resources.tf generation (tofu_resources_tf.j2)
    - Configure `TofuSetting` with worker_module_source and worker_instances
    - Create service and call extended template rendering method
    - Read generated `resources.tf` from workspace
    - Parse HCL and verify resource blocks exist:
      - tls_private_key resource
      - local_file resource for SSH key
      - worker module block with for_each
    - Verify module source is correctly set (git or registry)
    - Verify cloud-init reference for VM chassis
    - _Requirements: 2.1_

  - [ ] 4.6 Test resources.tf with Rift module (tofu_resources_tf.j2)
    - Configure `TofuSetting` with rift_required=true and rift_instances
    - Create service and call extended template rendering method
    - Read generated `resources.tf` from workspace
    - Parse HCL and verify rift module block exists
    - Verify rift module has for_each over rift_instances
    - Verify rift module source is correctly set
    - _Requirements: 2.1_

  - [ ] 4.7 Test variables.tf generation (tofu_variables_tf.j2)
    - Create service and call extended template rendering method
    - Read generated `variables.tf` from workspace
    - Parse HCL and verify variable blocks exist:
      - tofu_worker_instances variable
      - tofu_rift_instances variable (if rift_required)
    - Verify variable types are map(any)
    - Verify default values are empty maps
    - _Requirements: 2.1_

  - [ ] 4.8 Test .terraformrc generation (tofu_rc.j2)
    - Create service and call extended template rendering method
    - Read generated `.terraformrc` or `.tofurc` from workspace
    - Parse configuration file
    - Verify provider_installation block exists
    - Verify plugin_cache_dir is configured
    - _Requirements: 2.1_

  - [ ] 4.9 Test template rendering with optional parameters
    - Test versions.tf with optional profile, endpoint, role_arn
    - Test cloud-init with optional s3_cache_bucket, custom_commands, environment_vars
    - Test resources.tf with and without rift_required
    - Verify optional parameters are included when provided
    - Verify optional parameters are omitted when not provided
    - _Requirements: 2.1, 2.2_

  - [ ] 4.10 Test HCL syntax validation for all templates
    - Use `opentofu_config_service` fixture
    - Get binary path from service.updater
    - Run `tofu fmt -check` on workspace directory
    - Verify exit code 0 (valid syntax for all generated files)
    - _Requirements: 2.5_

  - [ ] 4.11 Test HCL semantic validation for all templates
    - Use `opentofu_config_service` fixture
    - Run `tofu validate` on workspace directory
    - Verify exit code 0 (valid configuration for all generated files)
    - Parse stdout to verify "Success!" message
    - _Requirements: 2.6_

- [ ] 5. Create S3 backend integration tests (test_opentofu_s3_backend_integration.py)
  - [ ] 5.1 Test S3 bucket creation
    - Use `s3_backend_config` fixture
    - Use `sqs_client` fixture to access LocalStack S3
    - Verify bucket exists using `head_bucket()`
    - Verify bucket is accessible
    - _Requirements: 3.2_

  - [ ] 5.2 Test DynamoDB table creation
    - Use `s3_backend_config` fixture
    - Create DynamoDB client with `aws_credentials`
    - Verify table exists using `describe_table()`
    - Verify table has correct key schema (LockID)
    - _Requirements: 3.3_

  - [ ] 5.3 Test backend initialization
    - Use `opentofu_config_service` fixture
    - Run `tofu init` using service.tofu or subprocess
    - Verify exit code 0
    - Verify `.terraform/` directory created
    - Verify backend configuration stored in `.terraform/terraform.tfstate`
    - _Requirements: 3.4_

  - [ ] 5.4 Test state storage
    - After successful init, create simple null_resource
    - Run `tofu apply -auto-approve`
    - Use boto3 to verify state file exists in S3 bucket
    - Download state file and verify it's valid JSON
    - Verify state contains null_resource
    - _Requirements: 3.5_

  - [ ] 5.5 Test state locking
    - Create two OpenTofu processes
    - Start first process with long-running operation
    - Start second process immediately
    - Verify second process waits for lock
    - Check DynamoDB for lock entry
    - Verify lock is released after first operation completes
    - _Requirements: 3.3_

- [ ] 6. Create OpenTofu initialization tests (test_opentofu_init_integration.py)
  - [ ] 6.1 Test provider download
    - Use `opentofu_config_service` fixture
    - Run `tofu init` using subprocess with binary from updater
    - Verify exit code 0
    - Verify providers are downloaded to `.terraform/providers/`
    - Check for provider binaries (e.g., `terraform-provider-null_*`)
    - _Requirements: 4.1_

  - [ ] 6.2 Test .terraform directory creation
    - After init, verify `.terraform/` directory exists
    - Verify subdirectories: `providers/`, `modules/` (if applicable)
    - Verify directory structure matches expected layout
    - _Requirements: 4.3_

  - [ ] 6.3 Test lock file generation
    - After init, verify `.terraform.lock.hcl` exists
    - Read and parse lock file using HCL parser
    - Verify provider blocks exist with versions
    - Verify checksums are present for each provider
    - _Requirements: 4.4_

  - [ ] 6.4 Test provider verification
    - Parse `.terraform.lock.hcl` to get installed providers
    - Verify all providers from `TofuSetting.providers` are installed
    - Verify provider versions match configuration
    - Verify provider binaries are executable
    - _Requirements: 4.5_

- [ ] 7. Create deployment plan generation tests (test_opentofu_plan_integration.py)
  - [ ] 7.1 Test plan generation with no changes
    - Use `opentofu_config_service` fixture
    - Create simple null_resource configuration in workspace
    - Run `tofu apply -auto-approve` to create initial state
    - Run `tofu plan -detailed-exitcode` again
    - Verify exit code 0 (no changes)
    - _Requirements: 5.2_

  - [ ] 7.2 Test plan generation with changes
    - Use `opentofu_config_service` fixture
    - Create configuration with null_resource
    - Run `tofu plan -detailed-exitcode`
    - Verify exit code 2 (changes present)
    - Parse plan output to verify expected resource changes
    - _Requirements: 5.1, 5.2, 5.4_

  - [ ] 7.3 Test plan binary format
    - Use `opentofu_config_service.generate_deployment_plan()`
    - Verify method returns tuple (plan_binary, is_valid)
    - Verify plan_binary is bytes type
    - Verify len(plan_binary) > 0
    - Verify is_valid is True
    - _Requirements: 5.3_

  - [ ] 7.4 Test plan readability
    - Generate plan using `generate_deployment_plan()`
    - Write plan_binary to temporary file
    - Run `tofu show <plan_file>` to read plan
    - Verify exit code 0
    - Verify output contains resource information
    - _Requirements: 5.5_

- [ ] 8. Create YDB plan storage tests (test_opentofu_ydb_storage_integration.py)
  - [ ] 8.1 Test plan storage
    - Use `opentofu_config_service` and `test_ydb_schema` fixtures
    - Generate deployment plan using `generate_deployment_plan()`
    - Create YDB table for deployment plans (if not exists)
    - Store plan_binary in YDB using AsyncYDBOperations
    - Verify upsert operation succeeds
    - _Requirements: 6.3_

  - [ ] 8.2 Test plan metadata storage
    - Generate plan with egg_name, config_hash, git_commit
    - Store in YDB with metadata fields
    - Query YDB to verify all fields stored correctly
    - Verify timestamp is recorded
    - _Requirements: 6.4_

  - [ ] 8.3 Test plan retrieval
    - Store plan in YDB with specific egg_name and config_hash
    - Retrieve plan using select query with those keys
    - Verify retrieved data is not None
    - Verify egg_name and config_hash match
    - _Requirements: 6.5_

  - [ ] 8.4 Test binary integrity
    - Generate plan and get plan_binary
    - Calculate SHA256 hash of original binary
    - Store plan_binary in YDB
    - Retrieve plan_binary from YDB
    - Calculate SHA256 hash of retrieved binary
    - Verify hashes match (byte-by-byte integrity)
    - _Requirements: 6.5_

- [ ] 9. Create provider plugin caching tests (test_opentofu_cache_integration.py)
  - [ ] 9.1 Test plugin cache to S3
    - Use `opentofu_config_service` with artifact_cache configured
    - Run `tofu init` to download providers
    - Get plugins directory: `.terraform/providers/`
    - Call `cache_provider_plugins()` method
    - Use boto3 to verify plugins exist in S3 cache bucket
    - Verify S3 key structure: `terraform-plugins/{source}/{version}/`
    - _Requirements: 7.1, 7.2_

  - [ ] 9.2 Test plugin restore from S3
    - Cache plugins to S3 first
    - Delete local `.terraform/providers/` directory
    - Call `restore_cached_plugins()` method
    - Verify method returns True (all restored)
    - Verify plugins exist in local directory
    - Run `tofu validate` to verify plugins are functional
    - _Requirements: 7.3, 7.5_

  - [ ] 9.3 Test cache performance improvement
    - Measure time for `tofu init` without cache (first run)
    - Clear local plugins, cache to S3
    - Measure time for `restore_cached_plugins()` + `tofu init`
    - Verify cached init is >= 50% faster than uncached
    - _Requirements: 7.4_

- [ ] 10. Create end-to-end workflow tests (test_opentofu_e2e_integration.py)
  - [ ] 10.1 Test complete workflow without caching
    - Use all fixtures: `opentofu_updater`, `s3_backend_config`, `opentofu_workspace`, `test_ydb_schema`
    - Execute workflow:
      1. Verify binary downloaded (check updater.c_version)
      2. Generate templates (call setup_topfu_configuration)
      3. Run `tofu init`
      4. Generate plan (call generate_deployment_plan)
      5. Store plan in YDB
    - Verify state exists in S3 using boto3
    - Verify plan exists in YDB using AsyncYDBOperations
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [ ] 10.2 Test complete workflow with caching
    - Execute workflow with artifact_cache_bucket configured
    - Cache plugins after first init
    - Clear local plugins
    - Execute workflow again with restore_cached_plugins
    - Verify results are consistent with first run
    - Verify state in S3 matches
    - _Requirements: 8.2, 8.5_

  - [ ] 10.3 Test workflow timing
    - Use `time.time()` to measure total workflow execution
    - Execute complete workflow from download to plan storage
    - Verify total time < 300 seconds (5 minutes)
    - Log timing breakdown for each step
    - _Requirements: 8.6, 10.1_

  - [ ] 10.4 Test workflow idempotency
    - Run complete workflow with specific configuration
    - Store resulting state and plan
    - Run workflow again with same configuration
    - Verify state files are identical (SHA256 hash)
    - Verify plan shows no changes (exit code 0)
    - _Requirements: 8.2_

- [ ] 11. Create error handling tests (test_opentofu_errors_integration.py)
  - [ ] 11.1 Test S3 backend unavailable
    - Use `opentofu_config_service` fixture
    - Stop LocalStack container using testcontainers API
    - Attempt `tofu init` using subprocess
    - Verify non-zero exit code
    - Verify error message mentions S3 or backend
    - Restart container for cleanup
    - _Requirements: 9.1_

  - [ ] 11.2 Test invalid provider version
    - Create `TofuSetting` with non-existent provider version (e.g., "999.999.999")
    - Create `OpenTofuConfiguration` with invalid settings
    - Call `setup_topfu_configuration()`
    - Attempt `tofu init`
    - Verify non-zero exit code
    - Verify error message mentions provider or version
    - _Requirements: 9.2_

  - [ ] 11.3 Test invalid HCL syntax
    - Use `opentofu_workspace` fixture
    - Write malformed HCL to versions.tf (e.g., missing closing brace)
    - Run `tofu validate` using subprocess
    - Verify non-zero exit code
    - Verify error message mentions syntax or parsing
    - _Requirements: 9.3_

  - [ ] 11.4 Test plan generation failure
    - Create invalid resource configuration (e.g., missing required argument)
    - Run `tofu plan` using subprocess
    - Verify non-zero exit code (exit code 1 for errors)
    - Verify stderr contains error description
    - _Requirements: 9.4_

  - [ ] 11.5 Test cleanup on failure
    - Use pytest fixture with yield for setup/teardown
    - Simulate failure during workflow (raise exception)
    - Verify workspace directory is cleaned up (in finally block)
    - Verify temporary files are removed
    - Note: Containers are session-scoped, so they persist
    - _Requirements: 9.5_

- [ ] 12. Implement test performance optimizations
  - [ ] 12.1 Binary caching optimization
    - Session-scoped `opentofu_updater` fixture already caches binary
    - Binary stored in `/mnt/tofu_binary/{version}/`
    - Subsequent test runs reuse cached binary automatically
    - Verify checksum only on first download
    - _Requirements: 10.3_

  - [ ] 12.2 Container reuse optimization
    - Existing fixtures already use session scope:
      - `ydb_container` (session-scoped)
      - `localstack_container` (session-scoped)
    - Containers start once per test session
    - All tests reuse same container instances
    - _Requirements: 10.3_

  - [ ] 12.3 Provider caching optimization
    - Use `S3ArtifactCache` for provider caching
    - Cache providers after first download
    - Restore from cache in subsequent tests
    - Measure and log performance improvement
    - _Requirements: 10.3_

  - [ ] 12.4 Configure parallel execution
    - Add pytest-xdist to dependencies
    - Configure pytest.ini with `-n auto` for parallel tests
    - Ensure each test uses isolated workspace (function-scoped fixture)
    - Verify tests can run in parallel without conflicts
    - Test with: `pytest tests/integration/ -n auto`
    - _Requirements: 10.1_

- [ ] 13. Add test documentation and CI integration
  - Create `tests/integration/README.md` with:
    - Overview of integration test suite
    - Prerequisites (Docker, disk space requirements)
    - How to run tests locally: `pytest tests/integration/ -v -m integration`
    - How to run specific test files
    - Environment variables needed (if any)
    - Troubleshooting common issues
  - Document required Docker setup:
    - Docker daemon must be running
    - Sufficient disk space for containers (~2GB)
    - Network access for downloading containers
  - Add integration tests to CI pipeline:
    - Create GitHub Actions workflow or update existing
    - Install Docker in CI environment
    - Run integration tests with timeout
    - Upload test results as artifacts
  - Configure test timeouts:
    - Add pytest-timeout to dependencies
    - Set timeout decorator on slow tests: `@pytest.mark.timeout(300)`
    - Configure global timeout in pytest.ini
  - _Requirements: 10.4, 10.5_

- [ ] 14. Final validation and cleanup
  - Run complete test suite: `pytest tests/integration/ -v -m integration`
  - Verify all tests pass
  - Check test coverage: `pytest tests/integration/ --cov=app.services.opentofu_configuration --cov=app.services.opentofu_binary`
  - Verify coverage meets requirements (>80% for integration-tested code)
  - Verify performance requirements met:
    - Complete workflow < 5 minutes
    - Cached workflow >= 50% faster
  - Clean up any test artifacts:
    - Remove temporary directories
    - Clear S3 test buckets (if not auto-cleaned)
    - Clear YDB test tables (if not auto-cleaned)
  - Document any known issues or limitations
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

## Notes

- Each task references specific requirements for traceability
- Tests should be implemented incrementally, validating each component before moving to the next
- Use `@pytest.mark.integration` marker for all integration tests
- Use `@pytest.mark.slow` marker for tests taking > 30 seconds
- Ensure Docker is running before executing integration tests
- Consider using pytest-timeout to prevent hanging tests
- **REUSE EXISTING FIXTURES**: `ydb_container`, `localstack_container`, `aws_credentials` from `tests/conftest.py`
- **USE EXISTING CLASSES**: `OpenTofuUpdateGithub`, `OpenTofuConfiguration`, schema classes
- Tests go in existing `tests/integration/` directory, not a new subdirectory
- Session-scoped fixtures provide performance optimization by reusing containers
- Function-scoped fixtures ensure test isolation for workspaces and configurations
