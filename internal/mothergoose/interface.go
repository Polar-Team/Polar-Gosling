package mothergoose

import (
	"context"
	"time"

	"github.com/polar-gosling/gosling/internal/deployer"
)

// RunnerMetricsPayload is the JSON body sent to POST /runners/{id}/metrics.
type RunnerMetricsPayload struct {
	EggName          string    `json:"egg_name"`
	State            string    `json:"state"`
	JobCount         int       `json:"job_count"`
	CPUUsage         float64   `json:"cpu_usage"`
	MemoryUsage      float64   `json:"memory_usage"`
	DiskUsage        float64   `json:"disk_usage"`
	AgentVersion     string    `json:"agent_version"`
	FailureCount     int       `json:"failure_count"`
	LastJobTimestamp time.Time `json:"last_job_timestamp"`
}

// HeartbeatPayload is the JSON body sent to POST /runners/{id}/heartbeat.
type HeartbeatPayload struct {
	EggName string `json:"egg_name"`
	State   string `json:"state"`
}

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

	// SendHeartbeat sends a liveness ping for the given runner ID.
	SendHeartbeat(ctx context.Context, runnerID string, payload HeartbeatPayload) error

	// ReportRunnerMetrics posts a full metrics snapshot for the given runner ID.
	ReportRunnerMetrics(ctx context.Context, runnerID string, payload RunnerMetricsPayload) error
}
