# Implementation Plan: Deploy Bootstrap Secrets

## Overview

Transform the `gosling deploy` command into a two-phase workflow (Bootstrap_Phase → Egg_Phase) that automatically provisions MG API credentials in cloud secret stores (YC Lockbox / AWS Secrets Manager). All new code lives under `dev-new-features/internal/bootstrap/` with modifications to `dev-new-features/internal/cli/deploy.go` and `dev-new-features/internal/deployer/deployer.go`.

## Tasks

- [x] 1. Create bootstrap package with core interfaces and types
  - [x] 1.1 Create `dev-new-features/internal/bootstrap/bootstrap.go` with `MGSecretStore` interface, `BackendDeployer` interface, `SecretBootstrapper` struct, and all data types (`MGSecret`, `Credentials`, `DeployResult`, `BootstrapResult`)
    - Define `MGSecretStore` interface with `Discover`, `Create`, `Update` methods
    - Define `BackendDeployer` interface with `DeployBackend` method returning `(*DeployResult, error)`
    - Define `SecretBootstrapper` struct holding `store MGSecretStore` and `deployer BackendDeployer`
    - Define `Credentials`, `MGSecret`, `DeployResult`, `BootstrapResult` structs
    - Define `ErrSecretDeleted` sentinel error for scheduled-deletion detection
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 3.1, 3.2, 4.1, 4.2_

  - [x] 1.2 Create `dev-new-features/internal/bootstrap/validate.go` with `ValidateDeployFlags` function
    - Implement both-or-neither validation logic for `apiURL` and `apiKey`
    - Implement HTTP(S) URL format validation for `apiURL` when non-empty
    - Return `(useBootstrap bool, err error)` tuple
    - Use `net/url` for URL parsing and scheme validation
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [x] 1.3 Implement `SecretBootstrapper.Bootstrap` method in `dev-new-features/internal/bootstrap/bootstrap.go`
    - Call `store.Discover` to check for existing secret
    - If discovered with populated credentials → return `BootstrapResult{WasDiscovered: true}`
    - If discovered with empty credentials → call `deployer.DeployBackend` then `store.Update`
    - If not found → call `store.Create` then `deployer.DeployBackend` then `store.Update`
    - Handle `ErrSecretDeleted` by treating as not-found and proceeding with creation
    - Validate `DeployResult.APIGatewayURL` is a valid HTTPS URL before updating secret
    - Use 30-second context deadline for secret store operations
    - _Requirements: 2.4, 2.5, 2.6, 2.7, 4.1, 4.2, 4.3, 4.6_

- [x] 2. Implement Yandex Cloud secret store
  - [x] 2.1 Create `dev-new-features/internal/bootstrap/ycstore/yc_store.go` implementing `MGSecretStore` for YC Lockbox
    - Implement `Discover`: list secrets in folder, match name `pg-mothergoose-secrets`, check status (skip `PENDING_DELETION`)
    - Implement `Create`: create secret with labels `polar-gosling=true`, `resource-type=mothergoose-api`, first version with empty `api-url` and `api-key` entries
    - Implement `Update`: add new version with populated `api-url` and `api-key` entries
    - Use `github.com/yandex-cloud/go-sdk` and `github.com/yandex-cloud/go-genproto` for YC Lockbox API
    - Return secret URI using `yc-lockbox://` scheme
    - _Requirements: 2.2, 2.6, 3.1, 3.3, 3.4, 3.5, 4.4_

  - [x] 2.2 Write unit tests for `ycstore` in `dev-new-features/internal/bootstrap/ycstore/yc_store_test.go`
    - Test secret name is exactly `pg-mothergoose-secrets`
    - Test labels include `polar-gosling=true` and `resource-type=mothergoose-api`
    - Test `PENDING_DELETION` status treated as not-found
    - Test error wrapping for auth failures returns `lockbox.editor` role guidance
    - _Requirements: 2.2, 2.6, 3.3, 6.2_

