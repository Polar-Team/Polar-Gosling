package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/polar-gosling/gosling/internal/deployer"
	"github.com/polar-gosling/gosling/internal/mothergoose"
)

// MockMotherGooseClient is a mock implementation of MotherGooseClient for testing
type MockMotherGooseClient struct {
	GetEggStatusCalls       int
	ListEggsCalls           int
	CreateOrUpdateEggCalls  int
	GetDeploymentPlanCalls  int
	ListDeploymentPlanCalls int
	EggConfigs              map[string]*deployer.EggConfig
	EggStatuses             map[string]*mothergoose.EggStatus
	DeploymentPlans         map[string][]*deployer.DeploymentPlan
}

func NewMockMotherGooseClient() *MockMotherGooseClient {
	return &MockMotherGooseClient{
		EggConfigs:      make(map[string]*deployer.EggConfig),
		EggStatuses:     make(map[string]*mothergoose.EggStatus),
		DeploymentPlans: make(map[string][]*deployer.DeploymentPlan),
	}
}

func (m *MockMotherGooseClient) GetEggStatus(ctx context.Context, eggName string) (*mothergoose.EggStatus, error) {
	m.GetEggStatusCalls++
	if status, ok := m.EggStatuses[eggName]; ok {
		return status, nil
	}
	return &mothergoose.EggStatus{
		EggName:           eggName,
		LatestPlan:        nil,
		DeploymentHistory: []*deployer.DeploymentPlan{},
		ActiveRunners:     []*mothergoose.Runner{},
		ConfigHash:        "",
	}, nil
}

func (m *MockMotherGooseClient) ListEggs(ctx context.Context) ([]*deployer.EggConfig, error) {
	m.ListEggsCalls++
	eggs := make([]*deployer.EggConfig, 0, len(m.EggConfigs))
	for _, egg := range m.EggConfigs {
		eggs = append(eggs, egg)
	}
	return eggs, nil
}

func (m *MockMotherGooseClient) CreateOrUpdateEgg(ctx context.Context, config *deployer.EggConfig) error {
	m.CreateOrUpdateEggCalls++
	m.EggConfigs[config.Name] = config
	return nil
}

func (m *MockMotherGooseClient) GetDeploymentPlan(ctx context.Context, eggName, planID string) (*deployer.DeploymentPlan, error) {
	m.GetDeploymentPlanCalls++
	if plans, ok := m.DeploymentPlans[eggName]; ok {
		for _, plan := range plans {
			if plan.ID == planID {
				return plan, nil
			}
		}
	}
	return nil, fmt.Errorf("plan not found")
}

func (m *MockMotherGooseClient) ListDeploymentPlans(ctx context.Context, eggName string) ([]*deployer.DeploymentPlan, error) {
	m.ListDeploymentPlanCalls++
	if plans, ok := m.DeploymentPlans[eggName]; ok {
		return plans, nil
	}
	return []*deployer.DeploymentPlan{}, nil
}

