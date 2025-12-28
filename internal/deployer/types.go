package deployer

import (
	"context"
	"time"
)

// CloudProvider represents the cloud provider type
type CloudProvider string

const (
	CloudProviderYandex CloudProvider = "yandex"
	CloudProviderAWS    CloudProvider = "aws"
)

// RunnerType represents the type of runner
type RunnerType string

const (
	RunnerTypeVM         RunnerType = "vm"
	RunnerTypeServerless RunnerType = "serverless"
)

// CloudConfig represents cloud provider configuration
type CloudConfig struct {
	Provider CloudProvider
	Region   string
}

// ResourceConfig represents resource requirements
type ResourceConfig struct {
	CPU    int // Number of CPU cores
	Memory int // Memory in MB
	Disk   int // Disk size in GB
}

// RunnerConfig represents runner-specific configuration
type RunnerConfig struct {
	Tags        []string
	Concurrent  int
	IdleTimeout time.Duration
}

// GitLabConfig represents GitLab integration configuration
type GitLabConfig struct {
	ProjectID   int
	TokenSecret string // Secret URI (yc-lockbox://, aws-sm://, vault://)
}

// EggConfig represents a complete Egg configuration
type EggConfig struct {
	Name        string
	Type        RunnerType
	Cloud       CloudConfig
	Resources   ResourceConfig
	Runner      RunnerConfig
	GitLab      GitLabConfig
	Environment map[string]string
}

// EggsBucketConfig represents a configuration for multiple repositories
type EggsBucketConfig struct {
	Name         string
	Type         RunnerType
	Cloud        CloudConfig
	Resources    ResourceConfig
	Runner       RunnerConfig
	Repositories []RepositoryConfig
	Environment  map[string]string
}

// RepositoryConfig represents a single repository in an EggsBucket
type RepositoryConfig struct {
	Name   string
	GitLab GitLabConfig
}

// VMConfig represents VM-specific deployment configuration
type VMConfig struct {
	EggName     string
	Cloud       CloudConfig
	Resources   ResourceConfig
	Runner      RunnerConfig
	GitLab      GitLabConfig
	Environment map[string]string
}

// ServerlessConfig represents serverless container deployment configuration
type ServerlessConfig struct {
	EggName     string
	Cloud       CloudConfig
	Resources   ResourceConfig
	Runner      RunnerConfig
	GitLab      GitLabConfig
	Environment map[string]string
	Timeout     time.Duration // Maximum execution time (60 minutes for serverless)
}

// VM represents a deployed virtual machine
type VM struct {
	ID            string
	EggName       string
	Provider      CloudProvider
	Region        string
	State         string
	PublicIP      string
	PrivateIP     string
	CreatedAt     time.Time
	LastHeartbeat time.Time
}

// Container represents a deployed serverless container
type Container struct {
	ID        string
	EggName   string
	Provider  CloudProvider
	Region    string
	State     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// CloudDeployer is the interface for deploying backend infrastructure to cloud providers
// Note: Individual runner deployment is handled by MotherGoose using OpenTofu, not by Gosling CLI
type CloudDeployer interface {
	// DeployBackendInfrastructure deploys MotherGoose, UglyFox, databases, and storage
	DeployBackendInfrastructure(ctx context.Context) error

	// GetStatus retrieves the current status of infrastructure
	GetStatus(ctx context.Context, resourceID string) (string, error)
}

// DeploymentPlan represents a deployment plan for rollback
type DeploymentPlan struct {
	ID           string
	EggName      string
	PlanType     string // "runner" or "rift"
	PlanBinary   []byte // OpenTofu plan output
	ConfigHash   string
	CreatedAt    time.Time
	AppliedAt    *time.Time
	Status       string // "pending", "applied", "rolled_back"
	RollbackPlan string // ID of the plan to rollback to
	Metadata     map[string]interface{}
}
