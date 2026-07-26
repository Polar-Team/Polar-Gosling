# Requirements Document

## Introduction

The `gosling deploy` command currently requires `--api-url` and `--api-key` flags to communicate with the MotherGoose API for Egg deployment. However, on a fresh Nest repository initialized with `gosling init`, no MotherGoose backend exists yet — making these values unavailable. This feature introduces a two-phase deploy workflow that automatically bootstraps MG API credentials into the cloud secret store (Yandex Cloud Lockbox or AWS Secrets Manager) during backend infrastructure deployment, eliminating the chicken-and-egg problem.

## Glossary

- **Deploy_Command**: The `gosling deploy` CLI command responsible for deploying backend infrastructure and Egg configurations
- **MG_Config**: The parsed MotherGoose configuration from `MG/*.fly` files containing cloud provider details
- **MG_API_Secret**: A dedicated secret resource in Lockbox or Secrets Manager storing `api-url` and `api-key` entries for the MotherGoose API
- **Bootstrap_Phase**: The first phase of deployment that creates the MG_API_Secret, deploys backend infrastructure, and populates the secret with real values
- **Egg_Phase**: The second phase of deployment that uses MG API credentials to deploy Egg configurations via the MotherGoose API
- **Secret_Store**: The cloud-native secret management service (YC Lockbox or AWS Secrets Manager) used to persist credentials
- **Deployer**: The backend infrastructure deployer component that provisions MotherGoose, UglyFox, databases, and queues
- **API_Gateway_URL**: The URL of the MotherGoose API Gateway endpoint produced by backend infrastructure deployment
- **API_Key**: An authentication key for the MotherGoose API generated or retrieved during backend deployment

## Requirements

### Requirement 1: Make API Credentials Optional

**User Story:** As a developer deploying a fresh Nest repository, I want the `--api-url` and `--api-key` flags to be optional, so that I can run `gosling deploy` without pre-existing MotherGoose infrastructure.

#### Acceptance Criteria

1. THE Deploy_Command SHALL accept `--api-url` and `--api-key` as optional flags that default to empty strings when omitted
2. WHEN `--api-url` and `--api-key` are both provided with non-empty values, THE Deploy_Command SHALL use the provided values for Egg_Phase without executing Bootstrap_Phase secret creation
3. WHEN `--api-url` and `--api-key` are both omitted or both empty, THE Deploy_Command SHALL execute Bootstrap_Phase to obtain credentials automatically
4. IF only one of `--api-url` or `--api-key` is provided with a non-empty value while the other is omitted or empty, THEN THE Deploy_Command SHALL validate the pair requirement first, exit with a non-zero exit code, print an error message indicating both flags must be provided together or both omitted, and SHALL NOT execute any deployment phases including Bootstrap_Phase and Egg_Phase
5. WHEN both `--api-url` and `--api-key` are provided together, IF `--api-url` contains a value that is not a valid HTTP or HTTPS URL, THEN THE Deploy_Command SHALL exit with a non-zero exit code and print an error message indicating the expected URL format

### Requirement 2: MG API Secret Discovery

**User Story:** As a developer, I want the deploy command to check for an existing MG API secret before creating a new one, so that repeated deploys do not create duplicate secrets.

#### Acceptance Criteria

1. WHEN Bootstrap_Phase begins, THE Deploy_Command SHALL read the cloud provider configuration from MG_Config including provider type, folder-id/cloud-id (Yandex Cloud) or region/account-id (AWS)
2. WHEN the cloud provider is Yandex Cloud, THE Deploy_Command SHALL search for an existing MG_API_Secret named `pg-mothergoose-secrets` in the configured folder by listing secrets and matching the exact name
3. WHEN the cloud provider is AWS, THE Deploy_Command SHALL search for an existing MG_API_Secret named `polar-gosling/mothergoose` in the configured region by describing the secret by exact name
4. WHEN an existing MG_API_Secret is found with `api-url` and `api-key` entries that each contain a non-zero-length string value, THE Deploy_Command SHALL use those values for Egg_Phase and skip secret creation and backend deployment
5. WHEN an existing MG_API_Secret is found with missing entries, zero-length `api-url` or zero-length `api-key` values, THE Deploy_Command SHALL proceed with backend deployment and update the secret after deployment succeeds
6. IF the discovered MG_API_Secret is in a scheduled-for-deletion state (Yandex Cloud) or marked as deleted in AWS Secrets Manager but contains valid non-empty `api-url` and `api-key` values, THEN THE Deploy_Command SHALL use those credential values for Egg_Phase before the secret is actually deleted
7. IF the discovered MG_API_Secret is in a scheduled-for-deletion state and does not contain valid credentials, THEN THE Deploy_Command SHALL treat it as not found and proceed with secret creation
8. IF the secret discovery API call does not respond within 30 seconds, THEN THE Deploy_Command SHALL treat the timeout as a hard failure, return an error indicating that the secret store is unreachable and the operation should be retried, and SHALL NOT use any cached or stale credential values

### Requirement 3: MG API Secret Creation

**User Story:** As a developer, I want the deploy command to create a dedicated secret for MG API credentials when none exists, so that credentials are stored securely in the cloud provider.

