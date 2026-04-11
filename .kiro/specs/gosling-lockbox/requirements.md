# Requirements Document

## Introduction

The `gosling lockbox` command family adds cloud secret store management to the Gosling CLI. Currently, `gosling add egg` generates placeholder secret URIs (e.g., `yc-lockbox://gitlab-tokens/my-app-runner-token`) in the egg's `config.fly`, but the actual Yandex Cloud Lockbox or AWS Secrets Manager instance does not exist. Users must manually create these secret stores and populate them with the required entries (runner-token, webhook-secret, repo-url).

This feature introduces three subcommands — `create`, `list`, and `verify` — under `gosling lockbox`, and integrates secret store provisioning into the `gosling add egg` interactive workflow. The goal is to make secret store setup a seamless, automated part of egg creation rather than a manual prerequisite.

## Glossary

- **Lockbox_Command**: The `gosling lockbox` parent Cobra command that groups secret store management subcommands.
- **Secret_Store**: A cloud-managed secret storage instance — either a Yandex Cloud Lockbox secret or an AWS Secrets Manager secret.
- **Secret_Entry**: A single key-value pair within a Secret_Store (e.g., key `runner-token` with a placeholder or user-supplied value).
- **Required_Entries**: The minimum set of Secret_Entry keys every egg needs: `runner-token`, `webhook-secret`, `repo-url`.
- **Secret_URI**: A URI string referencing a specific Secret_Entry, following the format `yc-lockbox://{secret_id}/{key}` or `aws-sm://{secret_name}/{key}`.
- **Lockbox_Create_Command**: The `gosling lockbox create` subcommand that provisions a new Secret_Store with Required_Entries.
- **Lockbox_List_Command**: The `gosling lockbox list` subcommand that lists existing Secret_Stores tagged for Polar Gosling use.
- **Lockbox_Verify_Command**: The `gosling lockbox verify` subcommand that checks whether a referenced Secret_Store exists and contains Required_Entries.
- **Egg_Add_Integration**: The enhanced `gosling add egg --interactive` flow that incorporates secret store setup.
- **Cloud_Provider**: The target cloud platform, either `yandex` (Yandex Cloud) or `aws` (AWS).
- **Nest_Root**: The root directory of a Nest repository containing `Eggs/`, `Jobs/`, `UF/`, and `MG/` directories.
- **Gosling_Tag**: A metadata label (`polar-gosling: true` and `egg-name: {name}`) applied to Secret_Stores for identification and filtering.

## Requirements

### Requirement 1: Lockbox Parent Command Registration

**User Story:** As a Gosling CLI user, I want a `gosling lockbox` parent command, so that I can access secret store management subcommands in a consistent command hierarchy.

#### Acceptance Criteria

1. THE Lockbox_Command SHALL register as a subcommand of the root `gosling` command with `Use: "lockbox"`.
2. THE Lockbox_Command SHALL display a short description of "Manage cloud secret stores for Egg configurations" when listed in help output.
3. WHEN the user runs `gosling lockbox` without a subcommand, THE Lockbox_Command SHALL display usage help listing all available subcommands (create, list, verify).

### Requirement 2: Create a Yandex Cloud Lockbox Secret

**User Story:** As a Gosling CLI user deploying to Yandex Cloud, I want to create a Lockbox secret with the required entries for an egg, so that the egg's config.fly can reference real secret URIs.

#### Acceptance Criteria

1. WHEN the user runs `gosling lockbox create` with `--provider yandex`, `--folder-id`, and `--egg-name`, THE Lockbox_Create_Command SHALL create a Yandex Cloud Lockbox secret in the specified folder using the Yandex Cloud Go SDK.
2. THE Lockbox_Create_Command SHALL create the Lockbox secret with all three Required_Entries as text entries: `runner-token`, `webhook-secret`, `repo-url`.
3. THE Lockbox_Create_Command SHALL set each Required_Entry value to an empty string as a placeholder upon creation.
4. THE Lockbox_Create_Command SHALL name the Lockbox secret using the pattern `pg-{egg-name}-secrets` (e.g., `pg-my-app-secrets`).
5. THE Lockbox_Create_Command SHALL apply Gosling_Tags (`polar-gosling: true`, `egg-name: {egg-name}`) as labels on the created Lockbox secret.
6. WHEN the Lockbox secret is created successfully, THE Lockbox_Create_Command SHALL print the generated Secret_URIs for all three Required_Entries to stdout (e.g., `yc-lockbox://{secret_id}/runner-token`).
7. IF the Yandex Cloud API returns an error during secret creation, THEN THE Lockbox_Create_Command SHALL print the error message to stderr and exit with a non-zero exit code.
8. IF a Lockbox secret with the name `pg-{egg-name}-secrets` already exists in the specified folder, THEN THE Lockbox_Create_Command SHALL return an error indicating the secret already exists and suggest using `gosling lockbox verify` instead.

