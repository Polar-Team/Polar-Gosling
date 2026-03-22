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

// DeployBackendInfrastructure deploys the backend infrastructure (MotherGoose, UglyFox, databases).
// The cloud provider is determined from mgCfg.Cloud.Provider — no separate provider parameter needed.
func (d *Deployer) DeployBackendInfrastructure(ctx context.Context, mgCfg *MGConfig, ufCfg *UFConfig) error {
	switch mgCfg.Cloud.Provider {
	case CloudProviderAWS:
		if d.awsClient == nil {
			client, err := NewAWSClient(ctx, mgCfg.Cloud.AWSRegion)
			if err != nil {
				return fmt.Errorf("failed to create AWS client: %w", err)
			}
			d.awsClient = client
		}
		return d.awsClient.DeployBackendInfrastructure(ctx, mgCfg, ufCfg)

	case CloudProviderYandex:
		if d.yandexClient == nil {
			client, err := NewYandexCloudClient(ctx, mgCfg.Cloud.YCFolderID, mgCfg.Cloud.YCCloudID)
			if err != nil {
				return fmt.Errorf("failed to create Yandex Cloud client: %w", err)
			}
			d.yandexClient = client
		}
		return d.yandexClient.DeployBackendInfrastructure(ctx, mgCfg, ufCfg)

	default:
		return fmt.Errorf("unsupported cloud provider: %s", mgCfg.Cloud.Provider)
	}
}