#### Acceptance Criteria

1. WHEN no existing MG_API_Secret is found and the cloud provider is Yandex Cloud, THE Deploy_Command SHALL create a Lockbox secret named `pg-mothergoose-secrets` in the configured folder with a first version containing two entries: `api-url` set to an empty string and `api-key` set to an empty string
2. WHEN no existing MG_API_Secret is found and the cloud provider is AWS, THE Deploy_Command SHALL create a Secrets Manager secret named `polar-gosling/mothergoose` in the configured region with a secret string containing a JSON object with two keys: `api-url` set to an empty string and `api-key` set to an empty string
3. WHEN the MG_API_Secret is created, THE Deploy_Command SHALL tag the secret with labels `polar-gosling=true` and `resource-type=mothergoose-api`
4. WHEN the MG_API_Secret is created successfully, THE Deploy_Command SHALL print the secret identifier to stdout and proceed with backend deployment
5. IF secret creation fails due to a cloud provider error, THEN THE Deploy_Command SHALL return an error message indicating the secret name that failed to create and the underlying provider error
6. IF the MG_API_Secret is created successfully but a subsequent operation (such as tagging) fails with a cloud provider error, THEN THE Deploy_Command SHALL return an error message indicating the partial failure, including the secret identifier that was created and the operation that failed

### Requirement 4: Backend Deployment and Credential Retrieval

**User Story:** As a developer, I want the deploy command to extract the API Gateway URL and API key from the backend deployment output, so that the MG API secret can be populated with real values.

#### Acceptance Criteria

1. WHEN backend infrastructure deployment succeeds, THE Deployer SHALL return the API_Gateway_URL from the deployed API Gateway resource as a valid HTTPS URL
2. WHEN backend infrastructure deployment succeeds, THE Deployer SHALL return the API_Key generated during deployment as a non-empty string
3. WHEN the Deployer returns API_Gateway_URL and API_Key, THE Deploy_Command SHALL update the MG_API_Secret with the real `api-url` and `api-key` values, and IF the MG_API_Secret update fails, THEN THE Deploy_Command SHALL treat the entire operation as failed
4. WHEN the cloud provider is Yandex Cloud, THE Deploy_Command SHALL create a new secret version with the populated `api-url` and `api-key` entries
5. WHEN the cloud provider is AWS, THE Deploy_Command SHALL update the secret string JSON with the populated `api-url` and `api-key` values
6. IF backend infrastructure deployment succeeds but does not return a valid API_Gateway_URL, THEN THE Deploy_Command SHALL return an error indicating that the API Gateway URL could not be extracted from the deployment output

### Requirement 5: Egg Deployment Using Bootstrapped Credentials

**User Story:** As a developer, I want the deploy command to seamlessly use the bootstrapped credentials for Egg deployment, so that the entire deploy workflow completes in a single invocation.

#### Acceptance Criteria

1. WHEN Bootstrap_Phase completes successfully with non-empty API_Gateway_URL and non-empty API_Key, THE Deploy_Command SHALL create a MotherGoose API client using those credentials and invoke Egg_Phase
2. WHEN MG_API_Secret already contains non-empty `api-url` and `api-key` values (discovered in Requirement 2), THE Deploy_Command SHALL create a MotherGoose API client using the discovered credentials and invoke Egg_Phase
3. WHEN the `Eggs/` directory does not exist or contains no subdirectories with a `config.fly` file, THE Deploy_Command SHALL skip Egg_Phase and complete successfully after Bootstrap_Phase
4. IF the MotherGoose API client fails to connect to the API_Gateway_URL during Egg_Phase, THEN THE Deploy_Command SHALL return an error indicating the API is unreachable and display the API_Gateway_URL that matches the credentials being used
5. IF deployment of an individual Egg configuration fails during Egg_Phase, THEN THE Deploy_Command SHALL stop processing remaining Eggs, return an error identifying the failed Egg by name, and report how many Eggs were successfully deployed before the failure

### Requirement 6: Error Handling for Secret Store Access

**User Story:** As a developer, I want clear error messages when the deploy command cannot access the cloud secret store, so that I can fix IAM permissions and retry.

#### Acceptance Criteria

1. IF the Deploy_Command cannot authenticate with the cloud provider secret store, THEN THE Deploy_Command SHALL return a non-zero exit code and an error message indicating the authentication failure, the cloud provider name, the target resource identifier (folder-id for YC, region/account for AWS), and the required IAM permissions for authentication
2. IF the Deploy_Command lacks permission to create a secret in Yandex Cloud Lockbox, THEN THE Deploy_Command SHALL return a non-zero exit code and an error specifying that the `lockbox.editor` role is required on the target folder
3. IF the Deploy_Command lacks permission to create a secret in AWS Secrets Manager, THEN THE Deploy_Command SHALL return a non-zero exit code and an error specifying that `secretsmanager:CreateSecret` and `secretsmanager:PutSecretValue` permissions are required
4. IF the Deploy_Command cannot list existing secrets to check for duplicates, THEN THE Deploy_Command SHALL return a non-zero exit code and an error specifying the required list permissions (`lockbox.viewer` for YC, `secretsmanager:ListSecrets` for AWS)
5. IF the Deploy_Command lacks permission to update an existing MG_API_Secret with new credential values, THEN THE Deploy_Command SHALL return a non-zero exit code and an error specifying the required update permissions (`lockbox.editor` for YC, `secretsmanager:PutSecretValue` for AWS)
6. IF the Deploy_Command cannot reach the cloud provider secret store endpoint within 30 seconds, THEN THE Deploy_Command SHALL return a non-zero exit code and an error message indicating the connectivity failure and the endpoint that was unreachable