- [x] 3. Implement AWS secret store
  - [x] 3.1 Create `dev-new-features/internal/bootstrap/awsstore/aws_store.go` implementing `MGSecretStore` for AWS Secrets Manager
    - Implement `Discover`: describe secret by name `polar-gosling/mothergoose`, handle `ResourceNotFoundException`, check deletion date
    - Implement `Create`: create secret with JSON `{"api-url":"","api-key":""}`, tags `polar-gosling=true`, `resource-type=mothergoose-api`
    - Implement `Update`: put secret value with populated JSON
    - Use `github.com/aws/aws-sdk-go-v2/service/secretsmanager`
    - Return secret URI using `aws-sm://` scheme
    - _Requirements: 2.3, 2.6, 3.2, 3.3, 3.4, 3.5, 4.5_

  - [x] 3.2 Write unit tests for `awsstore` in `dev-new-features/internal/bootstrap/awsstore/aws_store_test.go`
    - Test secret name is exactly `polar-gosling/mothergoose`
    - Test tags include `polar-gosling=true` and `resource-type=mothergoose-api`
    - Test deleted secret (with `DeletedDate`) treated as not-found
    - Test error wrapping for permission failures includes `secretsmanager:CreateSecret` and `secretsmanager:PutSecretValue`
    - _Requirements: 2.3, 2.6, 3.3, 6.3_

- [x] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Modify deployer to return credentials
  - [ ] 5.1 Update `dev-new-features/internal/deployer/deployer.go` to return `(*DeployResult, error)` from `DeployBackendInfrastructure`
    - Add `DeployResult` struct with `APIGatewayURL` and `APIKey` fields to deployer package
    - Change `DeployBackendInfrastructure` signature from returning `error` to `(*DeployResult, error)`
    - Update `CloudDeployer` interface in `types.go` to match new return type
    - Update `AWSClient.DeployBackendInfrastructure` and `YandexCloudClient.DeployBackendInfrastructure` signatures
    - _Requirements: 4.1, 4.2_

  - [ ] 5.2 Create `dev-new-features/internal/bootstrap/deployer_adapter.go` to adapt existing `deployer.Deployer` to `BackendDeployer` interface
    - Wrap `deployer.Deployer` in an adapter struct
    - Convert `deployer.DeployResult` to `bootstrap.DeployResult`
    - _Requirements: 4.1, 4.2, 4.3_