### Requirement 3: Create an AWS Secrets Manager Secret

**User Story:** As a Gosling CLI user deploying to AWS, I want to create a Secrets Manager secret with the required JSON keys for an egg, so that the egg's config.fly can reference real secret URIs.

#### Acceptance Criteria

1. WHEN the user runs `gosling lockbox create` with `--provider aws`, `--region`, and `--egg-name`, THE Lockbox_Create_Command SHALL create an AWS Secrets Manager secret in the specified region using the AWS SDK for Go v2.
2. THE Lockbox_Create_Command SHALL create the Secrets Manager secret with a JSON value containing all three Required_Entries as keys: `runner-token`, `webhook-secret`, `repo-url`.
3. THE Lockbox_Create_Command SHALL set each Required_Entry JSON value to an empty string as a placeholder upon creation.
4. THE Lockbox_Create_Command SHALL name the Secrets Manager secret using the pattern `polar-gosling/{egg-name}` (e.g., `polar-gosling/my-app`).
5. THE Lockbox_Create_Command SHALL apply Gosling_Tags (`polar-gosling: true`, `egg-name: {egg-name}`) as resource tags on the created Secrets Manager secret.
6. WHEN the Secrets Manager secret is created successfully, THE Lockbox_Create_Command SHALL print the generated Secret_URIs for all three Required_Entries to stdout (e.g., `aws-sm://polar-gosling/my-app/runner-token`).
7. IF the AWS API returns an error during secret creation, THEN THE Lockbox_Create_Command SHALL print the error message to stderr and exit with a non-zero exit code.
8. IF a Secrets Manager secret with the name `polar-gosling/{egg-name}` already exists, THEN THE Lockbox_Create_Command SHALL return an error indicating the secret already exists and suggest using `gosling lockbox verify` instead.

### Requirement 4: Create Command Input Validation

**User Story:** As a Gosling CLI user, I want the create command to validate my inputs before making cloud API calls, so that I receive clear error messages for invalid arguments.

#### Acceptance Criteria

1. WHEN the `--provider` flag value is not `yandex` or `aws`, THE Lockbox_Create_Command SHALL return an error stating the valid provider values.
2. WHEN the `--egg-name` flag is missing or empty, THE Lockbox_Create_Command SHALL return an error stating that egg-name is required.
3. WHEN the `--egg-name` value contains characters other than alphanumeric characters, hyphens, or underscores, THE Lockbox_Create_Command SHALL return an error stating the naming constraints.
4. WHEN `--provider` is `yandex` and `--folder-id` is missing or empty, THE Lockbox_Create_Command SHALL return an error stating that folder-id is required for Yandex Cloud.
5. WHEN `--provider` is `aws` and `--region` is missing or empty, THE Lockbox_Create_Command SHALL use the default region from the AWS SDK configuration.

### Requirement 5: List Secret Stores

**User Story:** As a Gosling CLI user, I want to list existing secret stores tagged for Polar Gosling use, so that I can see which eggs already have secret stores provisioned.

#### Acceptance Criteria

1. WHEN the user runs `gosling lockbox list` with `--provider yandex` and `--folder-id`, THE Lockbox_List_Command SHALL list all Yandex Cloud Lockbox secrets in the specified folder that have the label `polar-gosling: true`.
2. WHEN the user runs `gosling lockbox list` with `--provider aws` and optionally `--region`, THE Lockbox_List_Command SHALL list all AWS Secrets Manager secrets that have the tag `polar-gosling: true`.
3. THE Lockbox_List_Command SHALL display each Secret_Store with its name, secret ID or ARN, associated egg-name label, and creation date.
4. WHEN no Secret_Stores with Gosling_Tags are found, THE Lockbox_List_Command SHALL print a message indicating no Polar Gosling secret stores were found.
5. IF the cloud API returns an error during listing, THEN THE Lockbox_List_Command SHALL print the error message to stderr and exit with a non-zero exit code.

### Requirement 6: Verify Secret Store

**User Story:** As a Gosling CLI user, I want to verify that a referenced secret store exists and has the required keys, so that I can confirm my egg configuration will work before deploying.

#### Acceptance Criteria

