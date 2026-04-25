# Implementation Plan: gosling-lockbox

## Overview

Implement cloud secret store management for the Gosling CLI via a `gosling lockbox` command family and integrate secret provisioning into the `gosling add egg --interactive` workflow. The implementation follows a bottom-up approach: core package first (interfaces, validation, URI helpers), then cloud provider implementations, then CLI commands, and finally interactive integration with `add egg`.

All code goes in `dev-new-features/internal/lockbox/` (new package) and `dev-new-features/internal/cli/` (new command files).

## Tasks

- [x] 1. Create the lockbox package with core interfaces and types
  - [x] 1.1 Create `internal/lockbox/store.go` with `SecretStore` interface, `CreateParams`, `CreateResult`, `SecretInfo`, `VerifyResult` types, and `RequiredEntries` variable
    - Define the `SecretStore` interface with `Create`, `List`, and `Verify` methods
    - Define all supporting structs as specified in the design
    - _Requirements: 2.1, 2.2, 3.1, 3.2, 5.1, 5.2, 6.1, 6.2_

  - [x] 1.2 Create `internal/lockbox/validate.go` with `ValidateCreateInput` and `IsValidEggName` functions
    - Implement provider validation (must be "yandex" or "aws")
    - Implement egg-name validation (non-empty, alphanumeric + hyphens + underscores)
    - Implement provider-specific required field checks (folder-id for Yandex)
    - Allow empty region for AWS (SDK default fallback)
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x] 1.3 Create `internal/lockbox/uri.go` with `GenerateSecretURI`, `GenerateAllURIs`, `ParseSecretURI`, and `SecretNameForProvider` functions
    - `GenerateSecretURI`: build `yc-lockbox://{id}/{key}` or `aws-sm://{name}/{key}`
    - `GenerateAllURIs`: generate URIs for all `RequiredEntries`
    - `ParseSecretURI`: split URI into (scheme, identifier, key) with error handling
    - `SecretNameForProvider`: return `pg-{egg}-secrets` (YC) or `polar-gosling/{egg}` (AWS)
    - _Requirements: 9.1, 9.2, 9.3, 2.4, 3.4_

  - [x] 1.4 Write property test: Secret naming convention (Property 1)
    - **Property 1: Secret naming convention is deterministic and follows provider patterns**
    - Generate random valid egg names, verify `SecretNameForProvider` output matches `pg-{eggName}-secrets` for Yandex and `polar-gosling/{eggName}` for AWS, and egg name appears verbatim
    - **Validates: Requirements 2.4, 3.4**

  - [x] 1.5 Write property test: Secret URI round-trip (Property 2)
    - **Property 2: Secret URI generation/parse round-trip**
    - Generate random (provider, identifier, key) tuples where identifier has no `://` and key has no `/`, verify `ParseSecretURI(GenerateSecretURI(...))` returns original values with correct scheme
    - **Validates: Requirements 9.1, 9.2, 9.3**

  - [x] 1.6 Write property test: Invalid inputs rejected (Property 3)
    - **Property 3: Invalid inputs are rejected before cloud API calls**
    - Generate `CreateParams` with at least one invalid field (bad provider, empty egg-name, invalid chars, missing folder-id for Yandex), verify `ValidateCreateInput` returns non-nil error
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4**

  - [x] 1.7 Write property test: Valid inputs pass validation (Property 4)
    - **Property 4: Valid inputs pass validation**
    - Generate fully valid `CreateParams` (valid provider, valid egg-name chars, folder-id present for Yandex), verify `ValidateCreateInput` returns nil
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5**

  - [x] 1.8 Write unit tests for URI helpers and validation edge cases
    - Test `ParseSecretURI` with malformed URIs (missing `://`, missing key, empty identifier)
    - Test `IsValidEggName` with empty string, special characters, valid names
    - Test `GenerateAllURIs` returns exactly `len(RequiredEntries)` entries
    - _Requirements: 4.1, 4.2, 4.3, 9.1, 9.2, 9.3_

- [x] 2. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. Implement cloud provider SecretStore backends
  - [x] 3.1 Create `internal/lockbox/yc_lockbox.go` implementing `YCLockboxStore`
    - Implement `Create`: call `sdk.Lockbox().Secret().Create()` with name `pg-{egg}-secrets`, labels `polar-gosling: true` + `egg-name: {egg}`, and three empty-string text entries for `RequiredEntries`; check for existing secret first and return descriptive error if duplicate
    - Implement `List`: call `sdk.Lockbox().Secret().List()` with folder filter, filter results by `polar-gosling: true` label, map to `[]SecretInfo`
    - Implement `Verify`: call `sdk.Lockbox().Payload().Get()` to retrieve entry keys, partition into present/missing against `RequiredEntries`
    - Follow SDK patterns from `internal/deployer/yandex_client.go`
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 5.1, 5.3, 6.1, 6.3, 6.4, 6.5, 6.6_

  - [x] 3.2 Create `internal/lockbox/aws_sm.go` implementing `AWSSecretsManagerStore`
    - Implement `Create`: call `secretsmanager.CreateSecret()` with name `polar-gosling/{egg}`, JSON string value with three empty-string keys, tags `polar-gosling: true` + `egg-name: {egg}`; check for existing secret first and return descriptive error if duplicate
    - Implement `List`: call `secretsmanager.ListSecrets()` with tag filter `polar-gosling: true`, map to `[]SecretInfo`
    - Implement `Verify`: call `secretsmanager.GetSecretValue()`, parse JSON, partition keys into present/missing against `RequiredEntries`
    - Follow SDK patterns from `internal/deployer/aws_client.go`
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 5.2, 5.3, 6.2, 6.3, 6.4, 6.5, 6.6_

  - [x] 3.3 Write property test: Verify partitions entries correctly (Property 5)
    - **Property 5: Verify correctly partitions entries into present and missing**
    - Generate random subsets of `RequiredEntries` as present keys, build a mock payload, verify `VerifyResult` has exactly those keys in `Present` and the complement in `Missing`, with `len(Present) + len(Missing) == len(RequiredEntries)`
    - **Validates: Requirements 6.3, 6.4**

  - [x] 3.4 Write integration tests with mock SDK clients
    - Create mock implementations of YC SDK and AWS SM clients
    - Test end-to-end create → verify flow for each provider
    - Test list with mixed tagged/untagged secrets
    - Test error scenarios: API errors, duplicate secrets, secret not found
    - _Requirements: 2.7, 2.8, 3.7, 3.8, 5.4, 5.5, 6.5, 6.6_

