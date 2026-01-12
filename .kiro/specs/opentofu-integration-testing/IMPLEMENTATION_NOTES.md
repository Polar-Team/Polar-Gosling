# Implementation Notes: Using Existing MotherGoose Classes

## Overview

The integration tests will use **existing MotherGoose backend classes** rather than creating new helper classes. This ensures we're testing the actual production code paths.

## Key Classes to Use

### 1. Binary Management

**Use:** `OpenTofuUpdateGithub` from `app.services.opentofu_binary`

```python
from app.services.opentofu_binary import OpenTofuUpdateGithub

# In test fixture
@pytest.fixture(scope="session")
def opentofu_updater():
    """Fixture providing real OpenTofu updater."""
    updater = OpenTofuUpdateGithub()
    updater.start_update(rb=3)  # Download and verify binary
    return updater
```

**Features:**
- Downloads binary from GitHub releases
- Verifies SHA256 checksum automatically
- Caches binary in `/mnt/tofu_binary/{version}/`
- Provides `c_version` property: `(version_id, version_string)`
- Handles platform detection (Linux/Windows, amd64/arm64)

### 2. Configuration Management

**Use:** `OpenTofuConfiguration` from `app.services.opentofu_configuration`

```python
from app.services.opentofu_configuration import OpenTofuConfiguration, TofuSetting
from app.schema.tofu_schemas import TofuBackendS3Options, TofuProvidersVer

# In test fixture
@pytest.fixture
def opentofu_config_service(opentofu_updater, s3_backend_options):
    """Fixture providing configured OpenTofu service."""
    
    # Define providers
    providers = [
        TofuProvidersVer(
            name="null",
            version="3.2.0",
            source="hashicorp/null"
        ),
        TofuProvidersVer(
            name="local",
            version="2.4.0",
            source="hashicorp/local"
        ),
    ]
    
    # Create settings
    tofu_settings = TofuSetting(
        providers=providers,
        backend_s3_options=s3_backend_options,
        artifact_cache_bucket="test-cache-bucket",
        health_checks=[
            {
                "name": "test_health",
                "url": "https://example.com/health",
                "expected_status": 200,
            }
        ],
    )
    
    # Create configuration service
    config = OpenTofuConfiguration(
        updater=opentofu_updater,
        tofu_settings=tofu_settings,
        artifact_cache=None,  # Will be created from settings
    )
    
    # Setup configuration (generates templates)
    config.setup_topfu_configuration()
    
    return config
```

**Features:**
- Generates HCL templates from Jinja2 templates
- Configures S3 backend
- Manages provider plugins
- Handles caching to S3
- Provides methods for plan generation and application

### 3. Schema Classes

**Use:** Schema classes from `app.schema.tofu_schemas`

```python
from app.schema.tofu_schemas import (
    TofuBackendS3Options,
    TofuProvidersVer,
    OpenTofuBinFileInfo,
)

# S3 Backend Configuration
backend_options = TofuBackendS3Options(
    bucket="test-bucket",
    key="terraform.tfstate",
    region="us-east-1",
    endpoint="http://localhost:4566",  # LocalStack
    dynamodb_table="terraform-locks",
)

# Provider Configuration
provider = TofuProvidersVer(
    name="null",
    version="3.2.0",
    source="hashicorp/null",
)
```

## Test Structure

```
tests/
├── integration/
│   ├── __init__.py
│   ├── conftest.py                          # Fixtures using MotherGoose classes
│   ├── test_opentofu_integration.py         # Integration tests
│   └── fixtures/
│       └── test_configs/                    # Test HCL configurations
```

## Fixture Architecture

### Session-Scoped Fixtures

1. **`localstack_s3_container`** - LocalStack testcontainer
2. **`ydb_integration_container`** - YDB testcontainer
3. **`opentofu_updater`** - `OpenTofuUpdateGithub` instance (downloads binary once)

### Function-Scoped Fixtures

1. **`s3_backend_options`** - `TofuBackendS3Options` instance
2. **`opentofu_workspace`** - Temporary workspace directory
3. **`opentofu_config_service`** - `OpenTofuConfiguration` instance

## Integration Test Flow

