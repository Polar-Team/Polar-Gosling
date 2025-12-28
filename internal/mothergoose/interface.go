package mothergoose

import (
	"context"

	"github.com/polar-gosling/gosling/internal/deployer"
)

// MotherGooseClient is the interface for communicating with MotherGoose API
// CRITICAL: Gosling CLI MUST use this interface to query deployment status
// Gosling CLI MUST NOT access the database directly
type MotherGooseClient interface {
	// GetEggStatus retrieves deployment status for an Egg
	GetEggStatus(ctx context.Context, eggName string) (*EggStatus, error)

	// ListEggs lists all Egg configurations
	ListEggs(ctx context.Context) ([]*deployer.EggConfig, error)

	// CreateOrUpdateEgg creates or updates an Egg configuration
	CreateOrUpdateEgg(ctx context.Context, config *deployer.EggConfig) error

	// GetDeploymentPlan retrieves a specific deployment plan
	GetDeploymentPlan(ctx context.Context, eggName, planID string) (*deployer.DeploymentPlan, error)

	// ListDeploymentPlans lists all deployment plans for an Egg
	ListDeploymentPlans(ctx context.Context, eggName string) ([]*deployer.DeploymentPlan, error)
}