### Requirement 7: Error Handling for Deployment Failures

**User Story:** As a developer, I want the deploy command to handle partial failures gracefully, so that I can retry without manual cleanup.

#### Acceptance Criteria

1. IF backend infrastructure deployment fails after the MG_API_Secret has been created, THEN THE Deploy_Command SHALL print to stderr that the secret exists with empty values and that the user can retry deployment by re-running the same command
2. IF the Deploy_Command fails to update the MG_API_Secret after successful backend deployment, THEN THE Deploy_Command SHALL print the API_Gateway_URL and API_Key to stdout and print to stderr an instruction to provide them via `--api-url` and `--api-key` flags on the next run
3. IF Egg_Phase fails after successful Bootstrap_Phase, THEN THE Deploy_Command SHALL print to stderr that backend infrastructure is deployed, print the API_Gateway_URL and API_Key to stdout, and instruct the user to retry with `--api-url` and `--api-key` flags using those values
4. IF any deployment phase fails, THEN THE Deploy_Command SHALL exit with a non-zero exit code AND SHALL NOT roll back resources created in previously completed phases; both constraints are mandatory and neither may be relaxed

### Requirement 8: Dry Run Support

**User Story:** As a developer, I want the `--dry-run` flag to preview the bootstrap secret creation and two-phase deployment plan without making changes, so that I can verify the plan before executing.

#### Acceptance Criteria

1. WHEN `--dry-run` is specified and `--api-url`/`--api-key` are omitted, THE Deploy_Command SHALL display the MG_API_Secret name that would be created (e.g., `pg-mothergoose-secrets` for Yandex Cloud or `polar-gosling/mothergoose` for AWS), the target cloud provider, and the folder-id (Yandex Cloud) or region (AWS)
2. WHEN `--dry-run` is specified and `--api-url`/`--api-key` are provided, THE Deploy_Command SHALL display the Egg_Phase deployment plan showing each Egg name, runner type, cloud provider, and region, and SHALL skip Bootstrap_Phase display
3. WHEN `--dry-run` is specified, THE Deploy_Command SHALL display the two-phase deployment plan listing Bootstrap_Phase resources (MG_API_Secret, backend infrastructure components) followed by Egg_Phase resources (each Egg configuration name and target)
4. WHEN `--dry-run` is specified, THE Deploy_Command SHALL not create, update, or delete any secrets, shall not deploy or destroy any infrastructure, and shall not make any write API calls to the cloud provider or the MotherGoose API
5. WHEN `--dry-run` is specified and plan generation succeeds, THE Deploy_Command SHALL exit with code 0 upon successfully generating and displaying the plan
6. WHEN `--dry-run` is specified and plan generation fails due to errors (e.g., unable to read MG_Config, invalid configuration), THE Deploy_Command SHALL exit with a non-zero exit code and print an error message indicating the plan generation failure

### Requirement 9: Deploy Command Output

**User Story:** As a developer, I want clear progress output during the two-phase deploy workflow, so that I understand what is happening at each stage.

#### Acceptance Criteria

1. WHEN Bootstrap_Phase begins, THE Deploy_Command SHALL print a message to stdout that includes the text "bootstrapping" and the target cloud provider name (yandex or aws)
2. WHEN the MG_API_Secret is created or discovered, THE Deploy_Command SHALL print the secret identifier and its URI (using the `yc-lockbox://` or `aws-sm://` scheme) to stdout
3. WHEN backend infrastructure deployment completes, THE Deploy_Command SHALL print the API_Gateway_URL obtained from deployment to stdout
4. WHEN the MG_API_Secret is updated with real credentials, THE Deploy_Command SHALL print a message to stdout that includes the secret identifier and indicates that the credentials were written successfully
5. WHEN Egg_Phase begins using bootstrapped credentials from the current execution's Bootstrap_Phase, THE Deploy_Command SHALL print a message to stdout indicating the credentials were bootstrapped in this run
6. WHEN Egg_Phase begins using credentials provided via `--api-url` and `--api-key` flags, THE Deploy_Command SHALL print a message to stdout indicating the credentials are user-provided
7. WHEN Egg_Phase begins using credentials discovered from an existing MG_API_Secret (without running Bootstrap_Phase in this execution), THE Deploy_Command SHALL print a message to stdout indicating the credentials were retrieved from the secret store
8. WHEN no Egg configurations exist in the `Eggs/` directory and Bootstrap_Phase completes, THE Deploy_Command SHALL print a message to stdout indicating that Egg_Phase was skipped
