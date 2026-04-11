# Implementation Plan: gosling-init-upstream

## Overview

Enhance `gosling init` with Git repository initialization, interactive/flag-based upstream remote configuration, and updated .fly templates. All changes are scoped to `Polar-Gosling/dev-new-features/internal/cli/init.go` and its test file. Implementation proceeds incrementally: flags and helpers first, then git init, then upstream remote logic, then template updates, and finally updated output messaging.

## Tasks

- [ ] 1. Add CLI flags and helper functions
  - [ ] 1.1 Add `--remote-name`, `--remote-url`, and `--branch` flag variables and register them on `initCmd`
    - Add `remoteName`, `remoteURL`, `branchName` package-level vars
    - Register `StringVar` flags in `init()` with defaults: `remote-name`=`"main"`, `remote-url`=`""`, `branch`=`"main"`
    - _Requirements: 3.1, 3.2, 3.3_

  - [ ] 1.2 Implement `isTerminal()` helper function
    - Use `os.Stdin.Stat()` and check for `os.ModeCharDevice` to detect terminal stdin
    - Return `bool`
    - _Requirements: 3.5, 3.6_

  - [ ] 1.3 Implement `promptWithDefault(prompt, defaultVal string) string` helper function
    - Use `bufio.Scanner` to read a line from `os.Stdin`
    - If input is empty or whitespace-only, return `defaultVal`
    - Otherwise return trimmed input
    - _Requirements: 2.1, 2.2_

  - [ ] 1.4 Implement `addGitRemote(dir, name, url string) error` helper function
    - Execute `git remote add <name> <url>` via `os/exec.Command` with `Dir` set to `dir`
    - Wrap errors with `fmt.Errorf("failed to add upstream remote: %w", err)`
    - _Requirements: 2.5, 2.8_

  - [ ]* 1.5 Write property test for `promptWithDefault` — Property 2: Prompt default value preservation
    - **Property 2: Prompt default value preservation**
    - For any non-empty default value and empty/whitespace-only user input, `promptWithDefault` returns the original default unchanged
    - Use `gen.AnyString()` for defaults filtered to non-empty, `genWhitespaceString()` for input
    - Pipe simulated stdin via `os.Pipe()` or `strings.NewReader`
    - **Validates: Requirements 2.2**

- [ ] 2. Implement Git repository initialization
  - [ ] 2.1 Implement `initGitRepo(dir string) error` helper function
    - Execute `git init` via `os/exec.Command` with `Dir` set to `dir`
    - Wrap errors with `fmt.Errorf("failed to initialize git repository: %w", err)`
    - _Requirements: 1.1, 1.3_

  - [ ] 2.2 Integrate `initGitRepo` into `runInit` after file creation
    - Call `initGitRepo(absPath)` after writing `.gitignore`
    - Print confirmation message on success: `"  ✓ Initialized Git repository"`
    - Return error immediately on failure (do not proceed to remote config)
    - _Requirements: 1.1, 1.2, 1.3_

  - [ ]* 2.3 Write property test for `initGitRepo` — Property 1: Git initialization creates a valid repository
    - **Property 1: Git initialization creates a valid repository**
    - For any valid directory path name, `initGitRepo` results in a `.git` directory existing within the target path
    - Use `genValidPathName()` generator (already exists in test file)
    - Create temp dir, run `initGitRepo`, assert `.git` directory exists
    - **Validates: Requirements 1.1**

- [ ] 3. Implement upstream remote configuration
  - [ ] 3.1 Implement `configureUpstreamRemote(dir string) error` orchestration function
    - If `remoteURL` flag is set (non-empty): use flag values directly, call `addGitRemote`, skip prompts
    - Else if `isTerminal()`: run interactive prompts for remote name (default `"main"`), URL, and branch (default `"main"`)
    - Else (non-terminal, no flag): skip remote config silently
    - If interactive URL is empty: skip remote config, print skip message
    - On `addGitRemote` success: print confirmation with remote name, URL, and branch
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 3.4, 3.5, 3.6_

  - [ ] 3.2 Integrate `configureUpstreamRemote` into `runInit` after `initGitRepo`
    - Call `configureUpstreamRemote(absPath)` only when `initGitRepo` succeeds
    - _Requirements: 1.3, 2.1_

  - [ ]* 3.3 Write property test for `addGitRemote` — Property 3: Git remote registration round-trip
    - **Property 3: Git remote registration round-trip**
    - For any valid remote name and valid remote URL, calling `addGitRemote` then querying `git remote -v` shows the remote mapped to the URL
    - Use `genValidRemoteName()` and `genValidRemoteURL()` generators
    - Create temp dir, run `git init`, call `addGitRemote`, verify with `git remote -v`
    - **Validates: Requirements 2.5, 3.4**

  - [ ]* 3.4 Write unit tests for upstream remote configuration
    - `TestFlagSkipsPrompts`: verify `--remote-url` flag bypasses interactive prompts (_Requirements: 3.4, 3.5_)
    - `TestEmptyURLSkipsRemoteConfig`: verify empty URL in interactive mode prints skip message (_Requirements: 2.7_)
    - `TestGitRemoteAddFailure`: verify error wrapping on `git remote add` failure (_Requirements: 2.8_)
    - `TestNonTerminalSkipsPrompts`: verify non-terminal stdin skips remote config (_Requirements: 3.6_)

