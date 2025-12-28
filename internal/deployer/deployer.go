package deployer

import (
	"context"
	"fmt"
)

// Deployer is the main deployer for ecosystem infrastructure (MotherGoose, UglyFox, databases)
// Note: Individual runner deployment is handled by MotherGoose using OpenTofu, not by Gosling CLI
type Deployer struct {
	awsClient    *AWSClient
	yandexClient *YandexCloudClient
}

// NewDeployer creates a new deployer instance
func NewDeployer(ctx context.Context) (*Deployer, error) {
	return &Deployer{}, nil
}

// DeployBackendInfrastructure deploys the backend infrastructure (MotherGoose, UglyFox, databases)
func (d *Deployer) DeployBackendInfrastructure(ctx context.Context, provider CloudProvider, region string) error {
	switch provider {
	case CloudProviderAWS:
		if d.awsClient == nil {
			client, err := NewAWSClient(ctx, region)
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}
			d.awsClient = client
		}
		return d.awsClient.DeployBackendInfrastructure(ctx)

	case CloudProviderYandex:
		if d.yandexClient == nil {
			client, err := NewYandexCloudClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create Yandex Cloud client: %w", err)
			}
			d.yandexClient = client
		}
		return d.yandexClient.DeployBackendInfrastructure(ctx)

	default:
		return fmt.Errorf("unsupported cloud provider: %s", provider)
	}
}

// GetStatus retrieves the current status of infrastructure
func (d *Deployer) GetStatus(ctx context.Context, provider CloudProvider, region, resourceID string) (string, error) {
	switch provider {
	case CloudProviderAWS:
		if d.awsClient == nil {
			client, err := NewAWSClient(ctx, region)
			if err != nil {
				return "", fmt.Errorf("failed to create AWS client: %w", err)
			}
			d.awsClient = client
		}
		return d.awsClient.GetStatus(ctx, resourceID)

	case CloudProviderYandex:
		if d.yandexClient == nil {
			client, err := NewYandexCloudClient(ctx)
			if err != nil {
				return "", fmt.Errorf("failed to create Yandex Cloud client: %w", err)
			}
			d.yandexClient = client
		}
		return d.yandexClient.GetStatus(ctx, resourceID)

	default:
		return "", fmt.Errorf("unsupported cloud provider: %s", provider)
	}
}
