# Design Document: OpenTofu Integration Testing

## Overview

This design document describes the architecture and implementation approach for comprehensive integration testing of the OpenTofu configuration service. The tests will validate the complete workflow from binary download through deployment plan storage using real infrastructure components (LocalStack, YDB testcontainers) and actual OpenTofu binaries.

## Architecture

### Test Structure

```
tests/
├── conftest.py                          # EXISTING - Session fixtures (YDB, LocalStack)
├── integration/
│   ├── __init__.py
│   ├── test_opentofu_integration.py     # NEW - OpenTofu integration tests
│   └── fixtures/
│       └── test_configs/                # NEW - Test HCL configurations
└── unit/
    └── test_opentofu_configuration.py   # EXISTING - Unit tests
```

**Note:** We will **reuse existing session-scoped fixtures** from `tests/conftest.py`:
- `ydb_container` - Already provides YDB testcontainer
- `localstack_container` - Already provides LocalStack with S3/DynamoDB
- `aws_credentials` - Already provides LocalStack connection details

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Integration Test Suite                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Binary     │  │   Template   │  │   OpenTofu   │      │
│  │  Download    │→ │  Generation  │→ │     Init     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         ↓                                      ↓             │
│  ┌──────────────┐                    ┌──────────────┐      │
│  │   Checksum   │                    │   Provider   │      │
│  │ Verification │                    │   Download   │      │
│  └──────────────┘                    └──────────────┘      │
│                                               ↓             │
│                                      ┌──────────────┐      │
│                                      │  Plan Gen    │      │
│                                      └──────────────┘      │
│                                               ↓             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              External Test Infrastructure            │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐    │  │
│  │  │ LocalStack │  │    YDB     │  │  S3 Cache  │    │  │
│  │  │ Container  │  │ Container  │  │   Bucket   │    │  │
│  │  └────────────┘  └────────────┘  └────────────┘    │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Test Fixtures (tests/conftest.py - additions only)

**Note:** The following fixtures **already exist** in `tests/conftest.py` and will be reused:
- ✅ `ydb_container` (session-scoped) - YDB testcontainer
- ✅ `localstack_container` (session-scoped) - LocalStack with S3/DynamoDB
- ✅ `aws_credentials` (session-scoped) - LocalStack connection details

#### New Session-Scoped Fixtures to Add

**`opentofu_updater`**
- Creates `OpenTofuUpdateGithub` instance
- Downloads and verifies OpenTofu binary from GitHub
- Uses existing MotherGoose binary management
- Returns updater instance with `c_version` property
- Binary is cached automatically by the updater class

#### New Function-Scoped Fixtures to Add

**`s3_backend_config`**
- Uses existing `localstack_container` and `aws_credentials` fixtures
- Creates S3 bucket in LocalStack
- Creates DynamoDB table for state locking
- Returns `TofuBackendS3Options` instance
- Cleanup: Deletes bucket and table

**`opentofu_workspace`**
- Creates temporary directory for OpenTofu workspace
- Returns workspace path
- Cleanup: Removes directory

**`opentofu_config_service`**
- Creates `OpenTofuConfiguration` instance using real `OpenTofuUpdateGithub`
- Configures with `TofuSetting` (providers, backend, cache)
- Uses LocalStack S3 backend (via existing fixtures)
- Returns configured service ready for testing

### 2. Binary Download and Validation

**Use Existing Classes:** `OpenTofuUpdateGithub`, `OpenTofuDownloadGithub`

The integration tests will use the existing MotherGoose backend classes for binary management:

```python
from app.services.opentofu_binary import OpenTofuUpdateGithub
from app.services.opentofu_configuration import OpenTofuConfiguration, TofuSetting

# In test fixtures
@pytest.fixture(scope="session")
def opentofu_updater():
    """Fixture providing real OpenTofu updater."""
    updater = OpenTofuUpdateGithub()
    # This will download and verify the binary from GitHub
    updater.start_update(rb=3)  # rollback factor
    return updater
```

**Implementation Details:**
- Use `OpenTofuUpdateGithub` class to download binaries from GitHub
- The class already handles SHA256 verification via `get_sha256_hash_of_bundle_from_github()`
- Binary is automatically cached in `/mnt/tofu_binary/{version}/`
- The `c_version` property provides (version_id, version_string)
- No need to create custom binary manager - use existing infrastructure

### 3. Template Generation and Validation

**Test Class:** `TestTemplateGeneration`

**Templates to Test:**