// Feature: gitops-runner-orchestration, Property 24: Dry-Run Non-Modification
// Validates: Requirements 10.8
func TestDryRunNonModification(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("dry-run mode creates no cloud resources and makes no API calls to store configurations",
		prop.ForAll(
			func(eggConfig *deployer.EggConfig, cloudProvider deployer.CloudProvider, region string) bool {
				// Create a temporary Nest repository structure
				tempDir, err := os.MkdirTemp("", "nest-test-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tempDir)

				// Create Eggs directory
				eggsDir := filepath.Join(tempDir, "Eggs")
				if err := os.MkdirAll(eggsDir, 0755); err != nil {
					t.Logf("Failed to create Eggs dir: %v", err)
					return false
				}

				// Create egg directory and config.fly file
				eggDir := filepath.Join(eggsDir, eggConfig.Name)
				if err := os.MkdirAll(eggDir, 0755); err != nil {
					t.Logf("Failed to create egg dir: %v", err)
					return false
				}

				configPath := filepath.Join(eggDir, "config.fly")
				configContent := generateConfigFly(eggConfig)
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Logf("Failed to write config.fly: %v", err)
					return false
				}

				// Create a mock MotherGoose client
				mockClient := NewMockMotherGooseClient()

				// Record initial state
				initialCreateOrUpdateCalls := mockClient.CreateOrUpdateEggCalls
				initialEggConfigsCount := len(mockClient.EggConfigs)

				// Set up the deploy command with dry-run flag
				ctx := context.Background()

				// Parse the egg configs
				eggs, err := parseEggConfigs(eggsDir)
				if err != nil {
					t.Logf("Failed to parse egg configs: %v", err)
					return false
				}

				if len(eggs) == 0 {
					t.Logf("No eggs parsed")
					return false
				}

				// Set dry-run flag to true
				originalDryRun := deployDryRun
				deployDryRun = true
				defer func() { deployDryRun = originalDryRun }()

				// Execute deployment with dry-run
				for _, egg := range eggs {
					if err := deployEgg(ctx, egg, cloudProvider, region, mockClient); err != nil {
						t.Logf("Deploy failed: %v", err)
						return false
					}
				}

				// Verify that no API calls were made to create or update eggs
				if mockClient.CreateOrUpdateEggCalls != initialCreateOrUpdateCalls {
					t.Logf("Expected no CreateOrUpdateEgg calls in dry-run mode, but got %d calls",
						mockClient.CreateOrUpdateEggCalls-initialCreateOrUpdateCalls)
					return false
				}

				// Verify that no egg configurations were stored
				if len(mockClient.EggConfigs) != initialEggConfigsCount {
					t.Logf("Expected no new egg configs in dry-run mode, but got %d new configs",
						len(mockClient.EggConfigs)-initialEggConfigsCount)
					return false
				}

				// Verify that GetEggStatus was still called (to check for changes)
				// This is acceptable in dry-run mode as it's a read-only operation
				if mockClient.GetEggStatusCalls == 0 {
					t.Logf("Expected GetEggStatus to be called even in dry-run mode")
					return false
				}

				return true
			},
			genDryRunEggConfig(),
			genDryRunCloudProvider(),
			genDryRunRegion(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genDryRunEggConfig generates random EggConfig for testing
func genDryRunEggConfig() gopter.Gen {
	return gen.Identifier().Map(func(name string) *deployer.EggConfig {
		return &deployer.EggConfig{
			Name: name,
			Type: deployer.RunnerTypeVM,
			Cloud: deployer.CloudConfig{
				Provider: deployer.CloudProviderYandex,
				Region:   "ru-central1-a",
			},
			Resources: deployer.ResourceConfig{
				CPU:    2,
				Memory: 4096,
				Disk:   20,
			},
			Runner: deployer.RunnerConfig{
				Tags:        []string{"docker", "linux"},
				Concurrent:  3,
				IdleTimeout: 10 * time.Minute,
			},
			GitLab: deployer.GitLabConfig{
				ProjectID:   12345,
				TokenSecret: "vault://gitlab/runner-token",
			},
			Environment: map[string]string{
				"DOCKER_DRIVER": "overlay2",
			},
		}
	})
}

// genDryRunCloudProvider generates random CloudProvider for testing
func genDryRunCloudProvider() gopter.Gen {
	return gen.OneConstOf(deployer.CloudProviderYandex, deployer.CloudProviderAWS)
}

// genDryRunRegion generates random region strings for testing
func genDryRunRegion() gopter.Gen {
	return gen.OneConstOf("ru-central1-a", "us-east-1", "eu-west-1")
}

// generateConfigFly generates a .fly configuration file content from EggConfig
func generateConfigFly(egg *deployer.EggConfig) string {
	return fmt.Sprintf(`egg "%s" {
  type = "%s"

  cloud {
    provider = "%s"
    region   = "%s"
  }

  resources {
    cpu    = %d
    memory = %d
    disk   = %d
  }

  runner {
    tags = [%s]
    concurrent = %d
    idle_timeout = "%s"
  }

  gitlab {
    project_id = %d
    token_secret = "%s"
  }

  environment {
    DOCKER_DRIVER = "%s"
  }
}
`,
		egg.Name,
		egg.Type,
		egg.Cloud.Provider,
		egg.Cloud.Region,
		egg.Resources.CPU,
		egg.Resources.Memory,
		egg.Resources.Disk,
		formatTags(egg.Runner.Tags),
		egg.Runner.Concurrent,
		egg.Runner.IdleTimeout.String(),
		egg.GitLab.ProjectID,
		egg.GitLab.TokenSecret,
		egg.Environment["DOCKER_DRIVER"],
	)
}

// formatTags formats a slice of tags for .fly file
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, tag)
	}
	return result
}