- [ ] 6. Modify deploy command for two-phase workflow
  - [ ] 6.1 Update `dev-new-features/internal/cli/deploy.go` to make `--api-url` and `--api-key` optional
    - Remove `mustMarkRequired` calls for `api-url` and `api-key` flags
    - Add call to `bootstrap.ValidateDeployFlags` at start of `runDeploy`
    - Exit with non-zero code and descriptive error on validation failure
    - _Requirements: 1.1, 1.4, 1.5_

  - [ ] 6.2 Implement Bootstrap_Phase logic in `dev-new-features/internal/cli/deploy.go`
    - When `ValidateDeployFlags` returns `useBootstrap=true`, instantiate `SecretBootstrapper` with appropriate `MGSecretStore` (YC or AWS based on `MGConfig.Cloud.Provider`)
    - Call `bootstrapper.Bootstrap(ctx, mgCfg, ufCfg)`
    - Use returned `BootstrapResult.APIURL` and `BootstrapResult.APIKey` for Egg_Phase
    - Print progress messages: "bootstrapping" + provider name, secret ID/URI, API Gateway URL, credential write confirmation
    - _Requirements: 1.3, 5.1, 9.1, 9.2, 9.3, 9.4_

  - [ ] 6.3 Implement Egg_Phase integration with bootstrapped credentials in `dev-new-features/internal/cli/deploy.go`
    - Create `mothergoose.NewClient` with bootstrapped or user-provided credentials
    - Print credential source (bootstrapped vs user-provided) before Egg_Phase
    - Skip Egg_Phase when `Eggs/` directory is absent or contains no valid configs; print skip message
    - Implement stop-on-failure: report failed Egg name and count of previously successful Eggs
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 9.5, 9.6_

  - [ ] 6.4 Implement error handling and recovery output in `dev-new-features/internal/cli/deploy.go`
    - On backend failure after secret creation: print to stderr that secret exists with empty values, retry instruction
    - On secret update failure after backend success: print API URL and key to stdout, print `--api-url`/`--api-key` instruction to stderr
    - On Egg_Phase failure: print API URL and key to stdout, retry instruction with flags to stderr
    - Never roll back completed phases; always exit non-zero on failure
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [ ] 6.5 Implement error handling for secret store access in `dev-new-features/internal/cli/deploy.go`
    - Detect authentication failures: print provider name, resource identifier, required IAM permissions
    - Detect permission errors for create/list/update: print specific required roles (`lockbox.editor`, `lockbox.viewer`, `secretsmanager:CreateSecret`, `secretsmanager:PutSecretValue`, `secretsmanager:ListSecrets`)
    - Detect 30-second timeout: print connectivity failure with unreachable endpoint
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6_

  - [ ] 6.6 Update dry-run output in `dev-new-features/internal/cli/deploy.go` for two-phase plan
    - When `--dry-run` and credentials omitted: show MG_API_Secret name, provider, folder-id/region
    - When `--dry-run` and credentials provided: show Egg_Phase plan only, skip Bootstrap_Phase display
    - Display both Bootstrap_Phase and Egg_Phase sections with resource details
    - Ensure no write API calls are made; exit with code 0
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 7. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 8. Write property-based tests for bootstrap logic
  - [ ]* 8.1 Write property test for flag validation consistency in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 1: Flag Validation Consistency**
    - Generate random string pairs (valid URLs, invalid URLs, empty strings, non-empty non-URL strings)
    - Assert correct (useBootstrap, error) result for all four quadrants
    - **Validates: Requirements 1.2, 1.3, 1.4, 1.5**

  - [ ]* 8.2 Write property test for discovery path determination in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 2: Discovery Determines Deployment Path**
    - Generate `MGSecret` instances with varying populated states
    - Assert `WasDiscovered=true` and no `DeployBackend` call when both fields non-empty
    - Assert `DeployBackend` is called when either field is empty
    - **Validates: Requirements 2.4, 2.5**

  - [ ]* 8.3 Write property test for deploy result validity in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 3: DeployResult Validity**
    - Generate `DeployResult` with valid/invalid HTTPS URLs and empty/non-empty keys
    - Assert error returned when URL is invalid or key is empty
    - **Validates: Requirements 4.1, 4.2, 4.6**

  - [ ]* 8.4 Write property test for dry-run read-only behavior in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 5: Dry-Run Is Read-Only**
    - Generate configs, run bootstrap with dry-run=true, count mock write calls
    - Assert zero writes to secret store, deployer, and MG client
    - **Validates: Requirements 8.4**

  - [ ]* 8.5 Write property test for stop-on-failure reporting in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 6: Stop-On-Failure Reports Progress**
    - Generate Egg name lists of length 1–20 with random failure position K
    - Assert error contains failed Egg name and K successful count
    - **Validates: Requirements 5.5**

  - [ ]* 8.6 Write property test for failure recovery output in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 7: Failure Recovery Prints Credentials**
    - Generate random URL+key pairs, trigger secret-update or Egg_Phase failures
    - Assert both values appear in stdout capture
    - **Validates: Requirements 7.2, 7.3**

  - [ ]* 8.7 Write property test for dry-run output completeness in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 8: Dry-Run Output Completeness**
    - Generate `MGConfig` variants (yandex/aws with varying IDs/regions)
    - Assert output contains MG_API_Secret name, provider, and location identifier
    - **Validates: Requirements 8.1, 8.3**

  - [ ]* 8.8 Write property test for bootstrap output identifiers in `dev-new-features/internal/bootstrap/bootstrap_property_test.go`
    - **Property 9: Bootstrap Output Contains Identifiers**
    - Generate bootstrap scenarios, capture stdout
    - Assert output contains "bootstrapping", provider name, secret ID, URI, API Gateway URL
    - **Validates: Requirements 9.1, 9.2, 9.3, 9.4**

- [x] 9. Add `aws-sdk-go-v2/service/secretsmanager` dependency
  - [x] 9.1 Add `github.com/aws/aws-sdk-go-v2/service/secretsmanager` to `dev-new-features/go.mod` and run `go mod tidy`
    - This new AWS SDK module is required by the `awsstore` package for Secrets Manager operations
    - _Requirements: 2.3, 3.2_

- [ ] 10. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using `gopter` (existing project dependency)
- Unit tests validate specific examples and edge cases
- The design specifies Go as the implementation language (Gosling CLI)
- All new code goes in `dev-new-features/internal/bootstrap/` and subpackages
- The existing `lockbox` package is NOT modified — bootstrap uses a separate package with different semantics
- The `deployer.DeployBackendInfrastructure` signature change (task 5.1) will require updating callers in `deploy.go`

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2"] },
    { "id": 1, "tasks": ["1.3", "2.1", "3.1", "9.1"] },
    { "id": 2, "tasks": ["2.2", "3.2", "5.1"] },
    { "id": 3, "tasks": ["5.2"] },
    { "id": 4, "tasks": ["6.1"] },
    { "id": 5, "tasks": ["6.2", "6.5"] },
    { "id": 6, "tasks": ["6.3", "6.4", "6.6"] },
    { "id": 7, "tasks": ["8.1", "8.2", "8.3", "8.4", "8.5", "8.6", "8.7", "8.8"] }
  ]
}
```