1. **tofu_versions_tf.j2** → `versions.tf`
   - Backend configuration (S3)
   - Required version constraints
   - Required providers (local, tls, random, custom providers)

2. **tofu_checks_tf.j2** → `checks.tf`
   - Health check blocks
   - Check conditions and URLs

3. **cloud-init-runner.tpl.j2** → cloud-init YAML
   - Runner initialization script
   - SSH key configuration
   - Gosling binary installation

4. **tofu_providers_tf.j2** → `providers.tf`
   - Provider configuration blocks
   - Yandex Cloud provider settings
   - AWS provider settings

5. **tofu_resources_tf.j2** → `resources.tf`
   - TLS private key resource
   - Local file resource for SSH keys
   - Worker module with for_each
   - Rift module (conditional)
   - Cloud-init injection

6. **tofu_variables_tf.j2** → `variables.tf`
   - Worker instances variable
   - Rift instances variable (conditional)

7. **tofu_data_tf.j2** → `data.tf`
   - Data source blocks (currently empty, prepared for future use)

8. **tofu_rc.j2** → `.terraformrc` or `.tofurc`
   - Provider installation configuration
   - Plugin cache directory

**Methods:**
- `test_versions_tf_generation()` - Validate versions.tf template
- `test_backend_configuration()` - Validate S3 backend config
- `test_provider_constraints()` - Validate provider versions
- `test_health_checks_generation()` - Validate checks.tf template
- `test_cloud_init_generation()` - Validate cloud-init YAML
- `test_providers_tf_generation()` - Validate providers.tf template
- `test_resources_tf_generation()` - Validate resources.tf template
- `test_resources_tf_with_rift()` - Validate resources.tf with Rift module
- `test_variables_tf_generation()` - Validate variables.tf template
- `test_terraformrc_generation()` - Validate .terraformrc template
- `test_optional_parameters()` - Test templates with optional parameters
- `test_hcl_syntax_validation()` - Run `tofu fmt -check` on all generated files
- `test_hcl_semantic_validation()` - Run `tofu validate` on all generated files

**Validation Strategy:**
- Generate templates using real service
- Parse generated HCL using `python-hcl2` library
- Verify all expected blocks are present
- Verify provider sources and versions match configuration
- Verify module sources (git vs registry)
- Verify for_each iteration over instances
- Verify conditional blocks (Rift module)
- Run OpenTofu validation commands on complete configuration

### 4. LocalStack S3 Backend Integration

**Test Class:** `TestS3BackendIntegration`

**Methods:**
- `test_s3_bucket_creation()` - Verify bucket creation
- `test_dynamodb_table_creation()` - Verify lock table creation
- `test_backend_initialization()` - Test `tofu init` with S3 backend
- `test_state_storage()` - Verify state is stored in S3
- `test_state_locking()` - Verify DynamoDB locking works

**Implementation Details:**
- Use `boto3` to interact with LocalStack
- Verify state file exists in S3 after operations
- Test concurrent access with locking
- Validate state file structure

### 5. OpenTofu Initialization

**Test Class:** `TestOpenTofuInitialization`

**Methods:**
- `test_provider_download()` - Verify providers are downloaded
- `test_terraform_directory_creation()` - Verify .terraform structure
- `test_lock_file_generation()` - Verify .terraform.lock.hcl
- `test_backend_initialization()` - Verify backend is configured
- `test_provider_verification()` - Verify providers are functional

**Validation Strategy:**
- Run `tofu init` with real providers
- Check `.terraform/providers/` directory structure
- Verify lock file contains expected provider versions
- Parse lock file to validate checksums

### 6. Deployment Plan Generation

**Test Class:** `TestDeploymentPlanGeneration`

**Methods:**
- `test_plan_generation_no_changes()` - Test with no infrastructure changes
- `test_plan_generation_with_changes()` - Test with resource changes
- `test_plan_binary_format()` - Verify plan file is valid binary
- `test_plan_exit_codes()` - Verify correct exit codes (0, 2)
- `test_plan_readability()` - Verify plan can be read back

**Test Infrastructure:**
- Use OpenTofu's `null_resource` or `local_file` for testing
- Create simple test configurations that produce predictable plans
- Avoid creating real cloud resources

### 7. YDB Plan Storage Integration

**Test Class:** `TestYDBPlanStorage`

**Methods:**
- `test_ydb_table_creation()` - Verify tables are created
- `test_plan_storage()` - Store plan binary in YDB
- `test_plan_retrieval()` - Retrieve plan from YDB
- `test_plan_metadata()` - Verify metadata is stored correctly
- `test_binary_integrity()` - Verify binary data integrity

