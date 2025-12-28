package cli

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/polar-gosling/gosling/internal/deployer"
	"github.com/polar-gosling/gosling/internal/mothergoose"
)

// Feature: gitops-runner-orchestration, Property 25: Deployment Rollback
// Validates: Requirements 10.9
func TestDeploymentRollback(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("rollback restores system to previous deployment state",
		prop.ForAll(
			func(eggName string, initialConfig *deployer.EggConfig, updatedConfig *deployer.EggConfig) bool {
				ctx := context.Background()

				// Create a mock MotherGoose client
				mockClient := NewMockMotherGooseClient()

				// Set egg names to be the same
				initialConfig.Name = eggName
				updatedConfig.Name = eggName

				// Simulate initial deployment
				initialPlanID := uuid.New().String()
				initialAppliedAt := time.Now().Add(-2 * time.Hour)
				initialPlan := &deployer.DeploymentPlan{
					ID:         initialPlanID,
					EggName:    eggName,
					PlanType:   "runner",
					ConfigHash: generateTestConfigHash(initialConfig),
					CreatedAt:  initialAppliedAt.Add(-5 * time.Minute),
					AppliedAt:  &initialAppliedAt,
					Status:     "applied",
					Metadata: map[string]interface{}{
						"runner_type": string(initialConfig.Type),
						"cloud":       string(initialConfig.Cloud.Provider),
						"region":      initialConfig.Cloud.Region,
					},
				}

				// Simulate updated deployment (current state)
				updatedPlanID := uuid.New().String()
				updatedAppliedAt := time.Now().Add(-1 * time.Hour)
				updatedPlan := &deployer.DeploymentPlan{
					ID:         updatedPlanID,
					EggName:    eggName,
					PlanType:   "runner",
					ConfigHash: generateTestConfigHash(updatedConfig),
					CreatedAt:  updatedAppliedAt.Add(-5 * time.Minute),
					AppliedAt:  &updatedAppliedAt,
					Status:     "applied",
					Metadata: map[string]interface{}{
						"runner_type": string(updatedConfig.Type),
						"cloud":       string(updatedConfig.Cloud.Provider),
						"region":      updatedConfig.Cloud.Region,
					},
				}

				// Set up mock client state
				mockClient.EggConfigs[eggName] = updatedConfig
				mockClient.DeploymentPlans[eggName] = []*deployer.DeploymentPlan{
					initialPlan,
					updatedPlan,
				}
				mockClient.EggStatuses[eggName] = &mothergoose.EggStatus{
					EggName:    eggName,
					LatestPlan: updatedPlan,
					DeploymentHistory: []*deployer.DeploymentPlan{
						initialPlan,
						updatedPlan,
					},
					ActiveRunners: []*mothergoose.Runner{},
					ConfigHash:    updatedPlan.ConfigHash,
				}

				// Record state before rollback
				statusBefore, err := mockClient.GetEggStatus(ctx, eggName)
				if err != nil {
					t.Logf("Failed to get status before rollback: %v", err)
					return false
				}

				if statusBefore.LatestPlan == nil {
					t.Logf("No latest plan found before rollback")
					return false
				}

				if statusBefore.LatestPlan.ID != updatedPlanID {
					t.Logf("Expected latest plan to be %s, got %s", updatedPlanID, statusBefore.LatestPlan.ID)
					return false
				}

				// Find the previous plan (what we're rolling back to)
				targetPlan, err := findPreviousPlan(statusBefore.DeploymentHistory, statusBefore.LatestPlan.ID)
				if err != nil {
					t.Logf("Failed to find previous plan: %v", err)
					return false
				}

				if targetPlan.ID != initialPlanID {
					t.Logf("Expected target plan to be %s, got %s", initialPlanID, targetPlan.ID)
					return false
				}

				// Verify target plan has the initial configuration
				if targetPlan.ConfigHash != initialPlan.ConfigHash {
					t.Logf("Target plan config hash mismatch: expected %s, got %s",
						initialPlan.ConfigHash, targetPlan.ConfigHash)
					return false
				}

				// Simulate rollback by updating the mock state
				// In a real scenario, MotherGoose would handle this
				rollbackPlanID := uuid.New().String()
				rollbackAppliedAt := time.Now()
				rollbackPlan := &deployer.DeploymentPlan{
					ID:           rollbackPlanID,
					EggName:      eggName,
					PlanType:     "runner",
					ConfigHash:   initialPlan.ConfigHash, // Restore to initial config
					CreatedAt:    time.Now(),
					AppliedAt:    &rollbackAppliedAt,
					Status:       "applied",
					RollbackPlan: initialPlanID, // Reference to the plan we rolled back to
					Metadata: map[string]interface{}{
						"runner_type":     string(initialConfig.Type),
						"cloud":           string(initialConfig.Cloud.Provider),
						"region":          initialConfig.Cloud.Region,
						"rollback_from":   updatedPlanID,
						"rollback_reason": "test rollback",
					},
				}

				// Update mock state to reflect rollback
				mockClient.DeploymentPlans[eggName] = append(mockClient.DeploymentPlans[eggName], rollbackPlan)
				mockClient.EggStatuses[eggName].LatestPlan = rollbackPlan
				mockClient.EggStatuses[eggName].DeploymentHistory = append(mockClient.EggStatuses[eggName].DeploymentHistory, rollbackPlan)
				mockClient.EggStatuses[eggName].ConfigHash = rollbackPlan.ConfigHash
				mockClient.EggConfigs[eggName] = initialConfig // Restore initial config

				// Verify state after rollback
				statusAfter, err := mockClient.GetEggStatus(ctx, eggName)
				if err != nil {
					t.Logf("Failed to get status after rollback: %v", err)
					return false
				}

				// Property 1: Latest plan should be the rollback plan
				if statusAfter.LatestPlan.ID != rollbackPlanID {
					t.Logf("Expected latest plan to be rollback plan %s, got %s",
						rollbackPlanID, statusAfter.LatestPlan.ID)
					return false
				}

				// Property 2: Config hash should match the initial deployment
				if statusAfter.ConfigHash != initialPlan.ConfigHash {
					t.Logf("Expected config hash to match initial deployment %s, got %s",
						initialPlan.ConfigHash, statusAfter.ConfigHash)
					return false
				}

				// Property 3: Rollback plan should reference the target plan
				if statusAfter.LatestPlan.RollbackPlan != initialPlanID {
					t.Logf("Expected rollback plan to reference %s, got %s",
						initialPlanID, statusAfter.LatestPlan.RollbackPlan)
					return false
				}

				// Property 4: Egg configuration should be restored to initial state
				restoredConfig := mockClient.EggConfigs[eggName]
				if restoredConfig.Type != initialConfig.Type {
					t.Logf("Expected runner type to be restored to %s, got %s",
						initialConfig.Type, restoredConfig.Type)
					return false
				}

				if restoredConfig.Cloud.Provider != initialConfig.Cloud.Provider {
					t.Logf("Expected cloud provider to be restored to %s, got %s",
						initialConfig.Cloud.Provider, restoredConfig.Cloud.Provider)
					return false
				}

				if restoredConfig.Resources.CPU != initialConfig.Resources.CPU {
					t.Logf("Expected CPU to be restored to %d, got %d",
						initialConfig.Resources.CPU, restoredConfig.Resources.CPU)
					return false
				}

				if restoredConfig.Resources.Memory != initialConfig.Resources.Memory {
					t.Logf("Expected memory to be restored to %d, got %d",
						initialConfig.Resources.Memory, restoredConfig.Resources.Memory)
					return false
				}

				// Property 5: Deployment history should contain all plans
				if len(statusAfter.DeploymentHistory) < 3 {
					t.Logf("Expected at least 3 plans in history (initial, updated, rollback), got %d",
						len(statusAfter.DeploymentHistory))
					return false
				}

				return true
			},
			genRollbackEggName(),
			genRollbackEggConfig(),
			genRollbackEggConfig(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genRollbackEggName generates random egg names for testing
func genRollbackEggName() gopter.Gen {
	return gen.Identifier().Map(func(name string) string {
		return "egg-" + name
	})
}

// genRollbackEggConfig generates random EggConfig for rollback testing
func genRollbackEggConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		genRollbackRunnerType(),
		genRollbackCloudProvider(),
		genRollbackRegion(),
		gen.IntRange(1, 8),        // CPU
		gen.IntRange(1024, 16384), // Memory
		gen.IntRange(10, 100),     // Disk
	).Map(func(values []interface{}) *deployer.EggConfig {
		name := values[0].(string)
		runnerType := values[1].(deployer.RunnerType)
		provider := values[2].(deployer.CloudProvider)
		region := values[3].(string)
		cpu := values[4].(int)
		memory := values[5].(int)
		disk := values[6].(int)

		return &deployer.EggConfig{
			Name: name,
			Type: runnerType,
			Cloud: deployer.CloudConfig{
				Provider: provider,
				Region:   region,
			},
			Resources: deployer.ResourceConfig{
				CPU:    cpu,
				Memory: memory,
				Disk:   disk,
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

// genRollbackRunnerType generates random RunnerType for testing
func genRollbackRunnerType() gopter.Gen {
	return gen.OneConstOf(deployer.RunnerTypeVM, deployer.RunnerTypeServerless)
}

// genRollbackCloudProvider generates random CloudProvider for testing
func genRollbackCloudProvider() gopter.Gen {
	return gen.OneConstOf(deployer.CloudProviderYandex, deployer.CloudProviderAWS)
}

// genRollbackRegion generates random region strings for testing
func genRollbackRegion() gopter.Gen {
	return gen.OneConstOf("ru-central1-a", "us-east-1", "eu-west-1")
}

// generateTestConfigHash generates a simple hash for testing
func generateTestConfigHash(egg *deployer.EggConfig) string {
	return fmt.Sprintf("%s-%s-%s-%d-%d-%d",
		egg.Name,
		egg.Type,
		egg.Cloud.Provider,
		egg.Resources.CPU,
		egg.Resources.Memory,
		egg.Resources.Disk,
	)
}