```
1. Start Containers (session scope)
   ├── LocalStack (S3 + DynamoDB)
   └── YDB

2. Download Binary (session scope)
   └── OpenTofuUpdateGithub.start_update()

3. For Each Test (function scope):
   ├── Create S3 Backend Config
   ├── Create Workspace
   ├── Create OpenTofuConfiguration
   ├── Generate Templates
   ├── Run OpenTofu Commands
   ├── Verify Results
   └── Cleanup

4. Stop Containers (session teardown)
```

## Key Methods to Test

### From `OpenTofuUpdateGithub`:
- `start_update(rb=3)` - Download and verify binary
- `c_version` property - Get (version_id, version_string)
- `get_sha256_hash_of_bundle_from_github()` - Verify checksums

### From `OpenTofuConfiguration`:
- `setup_topfu_configuration()` - Generate templates and configure
- `generate_cloud_init_script()` - Generate cloud-init for runners
- `cache_provider_plugins()` - Cache plugins to S3
- `restore_cached_plugins()` - Restore plugins from S3
- `cache_terraform_directory()` - Cache .terraform to S3
- `restore_cached_terraform_directory()` - Restore .terraform from S3
- `generate_deployment_plan()` - Generate OpenTofu plan
- `apply_deployment_plan()` - Apply OpenTofu plan
- `rollback_deployment()` - Rollback deployment

## Example Test

```python
@pytest.mark.integration
class TestOpenTofuIntegration:
    """Integration tests using real MotherGoose classes."""
    
    def test_binary_download_and_verification(self, opentofu_updater):
        """Test binary download using OpenTofuUpdateGithub."""
        version_id, version_string = opentofu_updater.c_version
        
        # Verify binary was downloaded
        assert version_id != "dummy_id"
        assert version_string.startswith("1.")
        
        # Verify binary is cached
        binary_path = f"/mnt/tofu_binary/{version_string}/tofu"
        assert os.path.exists(binary_path)
        assert os.access(binary_path, os.X_OK)
    
    @pytest.mark.asyncio
    async def test_template_generation_and_init(
        self,
        opentofu_config_service,
        opentofu_workspace,
    ):
        """Test template generation and OpenTofu init."""
        # Templates already generated by setup_topfu_configuration()
        
        # Verify versions.tf exists
        versions_tf = os.path.join(opentofu_workspace, "versions.tf")
        assert os.path.exists(versions_tf)
        
        # Parse and verify HCL
        with open(versions_tf) as f:
            hcl_content = f.read()
            assert "terraform" in hcl_content
            assert "backend" in hcl_content
            assert "required_providers" in hcl_content
    
    @pytest.mark.asyncio
    async def test_plan_generation_and_storage(
        self,
        opentofu_config_service,
        ydb_integration_container,
    ):
        """Test plan generation and YDB storage."""
        # Generate plan
        plan_binary, is_valid = await opentofu_config_service.generate_deployment_plan(
            egg_name="test-app",
            config_hash="abc123",
            git_commit="def456",
        )
        
        assert is_valid
        assert len(plan_binary) > 0
        
        # Store in YDB (would use YDB client here)
        # Verify retrieval matches original
```

## Benefits of Using Existing Classes

1. **Tests Production Code** - We're testing the actual code that runs in production
2. **No Duplication** - Don't need to reimplement binary download/verification
3. **Consistent Behavior** - Tests use same logic as production
4. **Easier Maintenance** - Changes to production classes automatically tested
5. **Real Integration** - Tests validate actual integration points

## Dependencies

Already in `pyproject.toml`:
- `requests` - Used by `OpenTofuUpdateGithub`
- `boto3` / `aiobotocore` - Used by `S3ArtifactCache`
- `jinja2` - Used by `OpenTofuConfiguration`
- `tofupy` - Used by `OpenTofuConfiguration`

Need to add for integration tests:
- `testcontainers` - For LocalStack and YDB containers
- `python-hcl2` - For parsing generated HCL
- `pytest-asyncio` - For async test support
- `pytest-xdist` - For parallel test execution

## Next Steps

1. Implement fixtures in `tests/integration/conftest.py`
2. Create test classes using the fixtures
3. Verify tests work with real containers
4. Optimize for performance (caching, parallel execution)
5. Add to CI/CD pipeline