**Schema:**
```python
DeploymentPlan:
    - egg_name: str (primary key)
    - config_hash: str (sort key)
    - git_commit: str
    - plan_binary: bytes
    - created_at: timestamp
    - status: str
```

### 8. Provider Plugin Caching

**Test Class:** `TestProviderPluginCaching`

**Methods:**
- `test_plugin_cache_to_s3()` - Cache plugins after download
- `test_plugin_restore_from_s3()` - Restore plugins from cache
- `test_cache_performance()` - Verify caching improves performance
- `test_cache_organization()` - Verify S3 key structure
- `test_cached_plugin_functionality()` - Verify cached plugins work

**Cache Structure:**
```
s3://cache-bucket/
  terraform-plugins/
    hashicorp/
      aws/
        5.0.0/
          terraform-provider-aws_5.0.0_linux_amd64
    yandex-cloud/
      yandex/
        0.100.0/
          terraform-provider-yandex_0.100.0_linux_amd64
```

### 9. End-to-End Workflow

**Test Class:** `TestEndToEndWorkflow`

**Methods:**
- `test_complete_workflow()` - Execute full workflow
- `test_workflow_with_caching()` - Test with cached components
- `test_workflow_timing()` - Verify performance requirements
- `test_workflow_state_consistency()` - Verify state across components

**Workflow Steps:**
1. Download and verify OpenTofu binary
2. Generate HCL templates
3. Initialize OpenTofu with S3 backend
4. Download provider plugins
5. Cache plugins to S3
6. Generate deployment plan
7. Store plan in YDB
8. Verify state in S3
9. Verify plan in YDB
10. Verify plugins in S3 cache

### 10. Error Handling

**Test Class:** `TestErrorHandling`

**Methods:**
- `test_s3_backend_unavailable()` - Test with stopped LocalStack
- `test_invalid_provider_version()` - Test with non-existent provider
- `test_invalid_hcl_syntax()` - Test with malformed templates
- `test_plan_generation_failure()` - Test with invalid configuration
- `test_cleanup_on_failure()` - Verify cleanup happens

## Data Models

### Test Configuration

```python
@dataclass
class IntegrationTestConfig:
    """Configuration for integration tests."""
    opentofu_version: str = "1.6.0"
    test_providers: List[TofuProvidersVer]
    s3_bucket: str
    s3_region: str
    dynamodb_table: str
    cache_bucket: str
    ydb_endpoint: str
    ydb_database: str
```

### Test Resources

```python
@dataclass
class TestInfrastructure:
    """Test infrastructure resources."""
    null_resource_count: int = 3
    local_file_count: int = 2
    expected_plan_changes: int = 5
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Binary Checksum Verification
*For any* downloaded OpenTofu binary, the SHA256 checksum must match the official GitHub release checksum
**Validates: Requirements 1.2**

### Property 2: Template Syntax Validity
*For any* generated HCL template, running `tofu fmt -check` must return exit code 0 (valid syntax)
**Validates: Requirements 2.5**

### Property 3: Backend Initialization Success
*For any* valid S3 backend configuration, running `tofu init` must successfully initialize the backend and return exit code 0
**Validates: Requirements 3.4**

### Property 4: Provider Plugin Completeness
*For any* list of required providers, after `tofu init`, all providers must be present in `.terraform/providers/` directory
**Validates: Requirements 4.5**

### Property 5: Plan Binary Integrity
*For any* generated deployment plan, storing and retrieving the plan binary from YDB must produce identical bytes
**Validates: Requirements 6.5**

### Property 6: Cache Performance Improvement
*For any* OpenTofu initialization with cached plugins, the execution time must be at least 50% faster than without cache
**Validates: Requirements 7.4**

### Property 7: Workflow Idempotency
*For any* OpenTofu configuration, running the complete workflow twice must produce consistent results (same state, same plan)
**Validates: Requirements 8.2**

### Property 8: State Consistency
*For any* OpenTofu operation, the state stored in S3 must be valid JSON and parseable by OpenTofu
**Validates: Requirements 8.3**

## Error Handling

### Error Scenarios

1. **Binary Download Failure**
   - Retry with exponential backoff (3 attempts)
   - Fall back to cached binary if available
   - Clear error message with GitHub API status

2. **Template Generation Failure**
   - Validate input configuration before generation
   - Provide detailed error with template name and line number
   - Preserve partial output for debugging

3. **Backend Initialization Failure**
   - Check S3 bucket accessibility before init
   - Verify DynamoDB table exists
   - Provide clear error with backend configuration details

4. **Provider Download Failure**
   - Retry provider download (3 attempts)
   - Check provider registry accessibility
   - Provide clear error with provider name and version

5. **Plan Generation Failure**
   - Capture and return OpenTofu error output
   - Preserve workspace for debugging
   - Provide context about configuration being planned

### Cleanup Strategy

- Use pytest fixtures with `yield` for automatic cleanup
- Implement `finally` blocks for critical cleanup operations
- Stop containers even if tests fail
- Remove temporary directories
- Clear S3 buckets and DynamoDB tables

## Testing Strategy

### Test Organization

**Unit Tests** (existing):
- Mock external dependencies
- Fast execution (< 1 second per test)
- Test individual methods and functions

**Integration Tests** (this spec):
- Use real OpenTofu binaries
- Use testcontainers for infrastructure
- Slower execution (< 5 minutes total)
- Test complete workflows
- Marked with `@pytest.mark.integration`

### Test Execution

```bash
# Run only integration tests
pytest tests/integration/ -v -m integration

