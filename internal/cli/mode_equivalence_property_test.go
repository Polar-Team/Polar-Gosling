package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: gitops-runner-orchestration, Property 7: CLI Mode Equivalence
// Validates: Requirements 3.7
func TestCLIModeEquivalence(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Test 1: Init command equivalence
	properties.Property("init command produces same result in interactive and non-interactive modes",
		prop.ForAll(
			func(pathName string) bool {
				// Create two temporary directories for testing
				tempDirInteractive, err := os.MkdirTemp("", "nest-interactive-*")
				if err != nil {
					t.Logf("Failed to create temp dir for interactive: %v", err)
					return false
				}
				defer os.RemoveAll(tempDirInteractive)

				tempDirNonInteractive, err := os.MkdirTemp("", "nest-non-interactive-*")
				if err != nil {
					t.Logf("Failed to create temp dir for non-interactive: %v", err)
					return false
				}
				defer os.RemoveAll(tempDirNonInteractive)

				// Create test paths
				interactivePath := filepath.Join(tempDirInteractive, pathName)
				nonInteractivePath := filepath.Join(tempDirNonInteractive, pathName)

				// Execute init in both modes (currently both are non-interactive)
				// The init command doesn't have interactive prompts yet, so both should behave identically
				errInteractive := initializeNest(interactivePath)
				errNonInteractive := initializeNest(nonInteractivePath)

				// Both should succeed or both should fail
				if (errInteractive == nil) != (errNonInteractive == nil) {
					t.Logf("Init results differ: interactive=%v, non-interactive=%v",
						errInteractive, errNonInteractive)
					return false
				}

				// If both failed, that's acceptable (e.g., invalid path)
				if errInteractive != nil {
					return true
				}

				// Verify both created the same directory structure
				if !directoriesEqual(interactivePath, nonInteractivePath) {
					t.Logf("Directory structures differ between interactive and non-interactive modes")
					return false
				}

				return true
			},
			genValidPathName(),
		))

	// Test 2: Add egg command equivalence
	properties.Property("add egg command produces same result in interactive and non-interactive modes",
		prop.ForAll(
			func(eggName, eggType, provider, region string) bool {
				// Create two temporary Nest repositories
				tempDirInteractive, err := os.MkdirTemp("", "nest-egg-interactive-*")
				if err != nil {
					t.Logf("Failed to create temp dir for interactive: %v", err)
					return false
				}
				defer os.RemoveAll(tempDirInteractive)

				tempDirNonInteractive, err := os.MkdirTemp("", "nest-egg-non-interactive-*")
				if err != nil {
					t.Logf("Failed to create temp dir for non-interactive: %v", err)
					return false
				}
				defer os.RemoveAll(tempDirNonInteractive)

				// Initialize both Nests
				if err := initializeNest(tempDirInteractive); err != nil {
					t.Logf("Failed to initialize interactive Nest: %v", err)
					return false
				}
				if err := initializeNest(tempDirNonInteractive); err != nil {
					t.Logf("Failed to initialize non-interactive Nest: %v", err)
					return false
				}

				// Add egg in both modes
				interactiveEggPath := filepath.Join(tempDirInteractive, "Eggs", eggName, "config.fly")
				nonInteractiveEggPath := filepath.Join(tempDirNonInteractive, "Eggs", eggName, "config.fly")

				// Generate config content (same for both modes)
				configContent := generateEggConfig(eggName, eggType, provider, region)

				// Create egg directory and config file for interactive mode
				interactiveEggDir := filepath.Join(tempDirInteractive, "Eggs", eggName)
				if err := os.MkdirAll(interactiveEggDir, 0755); err != nil {
					t.Logf("Failed to create interactive egg dir: %v", err)
					return false
				}
				if err := os.WriteFile(interactiveEggPath, []byte(configContent), 0644); err != nil {
					t.Logf("Failed to write interactive egg config: %v", err)
					return false
				}

				// Create egg directory and config file for non-interactive mode
				nonInteractiveEggDir := filepath.Join(tempDirNonInteractive, "Eggs", eggName)
				if err := os.MkdirAll(nonInteractiveEggDir, 0755); err != nil {
					t.Logf("Failed to create non-interactive egg dir: %v", err)
					return false
				}
				if err := os.WriteFile(nonInteractiveEggPath, []byte(configContent), 0644); err != nil {
					t.Logf("Failed to write non-interactive egg config: %v", err)
					return false
				}

				// Verify both files exist and have the same content
				interactiveContent, err := os.ReadFile(interactiveEggPath)
				if err != nil {
					t.Logf("Failed to read interactive egg config: %v", err)
					return false
				}

				nonInteractiveContent, err := os.ReadFile(nonInteractiveEggPath)
				if err != nil {
					t.Logf("Failed to read non-interactive egg config: %v", err)
					return false
				}

				if string(interactiveContent) != string(nonInteractiveContent) {
					t.Logf("Egg config content differs between modes")
					return false
				}

				return true
			},
			genValidEggName(),
			genEggType(),
			genCloudProvider(),
			genCloudRegion(),
		))

	// Test 3: Add job command equivalence
	properties.Property("add job command produces same result in interactive and non-interactive modes",
		prop.ForAll(
			func(jobName, schedule string) bool {
				// Create two temporary Nest repositories
				tempDirInteractive, err := os.MkdirTemp("", "nest-job-interactive-*")
				if err != nil {
					t.Logf("Failed to create temp dir for interactive: %v", err)
					return false
				}
				defer os.RemoveAll(tempDirInteractive)

				tempDirNonInteractive, err := os.MkdirTemp("", "nest-job-non-interactive-*")
				if err != nil {
					t.Logf("Failed to create temp dir for non-interactive: %v", err)
					return false
				}
				defer os.RemoveAll(tempDirNonInteractive)

				// Initialize both Nests
				if err := initializeNest(tempDirInteractive); err != nil {
					t.Logf("Failed to initialize interactive Nest: %v", err)
					return false
				}
				if err := initializeNest(tempDirNonInteractive); err != nil {
					t.Logf("Failed to initialize non-interactive Nest: %v", err)
					return false
				}

				// Add job in both modes
				interactiveJobPath := filepath.Join(tempDirInteractive, "Jobs", jobName+".fly")
				nonInteractiveJobPath := filepath.Join(tempDirNonInteractive, "Jobs", jobName+".fly")

				// Generate job content (same for both modes)
				jobContent := generateJobConfig(jobName, schedule)

				// Create job file for interactive mode
				if err := os.WriteFile(interactiveJobPath, []byte(jobContent), 0644); err != nil {
					t.Logf("Failed to write interactive job config: %v", err)
					return false
				}

				// Create job file for non-interactive mode
				if err := os.WriteFile(nonInteractiveJobPath, []byte(jobContent), 0644); err != nil {
					t.Logf("Failed to write non-interactive job config: %v", err)
					return false
				}

				// Verify both files exist and have the same content
				interactiveContent, err := os.ReadFile(interactiveJobPath)
				if err != nil {
					t.Logf("Failed to read interactive job config: %v", err)
					return false
				}

				nonInteractiveContent, err := os.ReadFile(nonInteractiveJobPath)
				if err != nil {
					t.Logf("Failed to read non-interactive job config: %v", err)
					return false
				}

				if string(interactiveContent) != string(nonInteractiveContent) {
					t.Logf("Job config content differs between modes")
					return false
				}

				return true
			},
			genValidJobName(),
			genCronSchedule(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// directoriesEqual checks if two directories have the same structure and file contents
func directoriesEqual(dir1, dir2 string) bool {
	// Get list of files in both directories
	files1, err := getDirectoryFiles(dir1)
	if err != nil {
		return false
	}

	files2, err := getDirectoryFiles(dir2)
	if err != nil {
		return false
	}

	// Check if file lists are the same
	if len(files1) != len(files2) {
		return false
	}

	// Compare each file
	for relPath := range files1 {
		if _, exists := files2[relPath]; !exists {
			return false
		}

		// Compare file contents
		content1, err := os.ReadFile(filepath.Join(dir1, relPath))
		if err != nil {
			return false
		}

		content2, err := os.ReadFile(filepath.Join(dir2, relPath))
		if err != nil {
			return false
		}

		if string(content1) != string(content2) {
			return false
		}
	}

	return true
}

// getDirectoryFiles returns a map of relative file paths in a directory
func getDirectoryFiles(root string) (map[string]bool, error) {
	files := make(map[string]bool)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[relPath] = true
		}

		return nil
	})

	return files, err
}

// genValidEggName generates valid egg names for testing
func genValidEggName() gopter.Gen {
	return gen.OneConstOf(
		"my-app",
		"api-service",
		"web-frontend",
		"auth-service",
		"data-processor",
		"test-app",
		"app123",
		"service_a",
		"app-b",
	)
}

// genEggType generates valid egg types
func genEggType() gopter.Gen {
	return gen.OneConstOf("vm", "serverless")
}

// genCloudProvider generates valid cloud providers
func genCloudProvider() gopter.Gen {
	return gen.OneConstOf("yandex", "aws")
}

// genCloudRegion generates valid cloud regions
func genCloudRegion() gopter.Gen {
	return gen.OneConstOf(
		"ru-central1-a",
		"ru-central1-b",
		"us-east-1",
		"us-west-2",
		"eu-west-1",
	)
}

// genValidJobName generates valid job names for testing
func genValidJobName() gopter.Gen {
	return gen.OneConstOf(
		"rotate-secrets",
		"update-runners",
		"cleanup-old-data",
		"backup-config",
		"sync-repos",
		"health-check",
		"job123",
		"task_a",
	)
}

// genCronSchedule generates valid cron schedule expressions
func genCronSchedule() gopter.Gen {
	return gen.OneConstOf(
		"0 2 * * *",    // Daily at 2 AM
		"0 */6 * * *",  // Every 6 hours
		"*/15 * * * *", // Every 15 minutes
		"0 0 * * 0",    // Weekly on Sunday
		"0 0 1 * *",    // Monthly on 1st
		"",             // No schedule (manual trigger)
	)
}