1. WHEN the user runs `gosling lockbox verify` with `--provider yandex`, `--folder-id`, and `--secret-id`, THE Lockbox_Verify_Command SHALL check that the specified Yandex Cloud Lockbox secret exists and contains all three Required_Entries.
2. WHEN the user runs `gosling lockbox verify` with `--provider aws`, `--region`, and `--secret-name`, THE Lockbox_Verify_Command SHALL check that the specified AWS Secrets Manager secret exists and its JSON value contains all three Required_Entries as keys.
3. WHEN all Required_Entries are present, THE Lockbox_Verify_Command SHALL print a success message listing each found entry and exit with code 0.
4. WHEN one or more Required_Entries are missing, THE Lockbox_Verify_Command SHALL print a report listing present and missing entries and exit with a non-zero exit code.
5. IF the referenced Secret_Store does not exist, THEN THE Lockbox_Verify_Command SHALL print an error indicating the secret was not found and exit with a non-zero exit code.
6. IF the cloud API returns an error during verification, THEN THE Lockbox_Verify_Command SHALL print the error message to stderr and exit with a non-zero exit code.

### Requirement 7: Integration with Egg Add Interactive Mode

**User Story:** As a Gosling CLI user creating a new egg interactively, I want the add egg command to guide me through secret store setup, so that my egg's config.fly references real secret URIs from the start.

#### Acceptance Criteria

1. WHEN the user runs `gosling add egg <name> --interactive`, THE Egg_Add_Integration SHALL prompt the user asking whether a Secret_Store already exists for the egg's cloud provider.
2. WHEN the user indicates a Secret_Store already exists, THE Egg_Add_Integration SHALL prompt for the existing secret ID (Yandex Cloud) or secret name (AWS) and generate the correct Secret_URIs in the egg's config.fly.
3. WHEN the user indicates no Secret_Store exists, THE Egg_Add_Integration SHALL offer to create one by invoking the Lockbox_Create_Command flow programmatically.
4. WHEN the Lockbox_Create_Command flow completes successfully during interactive egg creation, THE Egg_Add_Integration SHALL use the returned secret ID or name to generate the correct Secret_URIs in the egg's config.fly.
5. WHEN the user declines to create a Secret_Store during interactive egg creation, THE Egg_Add_Integration SHALL generate the config.fly with placeholder Secret_URIs and print a reminder to set up secrets before deploying.
6. THE Egg_Add_Integration SHALL generate `gitlab_token_secret`, `gitlab_webhook_secret`, and `git_repo_url_secret` attributes in the egg's config.fly using the correct Secret_URI format for the chosen Cloud_Provider.

### Requirement 8: Non-Interactive Create Mode

**User Story:** As a CI/CD pipeline operator, I want to create secret stores non-interactively using flags, so that I can automate secret store provisioning in scripts.

#### Acceptance Criteria

1. THE Lockbox_Create_Command SHALL support fully non-interactive execution when all required flags are provided (`--provider`, `--egg-name`, and provider-specific flags).
2. WHEN running non-interactively, THE Lockbox_Create_Command SHALL not prompt for any user input.
3. WHEN running non-interactively and the secret is created successfully, THE Lockbox_Create_Command SHALL output the secret ID or name and the three Secret_URIs to stdout, one per line.

### Requirement 9: Secret URI Generation

**User Story:** As a Gosling CLI user, I want generated secret URIs to follow the established URI scheme conventions, so that MotherGoose SecretManager can resolve them at runtime.

#### Acceptance Criteria

1. WHEN the Cloud_Provider is `yandex`, THE Lockbox_Create_Command SHALL generate Secret_URIs in the format `yc-lockbox://{secret_id}/{key}` where `secret_id` is the Lockbox secret UUID returned by the API and `key` is the entry name.
2. WHEN the Cloud_Provider is `aws`, THE Lockbox_Create_Command SHALL generate Secret_URIs in the format `aws-sm://{secret_name}/{key}` where `secret_name` is the Secrets Manager secret name and `key` is the JSON key name.
3. FOR ALL generated Secret_URIs, parsing the URI by splitting on `://` and then `/` SHALL yield exactly three components: scheme, identifier, and key name.

### Requirement 10: Config.fly Template Update

**User Story:** As a Gosling CLI user, I want the generated egg config.fly to use the correct secret attribute names matching the .fly schema, so that `gosling validate` accepts the configuration.

#### Acceptance Criteria

1. THE Egg_Add_Integration SHALL generate egg config.fly files with `gitlab_token_secret`, `gitlab_webhook_secret`, and `git_repo_url_secret` as top-level attributes inside the egg block, matching the .fly egg schema.
2. WHEN a real Secret_Store is referenced, THE Egg_Add_Integration SHALL omit TODO comments from the secret attributes in the generated config.fly.
3. WHEN placeholder Secret_URIs are used, THE Egg_Add_Integration SHALL include a TODO comment above each secret attribute indicating the user must configure the secret store.
