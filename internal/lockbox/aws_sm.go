package lockbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// AWSSecretsManagerStore implements SecretStore for AWS Secrets Manager.
type AWSSecretsManagerStore struct {
	client *secretsmanager.Client
}

// NewAWSSecretsManagerStore creates a new AWSSecretsManagerStore.
// If region is empty, the SDK uses the default region from environment/config.
func NewAWSSecretsManagerStore(ctx context.Context, region string) (*AWSSecretsManagerStore, error) {
	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &AWSSecretsManagerStore{
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

// NewAWSSecretsManagerStoreFromClient creates a store with a pre-configured client.
// Useful for testing with mock clients.
func NewAWSSecretsManagerStoreFromClient(client *secretsmanager.Client) *AWSSecretsManagerStore {
	return &AWSSecretsManagerStore{client: client}
}

// Create provisions a new AWS Secrets Manager secret with placeholder JSON entries.
func (s *AWSSecretsManagerStore) Create(ctx context.Context, params CreateParams) (*CreateResult, error) {
	secretName := SecretNameForProvider("aws", params.EggName)

	// Check for existing secret with the same name
	_, err := s.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	})
	if err == nil {
		return nil, fmt.Errorf(
			"secret %q already exists. Use 'gosling lockbox verify --secret-name %s' to check its entries",
			secretName, secretName,
		)
	}
	// If the error is not "resource not found", it's an unexpected error
	var notFoundErr *smtypes.ResourceNotFoundException
	if !errors.As(err, &notFoundErr) {
		return nil, fmt.Errorf("checking for existing secret: %w", err)
	}

	// Build JSON value with empty placeholder entries
	secretValue := make(map[string]string, len(RequiredEntries))
	for _, key := range RequiredEntries {
		secretValue[key] = ""
	}
	jsonBytes, err := json.Marshal(secretValue)
	if err != nil {
		return nil, fmt.Errorf("marshaling secret value: %w", err)
	}

	_, err = s.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(string(jsonBytes)),
		Tags: []smtypes.Tag{
			{Key: aws.String("polar-gosling"), Value: aws.String("true")},
			{Key: aws.String("egg-name"), Value: aws.String(params.EggName)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Secrets Manager secret: %w", err)
	}

	return &CreateResult{
		ID:   secretName,
		URIs: GenerateAllURIs("aws", secretName),
	}, nil
}

// List returns all Polar Gosling-tagged Secrets Manager secrets.
func (s *AWSSecretsManagerStore) List(ctx context.Context) ([]SecretInfo, error) {
	var results []SecretInfo
	var nextToken *string

	for {
		resp, err := s.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
			Filters: []smtypes.Filter{
				{
					Key:    smtypes.FilterNameStringTypeTagKey,
					Values: []string{"polar-gosling"},
				},
				{
					Key:    smtypes.FilterNameStringTypeTagValue,
					Values: []string{"true"},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing Secrets Manager secrets: %w", err)
		}

		for _, secret := range resp.SecretList {
			info := SecretInfo{
				Name: aws.ToString(secret.Name),
				ID:   aws.ToString(secret.ARN),
			}
			for _, tag := range secret.Tags {
				if aws.ToString(tag.Key) == "egg-name" {
					info.EggName = aws.ToString(tag.Value)
					break
				}
			}
			if secret.CreatedDate != nil {
				info.CreatedAt = *secret.CreatedDate
			}
			results = append(results, info)
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	return results, nil
}

// Verify checks that a Secrets Manager secret exists and its JSON contains all RequiredEntries.
func (s *AWSSecretsManagerStore) Verify(ctx context.Context, secretRef string) (*VerifyResult, error) {
	resp, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretRef),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving secret %q: %w", secretRef, err)
	}

	var secretData map[string]interface{}
	if err := json.Unmarshal([]byte(aws.ToString(resp.SecretString)), &secretData); err != nil {
		return nil, fmt.Errorf("parsing secret JSON for %q: %w", secretRef, err)
	}

	result := &VerifyResult{}
	for _, key := range RequiredEntries {
		if _, ok := secretData[key]; ok {
			result.Present = append(result.Present, key)
		} else {
			result.Missing = append(result.Missing, key)
		}
	}

	return result, nil
}