- [ ] 4. Checkpoint
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Update .fly templates
  - [ ] 5.1 Update `defaultUglyFoxConfig()` to return corrected .fly syntax
    - Add block label: `uglyfox "default" {`
    - Add `mothergoose = "default"` attribute
    - Add `cpu_threshold` and `memory_threshold` to `apex` block
    - Add `policies` block with example `rule "terminate_old_failed"` sub-block
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [ ] 5.2 Update `defaultMotherGooseConfig()` to return corrected .fly syntax
    - Add block label: `mothergoose "default" {`
    - Add `cloud` block with `provider`, `yc_folder_id`, `yc_cloud_id`
    - Update `fastapi_app` with `image`, `cores`, `execution_timeout`, `concurrency`, `service_account`
    - Update `celery_workers` with `cores`, `execution_timeout`, `concurrency`
    - Replace `triggers` with `git_sync_trigger`
    - Replace `message_queues` with `mothergoose_queues` containing `task_queue` and `dlq`
    - Replace `service_accounts` wrapper with labeled `service_account "name"` blocks
    - Replace `mode = "serverless"` with `serverless_mode = true` in `database`
    - Remove `uglyfox_workers` block
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9_

  - [ ]* 5.3 Write smoke tests for UglyFox template
    - `TestUglyFoxTemplateHasBlockLabel`: assert `uglyfox "default" {` present (_Requirements: 4.1_)
    - `TestUglyFoxTemplateHasMothergooseAttr`: assert `mothergoose = "default"` present (_Requirements: 4.2_)
    - `TestUglyFoxTemplateHasThresholds`: assert `cpu_threshold` and `memory_threshold` in output (_Requirements: 4.3_)
    - `TestUglyFoxTemplateHasPolicies`: assert `policies` block with `rule` sub-block (_Requirements: 4.4_)

  - [ ]* 5.4 Write smoke tests for MotherGoose template
    - `TestMotherGooseTemplateHasBlockLabel`: assert `mothergoose "default" {` present (_Requirements: 5.1_)
    - `TestMotherGooseTemplateHasCloudBlock`: assert `cloud` block with `provider`, `yc_folder_id`, `yc_cloud_id` (_Requirements: 5.2_)
    - `TestMotherGooseTemplateHasFastapiAttrs`: assert `image`, `cores`, `execution_timeout`, `concurrency`, `service_account` (_Requirements: 5.3_)
    - `TestMotherGooseTemplateHasCeleryAttrs`: assert `cores`, `execution_timeout`, `concurrency` (_Requirements: 5.4_)
    - `TestMotherGooseTemplateUsesGitSyncTrigger`: assert `git_sync_trigger`, not `triggers` (_Requirements: 5.5_)
    - `TestMotherGooseTemplateUsesMothergooseQueues`: assert `mothergoose_queues` with `task_queue` + `dlq` (_Requirements: 5.6_)
    - `TestMotherGooseTemplateUsesLabeledServiceAccounts`: assert `service_account "mg-sa"`, not `service_accounts` (_Requirements: 5.7_)
    - `TestMotherGooseTemplateUsesServerlessMode`: assert `serverless_mode = true`, not `mode = "serverless"` (_Requirements: 5.8_)

- [ ] 6. Update success output and next steps messaging
  - [ ] 6.1 Update `runInit` success output to reflect Git and remote setup
    - When remote configured: include remote name, URL, and suggest `git push -u <remote_name> <branch_name>`
    - When no remote configured: suggest manually adding a remote as next step
    - _Requirements: 6.1, 6.2, 6.3_

  - [ ]* 6.2 Write unit tests for success output
    - `TestSuccessOutputWithRemote`: verify output includes remote info and push command (_Requirements: 6.1, 6.2_)
    - `TestSuccessOutputWithoutRemote`: verify output suggests manual remote addition (_Requirements: 6.3_)

- [ ] 7. Final checkpoint
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- All code changes are in `Polar-Gosling/dev-new-features/internal/cli/init.go` (_Requirements: 7.1, 7.2_)
- Tests go in `Polar-Gosling/dev-new-features/internal/cli/init_test.go` (unit + smoke) and `init_property_test.go` (property tests)
- Property tests use `github.com/leanovate/gopter` (already in `go.mod`)
- Existing `genValidPathName()` generator in `init_property_test.go` can be reused for Property 1
- Each property test references a specific correctness property from the design document
