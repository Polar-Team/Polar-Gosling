package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	eggType     string
	eggProvider string
	eggRegion   string
	jobSchedule string
	interactive bool
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add configurations to the Nest repository",
	Long:  `Add Egg configurations or Job definitions to the Nest repository.`,
}

// addEggCmd represents the add egg command
var addEggCmd = &cobra.Command{
	Use:   "egg <name>",
	Short: "Add a new Egg configuration",
	Long: `Add a new Egg configuration for a managed repository.

An Egg represents a single managed repository with its runner configuration.
The configuration file will be created at Eggs/<name>/config.fly

Example:
  gosling add egg my-app --type vm --provider yandex
  gosling add egg api-service --type serverless --provider aws`,
	Args: cobra.ExactArgs(1),
	RunE: runAddEgg,
}

// addJobCmd represents the add job command
var addJobCmd = &cobra.Command{
	Use:   "job <name>",
	Short: "Add a new Job definition",
	Long: `Add a new self-management Job definition.

Jobs are automated tasks that maintain the Nest repository and runner infrastructure.
The job file will be created at Jobs/<name>.fly

Example:
  gosling add job rotate-secrets --schedule "0 2 * * *"
  gosling add job update-runners`,
	Args: cobra.ExactArgs(1),
	RunE: runAddJob,
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.AddCommand(addEggCmd)
	addCmd.AddCommand(addJobCmd)

	// Egg flags
	addEggCmd.Flags().StringVarP(&eggType, "type", "t", "vm", "Runner type: vm or serverless")
	addEggCmd.Flags().StringVarP(&eggProvider, "provider", "p", "yandex", "Cloud provider: yandex or aws")
	addEggCmd.Flags().StringVarP(&eggRegion, "region", "r", "", "Cloud region (e.g., ru-central1-a, us-east-1)")
	addEggCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")

	// Job flags
	addJobCmd.Flags().StringVarP(&jobSchedule, "schedule", "s", "", "Cron schedule expression")
	addJobCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")
}

func runAddEgg(cmd *cobra.Command, args []string) error {
	eggName := args[0]

	// Validate egg name
	if !isValidName(eggName) {
		return fmt.Errorf("invalid egg name: must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Validate type
	if eggType != "vm" && eggType != "serverless" {
		return fmt.Errorf("invalid type: must be 'vm' or 'serverless'")
	}

	// Validate provider
	if eggProvider != "yandex" && eggProvider != "aws" {
		return fmt.Errorf("invalid provider: must be 'yandex' or 'aws'")
	}

	// Set default region if not provided
	if eggRegion == "" {
		if eggProvider == "yandex" {
			eggRegion = "ru-central1-a"
		} else {
			eggRegion = "us-east-1"
		}
	}

	// Find Nest root
	nestRoot, err := findNestRoot()
	if err != nil {
		return fmt.Errorf("not in a Nest repository: %w\nRun 'gosling init' to create a new Nest repository", err)
	}

	// Create Egg directory
	eggDir := filepath.Join(nestRoot, "Eggs", eggName)
	if err := os.MkdirAll(eggDir, 0755); err != nil {
		return fmt.Errorf("failed to create Egg directory: %w", err)
	}

	// Create config.fly
	configPath := filepath.Join(eggDir, "config.fly")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("Egg configuration already exists at %s", configPath)
	}

	configContent := generateEggConfig(eggName, eggType, eggProvider, eggRegion)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config.fly: %w", err)
	}

	fmt.Printf("✅ Created Egg configuration: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit the configuration file to customize settings")
	fmt.Println("  2. Add GitLab project ID and token secret")
	fmt.Println("  3. Validate: gosling validate")
	fmt.Println("  4. Deploy: gosling deploy")

	return nil
}

