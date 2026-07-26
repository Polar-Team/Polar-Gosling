// Package awsstore implements bootstrap.MGSecretStore for AWS Secrets Manager.
package awsstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"

	"github.com/polar-gosling/gosling/internal/bootstrap"
)

const (
	// SecretName is the fixed name for the MG API secret in AWS Secrets Manager.
	SecretName = "polar-gosling/mothergoose"

	// TagKeyPolarGosling is the tag key identifying Polar Gosling resources.
	TagKeyPolarGosling = "polar-gosling"
	// TagKeyResourceType is the tag key identifying the resource type.
	TagKeyResourceType = "resource-type"
	// TagValueTrue is the tag value for the polar-gosling tag.
	TagValueTrue = "true"
	// TagValueMothergooseAPI is the tag value for the resource-type tag.
	TagValueMothergooseAPI = "mothergoose-api"

	// JSONKeyAPIURL is the JSON key for the API URL entry.
	JSONKeyAPIURL = "api-url"
	// JSONKeyAPIKey is the JSON key for the API key entry.
	JSONKeyAPIKey = "api-key"

	// SecretURIScheme is the URI scheme prefix for AWS Secrets Manager secrets.
	SecretURIScheme = "aws-sm://"
)

// AWSMGSecretStore implements bootstrap.MGSecretStore for AWS Secrets Manager.
type AWSMGSecretStore struct {
	client *secretsmanager.Client
	region string
}

// New creates a new AWSMGSecretStore with the given client and region.
func New(client *secretsmanager.Client, region string) *AWSMGSecretStore {
	return &AWSMGSecretStore{
		client: client,
		region: region,
	}
}

// Discover searches for an existing MG API secret in AWS Secrets Manager.
// Returns nil, nil if no secret exists.
// Returns bootstrap.ErrSecretDeleted if the secret is scheduled for deletion.
func (s *AWSMGSecretStore) Discover(ctx context.Context) (*bootstrap.MGSecret, error) {
	describeOut, err := s.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(SecretName),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ResourceNotFoundException" {
			return nil, nil
		}
		return nil, fmt.Errorf("discovering MG API secret (requires secretsmanager:ListSecrets): %w", err)
	}

	// Check if the secret is scheduled for deletion
	if describeOut.DeletedDate != nil {
		return nil, bootstrap.ErrSecretDeleted
	}

	// Get the secret value
	getOut, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(SecretName),
	})
	if err != nil {
		return nil, fmt.Errorf("reading MG API secret value: %w", err)
	}

	// Parse JSON secret string
	var secretData map[string]string
	if err := json.Unmarshal([]byte(aws.ToString(getOut.SecretString)), &secretData); err != nil {
		return nil, fmt.Errorf("parsing MG API secret JSON: %w", err)
	}

	apiURL := secretData[JSONKeyAPIURL]
	apiKey := secretData[JSONKeyAPIKey]

	return &bootstrap.MGSecret{
		ID: aws.ToString(describeOut.ARN),
		Credentials: bootstrap.Credentials{
			APIURL: apiURL,
			APIKey: apiKey,
		},
		Populated: apiURL != "" && apiKey != "",
	}, nil
}

// Create provisions a new MG API secret in AWS Secrets Manager with empty placeholder entries.
func (s *AWSMGSecretStore) Create(ctx context.Context) (*bootstrap.MGSecret, error) {
	// Build empty placeholder JSON
	secretValue := map[string]string{
		JSONKeyAPIURL: "",
		JSONKeyAPIKey: "",
	}
	jsonBytes, err := json.Marshal(secretValue)
	if err != nil {
		return nil, fmt.Errorf("marshaling MG API secret value: %w", err)
	}

	createOut, err := s.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(SecretName),
		SecretString: aws.String(string(jsonBytes)),
		Tags: []smtypes.Tag{
			{Key: aws.String(TagKeyPolarGosling), Value: aws.String(TagValueTrue)},
			{Key: aws.String(TagKeyResourceType), Value: aws.String(TagValueMothergooseAPI)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating MG API secret (requires secretsmanager:CreateSecret): %w", err)
	}

	return &bootstrap.MGSecret{
		ID: aws.ToString(createOut.ARN),
		Credentials: bootstrap.Credentials{
			APIURL: "",
			APIKey: "",
		},
		Populated: false,
	}, nil
}

// Update populates the MG API secret with real credential values.
func (s *AWSMGSecretStore) Update(ctx context.Context, secretID string, creds bootstrap.Credentials) error {
	secretValue := map[string]string{
		JSONKeyAPIURL: creds.APIURL,
		JSONKeyAPIKey: creds.APIKey,
	}
	jsonBytes, err := json.Marshal(secretValue)
	if err != nil {
		return fmt.Errorf("marshaling MG API credentials: %w", err)
	}

	_, err = s.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretID),
		SecretString: aws.String(string(jsonBytes)),
	})
	if err != nil {
		return fmt.Errorf("updating MG API secret (requires secretsmanager:PutSecretValue): %w", err)
	}

	return nil
}

// SecretURI returns the full URI for the MG API secret using the aws-sm:// scheme.
func (s *AWSMGSecretStore) SecretURI() string {
	return SecretURIScheme + SecretName
}
