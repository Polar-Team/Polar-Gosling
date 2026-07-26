// Package bootstrap orchestrates MG API credential discovery, creation, and population
// during the deploy command's Bootstrap_Phase.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/polar-gosling/gosling/internal/deployer"
)

// ErrSecretDeleted is returned when a secret is in scheduled-deletion state
// and cannot be used or updated.
var ErrSecretDeleted = errors.New("secret is scheduled for deletion")

// MGSecretStore abstracts cloud-specific MG API secret operations.
// Implementations: YCMGSecretStore (Yandex Cloud Lockbox), AWSMGSecretStore (AWS Secrets Manager).
type MGSecretStore interface {
	// Discover searches for an existing MG API secret.
	// Returns nil, nil if no secret exists.
	// Returns ErrSecretDeleted if secret is in scheduled-deletion state.
	Discover(ctx context.Context) (*MGSecret, error)

	// Create provisions a new MG API secret with empty placeholder entries.
	Create(ctx context.Context) (*MGSecret, error)

	// Update populates the MG API secret with real credential values.
	Update(ctx context.Context, secretID string, creds Credentials) error
}

// BackendDeployer abstracts backend infrastructure deployment.
// The existing deployer.Deployer satisfies this via an adapter.
type BackendDeployer interface {
	// DeployBackend deploys MotherGoose infrastructure and returns API credentials.
	DeployBackend(ctx context.Context, mgCfg *deployer.MGConfig, ufCfg *deployer.UFConfig) (*DeployResult, error)
}

// SecretBootstrapper orchestrates MG API credential discovery, creation, and population.
type SecretBootstrapper struct {
	store    MGSecretStore
	deployer BackendDeployer
}

// NewSecretBootstrapper creates a new SecretBootstrapper with the given store and deployer.
func NewSecretBootstrapper(store MGSecretStore, deployer BackendDeployer) *SecretBootstrapper {
	return &SecretBootstrapper{
		store:    store,
		deployer: deployer,
	}
}

// MGSecret represents a discovered or created MG API secret.
type MGSecret struct {
	ID          string      // Secret UUID (YC) or secret name (AWS)
	Credentials Credentials // May have empty values if not yet populated
	Populated   bool        // True if both api-url and api-key are non-empty
}

// Credentials holds the MG API access values.
type Credentials struct {
	APIURL string
	APIKey string
}

// DeployResult holds outputs from backend infrastructure deployment.
type DeployResult struct {
	APIGatewayURL string // HTTPS URL of the deployed API Gateway
	APIKey        string // Generated API key for authentication
}

// BootstrapResult is the output of a successful bootstrap operation.
type BootstrapResult struct {
	APIURL        string
	APIKey        string
	SecretID      string
	SecretURI     string // Full URI (yc-lockbox://... or aws-sm://...)
	WasDiscovered bool   // True if credentials came from existing populated secret
}

// Bootstrap executes the full bootstrap workflow: discover/create secret,
// deploy backend infrastructure, validate results, and update secret with credentials.
func (b *SecretBootstrapper) Bootstrap(ctx context.Context, mgCfg *deployer.MGConfig, ufCfg *deployer.UFConfig) (*BootstrapResult, error) {
	// Phase 1: Discover existing secret with 30-second deadline.
	discoverCtx, discoverCancel := context.WithTimeout(ctx, 30*time.Second)
	defer discoverCancel()

	secret, err := b.store.Discover(discoverCtx)
	if err != nil && !errors.Is(err, ErrSecretDeleted) {
		return nil, fmt.Errorf("failed to discover MG API secret: %w", err)
	}

	// If secret is in scheduled-deletion state without valid credentials, treat as not found.
	deletedSecret := errors.Is(err, ErrSecretDeleted)

	// Path determination based on discovery result.
	switch {
	case secret != nil && secret.Populated && !deletedSecret:
		// Secret found with populated credentials — return immediately.
		return &BootstrapResult{
			APIURL:        secret.Credentials.APIURL,
			APIKey:        secret.Credentials.APIKey,
			SecretID:      secret.ID,
			WasDiscovered: true,
		}, nil

	case secret != nil && !secret.Populated && !deletedSecret:
		// Secret found with empty credentials — deploy and update.
		deployResult, err := b.deployer.DeployBackend(ctx, mgCfg, ufCfg)
		if err != nil {
			return nil, fmt.Errorf("backend deployment failed: %w", err)
		}

		if err := validateDeployResult(deployResult); err != nil {
			return nil, err
		}

		updateCtx, updateCancel := context.WithTimeout(ctx, 30*time.Second)
		defer updateCancel()

		creds := Credentials{
			APIURL: deployResult.APIGatewayURL,
			APIKey: deployResult.APIKey,
		}
		if err := b.store.Update(updateCtx, secret.ID, creds); err != nil {
			return nil, fmt.Errorf("failed to update MG API secret: %w", err)
		}

		return &BootstrapResult{
			APIURL:        deployResult.APIGatewayURL,
			APIKey:        deployResult.APIKey,
			SecretID:      secret.ID,
			WasDiscovered: false,
		}, nil

	default:
		// Secret not found (or deleted) — create, deploy, and update.
		createCtx, createCancel := context.WithTimeout(ctx, 30*time.Second)
		defer createCancel()

		newSecret, err := b.store.Create(createCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to create MG API secret: %w", err)
		}

		deployResult, err := b.deployer.DeployBackend(ctx, mgCfg, ufCfg)
		if err != nil {
			return nil, fmt.Errorf("backend deployment failed: %w", err)
		}

		if err := validateDeployResult(deployResult); err != nil {
			return nil, err
		}

		updateCtx, updateCancel := context.WithTimeout(ctx, 30*time.Second)
		defer updateCancel()

		creds := Credentials{
			APIURL: deployResult.APIGatewayURL,
			APIKey: deployResult.APIKey,
		}
		if err := b.store.Update(updateCtx, newSecret.ID, creds); err != nil {
			return nil, fmt.Errorf("failed to update MG API secret: %w", err)
		}

		return &BootstrapResult{
			APIURL:        deployResult.APIGatewayURL,
			APIKey:        deployResult.APIKey,
			SecretID:      newSecret.ID,
			WasDiscovered: false,
		}, nil
	}
}

// validateDeployResult checks that the deployment result contains a valid HTTPS URL
// and a non-empty API key.
func validateDeployResult(result *DeployResult) error {
	if strings.TrimSpace(result.APIKey) == "" {
		return fmt.Errorf("deployment returned empty API key")
	}

	parsed, err := url.Parse(result.APIGatewayURL)
	if err != nil {
		return fmt.Errorf("deployment returned invalid API Gateway URL: %s", result.APIGatewayURL)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("deployment returned invalid API Gateway URL: %s", result.APIGatewayURL)
	}
	if parsed.Host == "" {
		return fmt.Errorf("deployment returned invalid API Gateway URL: %s", result.APIGatewayURL)
	}

	return nil
}