# Run with coverage
pytest tests/integration/ --cov=app.services.opentofu_configuration

# Run specific test class
pytest tests/integration/test_opentofu_integration.py::TestEndToEndWorkflow -v
```

### Performance Optimization

1. **Binary Caching**
   - Cache downloaded binaries in `~/.cache/`
   - Reuse across test runs
   - Verify checksum on first use only

2. **Container Reuse**
   - Use session-scoped fixtures for containers
   - Start containers once per test session
   - Reuse for all tests

3. **Provider Caching**
   - Cache downloaded providers
   - Restore from cache in subsequent tests
   - Reduces test time by 50-70%

4. **Parallel Execution**
   - Use `pytest-xdist` for parallel test execution
   - Isolate test workspaces
   - Avoid shared state between tests

### Test Data

**Test Configurations:**
```hcl
# Simple null resource configuration
resource "null_resource" "test" {
  count = 3
  
  triggers = {
    timestamp = timestamp()
  }
}

# Local file configuration
resource "local_file" "test" {
  content  = "test content"
  filename = "${path.module}/test.txt"
}
```

**Provider Versions:**
- `hashicorp/null`: `3.2.0`
- `hashicorp/local`: `2.4.0`

## Implementation Notes

### Dependencies

Add to `pyproject.toml`:
```toml
[tool.poetry.group.test.dependencies]
pytest = "^8.0.0"
pytest-asyncio = "^0.23.0"
pytest-xdist = "^3.5.0"
testcontainers = "^3.7.0"
python-hcl2 = "^4.3.0"
requests = "^2.31.0"
```

### Environment Variables

```bash
# Test configuration
OPENTOFU_TEST_VERSION=1.6.0
OPENTOFU_CACHE_DIR=~/.cache/opentofu-test-binaries
INTEGRATION_TEST_TIMEOUT=300

# Container configuration
LOCALSTACK_IMAGE=localstack/localstack:latest
YDB_IMAGE=ydbplatform/local-ydb:latest
```

### Pytest Markers

```python
# pytest.ini or pyproject.toml
[tool.pytest.ini_options]
markers = [
    "integration: Integration tests (deselect with '-m \"not integration\"')",
    "slow: Slow tests (> 30 seconds)",
    "requires_docker: Tests that require Docker",
]
```

## Security Considerations

1. **Binary Verification**
   - Always verify SHA256 checksums
   - Download from official GitHub releases only
   - Use HTTPS for all downloads

2. **Credential Management**
   - Use test credentials only
   - Never commit real credentials
   - Rotate test credentials regularly

3. **Container Isolation**
   - Use Docker networks for isolation
   - Limit container resource usage
   - Clean up containers after tests

4. **Temporary Files**
   - Use secure temporary directories
   - Clean up sensitive data
   - Set appropriate file permissions

## Future Enhancements

1. **Multi-Version Testing**
   - Test with multiple OpenTofu versions
   - Validate upgrade paths
   - Test backward compatibility

2. **Provider Matrix Testing**
   - Test with different provider versions
   - Test provider compatibility
   - Validate provider constraints

3. **Performance Benchmarking**
   - Track test execution time over time
   - Identify performance regressions
   - Optimize slow tests

4. **Chaos Testing**
   - Simulate network failures
   - Test container crashes
   - Validate recovery mechanisms