- [x] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Implement CLI commands
  - [ ] 5.1 Create `internal/cli/lockbox.go` with parent `lockbox` command
    - Register `lockboxCmd` as subcommand of `rootCmd` with `Use: "lockbox"` and `Short: "Manage cloud secret stores for Egg configurations"`
    - Display usage help with subcommand list when run without subcommand
    - Register `create`, `list`, `verify` as child commands
    - _Requirements: 1.1, 1.2, 1.3_

  - [ ] 5.2 Create `internal/cli/lockbox_create.go` with `lockbox create` command
    - Define flags: `--provider` (required), `--egg-name` (required), `--folder-id` (YC), `--region` (AWS)
    - Call `ValidateCreateInput` before any cloud API call
    - Instantiate the correct `SecretStore` based on provider flag
    - Call `SecretStore.Create()` and print generated Secret URIs to stdout (one per line)
    - Print errors to stderr with non-zero exit code
    - Support fully non-interactive execution when all flags provided
    - _Requirements: 2.1, 2.6, 2.7, 3.1, 3.6, 3.7, 4.1, 4.2, 4.3, 4.4, 4.5, 8.1, 8.2, 8.3_

  - [ ] 5.3 Create `internal/cli/lockbox_list.go` with `lockbox list` command
    - Define flags: `--provider` (required), `--folder-id` (YC), `--region` (AWS)
    - Call `SecretStore.List()` and display name, ID/ARN, egg-name, creation date for each result
    - Print "no Polar Gosling secret stores found" when list is empty
    - Print errors to stderr with non-zero exit code
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [ ] 5.4 Create `internal/cli/lockbox_verify.go` with `lockbox verify` command
    - Define flags: `--provider` (required), `--secret-id` (YC), `--secret-name` (AWS), `--folder-id` (YC), `--region` (AWS)
    - Call `SecretStore.Verify()` and print success message listing found entries (exit 0) or present/missing report (exit non-zero)
    - Handle secret-not-found with descriptive error
    - Print errors to stderr with non-zero exit code
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6_

  - [ ]* 5.5 Write unit tests for CLI commands
    - Test command registration: `lockbox` is subcommand of root with `create`, `list`, `verify` children
    - Test help output: `gosling lockbox` shows subcommand list
    - Test flag validation: missing required flags produce errors
    - Test error output goes to stderr
    - _Requirements: 1.1, 1.2, 1.3, 8.1, 8.2_

- [ ] 6. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Integrate lockbox into `gosling add egg --interactive`
  - [ ] 7.1 Modify `internal/cli/add.go` to add interactive secret store flow
    - After egg name/provider/region collection, prompt: "Does a secret store already exist? [y/n]"
    - If yes: prompt for secret ID (YC) or secret name (AWS), call `GenerateAllURIs` to build URIs
    - If no: prompt "Create one now? [y/n]"; if yes, call `SecretStore.Create()` programmatically and use returned URIs; if no, use placeholder URIs with TODO comments
    - Pass real or placeholder URIs to config.fly generation
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [ ] 7.2 Update `generateEggConfig` in `add.go` to use correct secret attribute names
    - Replace current `gitlab.token_secret` with `gitlab_token_secret`, `gitlab_webhook_secret`, and `git_repo_url_secret` as top-level egg block attributes
    - When real URIs provided: emit attributes without TODO comments
    - When placeholder URIs used: emit TODO comments above each secret attribute
    - _Requirements: 7.6, 10.1, 10.2, 10.3_

  - [ ]* 7.3 Write property test: Config.fly includes all secret attributes (Property 6)
    - **Property 6: Config.fly generation includes all secret attributes**
    - Generate random valid provider/identifier pairs, verify generated config contains `gitlab_token_secret`, `gitlab_webhook_secret`, and `git_repo_url_secret` each with correctly formatted Secret URI
    - **Validates: Requirements 7.6, 10.1**

  - [ ]* 7.4 Write unit tests for interactive flow
    - Test interactive flow with simulated stdin: user says secret exists → prompts for ID → generates URIs
    - Test interactive flow: user says no secret → declines creation → placeholder URIs with TODOs
    - Test non-interactive mode: no stdin reads when all flags provided
    - _Requirements: 7.1, 7.2, 7.5, 8.2_

- [ ] 8. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests use `github.com/leanovate/gopter` consistent with existing project tests
- All property tests run minimum 100 iterations
- All new code goes in `dev-new-features/internal/` per workspace conventions
