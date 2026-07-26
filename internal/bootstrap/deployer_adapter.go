package bootstrap

import (
	"context"

	"github.com/polar-gosling/gosling/internal/deployer"
)

// Compile-time interface check.
var _ BackendDeployer = (*DeployerAdapter)(nil)

// DeployerAdapter adapts *deployer.Deployer to the BackendDeployer interface.
type DeployerAdapter struct {
	deployer *deployer.Deployer
}

// NewDeployerAdapter creates a new DeployerAdapter wrapping the given Deployer.
func NewDeployerAdapter(d *deployer.Deployer) *DeployerAdapter {
	return &DeployerAdapter{deployer: d}
}

// DeployBackend deploys backend infrastructure and converts the result to bootstrap.DeployResult.
func (a *DeployerAdapter) DeployBackend(ctx context.Context, mgCfg *deployer.MGConfig, ufCfg *deployer.UFConfig) (*DeployResult, error) {
	result, err := a.deployer.DeployBackendInfrastructure(ctx, mgCfg, ufCfg)
	if err != nil {
		return nil, err
	}

	return &DeployResult{
		APIGatewayURL: result.APIGatewayURL,
		APIKey:        result.APIKey,
	}, nil
}