func runAddJob(cmd *cobra.Command, args []string) error {
	jobName := args[0]

	// Validate job name
	if !isValidName(jobName) {
		return fmt.Errorf("invalid job name: must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Find Nest root
	nestRoot, err := findNestRoot()
	if err != nil {
		return fmt.Errorf("not in a Nest repository: %w\nRun 'gosling init' to create a new Nest repository", err)
	}

	// Create job file
	jobPath := filepath.Join(nestRoot, "Jobs", fmt.Sprintf("%s.fly", jobName))
	if _, err := os.Stat(jobPath); err == nil {
		return fmt.Errorf("Job definition already exists at %s", jobPath)
	}

	jobContent := generateJobConfig(jobName, jobSchedule)
	if err := os.WriteFile(jobPath, []byte(jobContent), 0644); err != nil {
		return fmt.Errorf("failed to create job file: %w", err)
	}

	fmt.Printf("✅ Created Job definition: %s\n", jobPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit the job file to define the script and configuration")
	fmt.Println("  2. Validate: gosling validate")
	fmt.Println("  3. Deploy: gosling deploy")

	return nil
}

func generateEggConfig(name, runnerType, provider, region string) string {
	// Determine default resources based on type
	cpu := 2
	memory := 4096
	disk := 20
	concurrent := 3

	if runnerType == "serverless" {
		cpu = 1
		memory = 2048
		disk = 10
		concurrent = 1
	}

	return fmt.Sprintf(`# Egg Configuration: %s
# Runner Type: %s
# Cloud Provider: %s

egg "%s" {
  type = "%s"
  
  cloud {
    provider = "%s"
    region   = "%s"
  }
  
  resources {
    cpu    = %d
    memory = %d  # MB
    disk   = %d  # GB
  }
  
  runner {
    tags       = ["docker", "linux"]
    concurrent = %d
    idle_timeout = "10m"
  }
  
  gitlab {
    # TODO: Set your GitLab project ID
    project_id = 0
    
    # TODO: Set your GitLab runner token secret
    # Format: yc-lockbox://{secret-id}/{key} or aws-sm://{secret-name}/{key}
    token_secret = "%s-lockbox://gitlab-tokens/%s-runner-token"
  }
  
  environment {
    DOCKER_DRIVER = "overlay2"
    # Add custom environment variables here
  }
}
`, name, runnerType, provider, name, runnerType, provider, region, cpu, memory, disk, concurrent, provider, name)
}

func generateJobConfig(name, schedule string) string {
	scheduleComment := ""
	scheduleValue := ""

	if schedule != "" {
		scheduleValue = fmt.Sprintf("\n  schedule = %q", schedule)
	} else {
		scheduleComment = "\n  # TODO: Set cron schedule expression (e.g., \"0 2 * * *\" for daily at 2 AM)"
		scheduleValue = "\n  # schedule = \"0 2 * * *\""
	}

	return fmt.Sprintf(`# Job Definition: %s
# Self-management task for Nest repository

job "%s" {%s%s
  
  runner {
    type = "vm"
    tags = ["privileged"]
  }
  
  script = <<-EOT
    #!/bin/bash
    set -e
    
    # TODO: Add your job script here
    echo "Running job: %s"
    
    # Example: Rotate secrets
    # gosling rotate-tokens --all
    
    # Example: Update runner images
    # gosling update-images --latest
  EOT
  
  on_failure {
    # TODO: Add notification email addresses
    notify = ["ops@example.com"]
  }
}
`, name, name, scheduleComment, scheduleValue, name)
}

func isValidName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func findNestRoot() (string, error) {
	// Start from current directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check if this directory has Eggs, Jobs, and UF subdirectories
		eggsPath := filepath.Join(dir, "Eggs")
		jobsPath := filepath.Join(dir, "Jobs")
		ufPath := filepath.Join(dir, "UF")

		eggsInfo, eggsErr := os.Stat(eggsPath)
		jobsInfo, jobsErr := os.Stat(jobsPath)
		ufInfo, ufErr := os.Stat(ufPath)

		if eggsErr == nil && eggsInfo.IsDir() &&
			jobsErr == nil && jobsInfo.IsDir() &&
			ufErr == nil && ufInfo.IsDir() {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding Nest
			return "", fmt.Errorf("Nest repository not found")
		}
		dir = parent
	}
}
