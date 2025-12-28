package deployer

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/polar-gosling/gosling/internal/parser"
)

// Feature: gitops-runner-orchestration, Property 8: Fly to Cloud SDK Conversion
// Validates: Requirements 3.9, 9.3
//
// This test validates that .fly Egg configurations can be converted to deployment configurations
// that will be passed to MotherGoose. MotherGoose then uses OpenTofu to deploy the actual runners.
// Gosling CLI's role is to parse and convert .fly files, not to deploy runners directly.
func TestFlyToCloudSDKConversion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid .fly Egg configuration converts to SDK objects that pass validation",
		prop.ForAll(
			func(eggType, provider string) bool {
				// Generate a valid Egg configuration
				egg := generateValidEggConfig(eggType, provider)

				// Convert to deployment configuration
				converter := NewConverter()

				var err error
				if eggType == "vm" {
					vmConfig, convErr := converter.EggToVMConfig(egg)
					if convErr != nil {
						t.Logf("Conversion error: %v", convErr)
						return false
					}

					// Validate that the VMConfig can be used with SDK
					err = validateVMConfigForSDK(vmConfig, provider)
				} else {
					serverlessConfig, convErr := converter.EggToServerlessConfig(egg)
					if convErr != nil {
						t.Logf("Conversion error: %v", convErr)
						return false
					}

					// Validate that the ServerlessConfig can be used with SDK
					err = validateServerlessConfigForSDK(serverlessConfig, provider)
				}

				if err != nil {
					t.Logf("SDK validation error: %v", err)
					return false
				}

				return true
			},
			gen.OneConstOf("vm", "serverless"),
			gen.OneConstOf("yandex", "aws"),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gitops-runner-orchestration, Property 8: Fly to Cloud SDK Conversion (EggsBucket)
// Validates: Requirements 3.9, 9.3
//
// This test validates that .fly EggsBucket configurations can be converted to multiple deployment
// configurations (one per repository) that will be passed to MotherGoose for runner deployment.
func TestFlyToCloudSDKConversionEggsBucket(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("valid .fly EggsBucket configuration converts to multiple SDK objects that pass validation",
		prop.ForAll(
			func(bucketType, provider string, repoCount int) bool {
				// Generate a valid EggsBucket configuration
				bucket := generateValidEggsBucketConfig(bucketType, provider, repoCount)

				// Convert to deployment configurations
				converter := NewConverter()

				if bucketType == "vm" {
					vmConfigs, convErr := converter.EggsBucketToVMConfigs(bucket)
					if convErr != nil {
						t.Logf("Conversion error: %v", convErr)
						return false
					}

					// Should have one config per repository
					if len(vmConfigs) != repoCount {
						t.Logf("Expected %d configs, got %d", repoCount, len(vmConfigs))
						return false
					}

					// Validate each VMConfig can be used with SDK
					for i, vmConfig := range vmConfigs {
						if err := validateVMConfigForSDK(vmConfig, provider); err != nil {
							t.Logf("SDK validation error for config %d: %v", i, err)
							return false
						}
					}
				} else {
					serverlessConfigs, convErr := converter.EggsBucketToServerlessConfigs(bucket)
					if convErr != nil {
						t.Logf("Conversion error: %v", convErr)
						return false
					}

					// Should have one config per repository
					if len(serverlessConfigs) != repoCount {
						t.Logf("Expected %d configs, got %d", repoCount, len(serverlessConfigs))
						return false
					}

					// Validate each ServerlessConfig can be used with SDK
					for i, serverlessConfig := range serverlessConfigs {
						if err := validateServerlessConfigForSDK(serverlessConfig, provider); err != nil {
							t.Logf("SDK validation error for config %d: %v", i, err)
							return false
						}
					}
				}

				return true
			},
			gen.OneConstOf("vm", "serverless"),
			gen.OneConstOf("yandex", "aws"),
			gen.IntRange(1, 5), // Test with 1-5 repositories
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// generateValidEggConfig creates a valid ParsedEggConfig for testing
func generateValidEggConfig(eggType, provider string) *ParsedEggConfig {
	region := "ru-central1-a"
	if provider == "aws" {
		region = "us-east-1"
	}

	// Generate provider-specific valid resources
	var cpu, memory, disk int

	if provider == "yandex" {
		// Yandex Cloud: CPU must be even (or 1), memory at least 1GB per CPU
		cpuOptions := []int{2, 4, 6, 8}
		cpu = cpuOptions[rand.Intn(len(cpuOptions))]

		if eggType == "serverless" {
			// Yandex Cloud Functions: specific memory sizes
			memoryOptions := []int{128, 256, 512, 1024, 2048, 4096}
			memory = memoryOptions[rand.Intn(len(memoryOptions))]
		} else {
			// VM: at least 1GB per CPU
			memory = cpu*1024 + rand.Intn(cpu*1024) // 1-2GB per CPU
		}

		disk = 20 + rand.Intn(80) // 20-100GB
	} else {
		// AWS: more flexible
		cpu = 1 + rand.Intn(7) // 1-7 CPUs

		if eggType == "serverless" {
			// AWS Lambda: 128MB to 10240MB
			memory = 128 + rand.Intn(10112) // 128-10240MB
		} else {
			// EC2: at least 512MB
			memory = 512 + rand.Intn(15872) // 512MB-16GB
		}

		disk = 8 + rand.Intn(92) // 8-100GB (AWS minimum is 8GB)
	}

	return &ParsedEggConfig{
		Name: fmt.Sprintf("test-egg-%d", rand.Intn(10000)),
		Type: eggType,
		Cloud: CloudInfo{
			Provider: provider,
			Region:   region,
		},
		Resources: ResourceInfo{
			CPU:    cpu,
			Memory: memory,
			Disk:   disk,
		},
		Runner: RunnerInfo{
			Tags:        []string{"docker", "linux"},
			Concurrent:  1 + rand.Intn(10), // 1-10 concurrent jobs
			IdleTimeout: "10m",
		},
		GitLab: GitLabInfo{
			ProjectID:   10000 + rand.Intn(90000), // Random project ID
			TokenSecret: fmt.Sprintf("vault://gitlab/runner-token-%d", rand.Intn(1000)),
		},
		Environment: map[string]string{
			"DOCKER_DRIVER": "overlay2",
			"TEST_VAR":      "test-value",
		},
	}
}

// generateValidEggsBucketConfig creates a valid ParsedEggsBucketConfig for testing
func generateValidEggsBucketConfig(bucketType, provider string, repoCount int) *ParsedEggsBucketConfig {
	region := "ru-central1-a"
	if provider == "aws" {
		region = "us-east-1"
	}

	repos := make([]RepositoryInfo, repoCount)
	for i := 0; i < repoCount; i++ {
		repos[i] = RepositoryInfo{
			Name: fmt.Sprintf("repo-%d", i+1),
			GitLab: GitLabInfo{
				ProjectID:   10000 + i,
				TokenSecret: fmt.Sprintf("vault://gitlab/repo-%d-token", i+1),
			},
		}
	}

	// Generate provider-specific valid resources
	var cpu, memory, disk int

	if provider == "yandex" {
		// Yandex Cloud: CPU must be even (or 1), memory at least 1GB per CPU
		cpuOptions := []int{4, 6, 8}
		cpu = cpuOptions[rand.Intn(len(cpuOptions))]

		if bucketType == "serverless" {
			// Yandex Cloud Functions: specific memory sizes
			memoryOptions := []int{1024, 2048, 4096}
			memory = memoryOptions[rand.Intn(len(memoryOptions))]
		} else {
			// VM: at least 1GB per CPU
			memory = cpu*1024 + rand.Intn(cpu*1024) // 1-2GB per CPU
		}

		disk = 40 + rand.Intn(60) // 40-100GB
	} else {
		// AWS: more flexible
		cpu = 4 + rand.Intn(4) // 4-7 CPUs

		if bucketType == "serverless" {
			// AWS Lambda: 128MB to 10240MB
			memory = 1024 + rand.Intn(9216) // 1GB-10GB
		} else {
			// EC2: at least 512MB
			memory = 4096 + rand.Intn(12288) // 4GB-16GB
		}

		disk = 40 + rand.Intn(60) // 40-100GB
	}

	return &ParsedEggsBucketConfig{
		Name: fmt.Sprintf("test-bucket-%d", rand.Intn(10000)),
		Type: bucketType,
		Cloud: CloudInfo{
			Provider: provider,
			Region:   region,
		},
		Resources: ResourceInfo{
			CPU:    cpu,
			Memory: memory,
			Disk:   disk,
		},
		Runner: RunnerInfo{
			Tags:        []string{"docker", "linux", "microservices"},
			Concurrent:  5 + rand.Intn(15), // 5-20 concurrent jobs
			IdleTimeout: "15m",
		},
		Repositories: repos,
		Environment: map[string]string{
			"DOCKER_DRIVER": "overlay2",
			"SHARED_CACHE":  "s3://team-cache-bucket",
		},
	}
}

// validateVMConfigForSDK validates that a VMConfig can be used with cloud SDKs
func validateVMConfigForSDK(config *VMConfig, provider string) error {
	// Basic validation that would be performed by SDKs
	if config.EggName == "" {
		return fmt.Errorf("EggName is required")
	}

	if config.Cloud.Region == "" {
		return fmt.Errorf("Cloud.Region is required")
	}

	if config.Resources.CPU <= 0 {
		return fmt.Errorf("Resources.CPU must be positive")
	}

	if config.Resources.Memory <= 0 {
		return fmt.Errorf("Resources.Memory must be positive")
	}

	if config.Resources.Disk <= 0 {
		return fmt.Errorf("Resources.Disk must be positive")
	}

	if config.Runner.Concurrent <= 0 {
		return fmt.Errorf("Runner.Concurrent must be positive")
	}

	if config.GitLab.ProjectID <= 0 {
		return fmt.Errorf("GitLab.ProjectID must be positive")
	}

	if config.GitLab.TokenSecret == "" {
		return fmt.Errorf("GitLab.TokenSecret is required")
	}

	// Provider-specific validation
	switch provider {
	case "yandex":
		return validateYandexVMConfig(config)
	case "aws":
		return validateAWSVMConfig(config)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// validateServerlessConfigForSDK validates that a ServerlessConfig can be used with cloud SDKs
func validateServerlessConfigForSDK(config *ServerlessConfig, provider string) error {
	// Basic validation that would be performed by SDKs
	if config.EggName == "" {
		return fmt.Errorf("EggName is required")
	}

	if config.Cloud.Region == "" {
		return fmt.Errorf("Cloud.Region is required")
	}

	if config.Resources.CPU <= 0 {
		return fmt.Errorf("Resources.CPU must be positive")
	}

	if config.Resources.Memory <= 0 {
		return fmt.Errorf("Resources.Memory must be positive")
	}

	if config.Runner.Concurrent <= 0 {
		return fmt.Errorf("Runner.Concurrent must be positive")
	}

	if config.GitLab.ProjectID <= 0 {
		return fmt.Errorf("GitLab.ProjectID must be positive")
	}

	if config.GitLab.TokenSecret == "" {
		return fmt.Errorf("GitLab.TokenSecret is required")
	}

	// Serverless-specific validation
	if config.Timeout <= 0 {
		return fmt.Errorf("Timeout must be positive")
	}

	// Serverless runners have a maximum timeout of 60 minutes
	if config.Timeout > 60*time.Minute {
		return fmt.Errorf("Timeout exceeds maximum of 60 minutes")
	}

	// Provider-specific validation
	switch provider {
	case "yandex":
		return validateYandexServerlessConfig(config)
	case "aws":
		return validateAWSServerlessConfig(config)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// validateYandexVMConfig validates Yandex Cloud specific VM requirements
func validateYandexVMConfig(config *VMConfig) error {
	// Yandex Cloud specific validation

	// Region must be a valid Yandex Cloud zone
	validZones := []string{"ru-central1-a", "ru-central1-b", "ru-central1-c"}
	validZone := false
	for _, zone := range validZones {
		if config.Cloud.Region == zone {
			validZone = true
			break
		}
	}
	if !validZone {
		return fmt.Errorf("invalid Yandex Cloud zone: %s", config.Cloud.Region)
	}

	// CPU must be 2, 4, 6, 8, etc. (even numbers)
	if config.Resources.CPU%2 != 0 && config.Resources.CPU != 1 {
		return fmt.Errorf("Yandex Cloud CPU must be 1 or an even number, got %d", config.Resources.CPU)
	}

	// Memory must be at least 1GB per CPU core
	minMemory := config.Resources.CPU * 1024
	if config.Resources.Memory < minMemory {
		return fmt.Errorf("Yandex Cloud requires at least 1GB memory per CPU core (min %d MB for %d CPUs)", minMemory, config.Resources.CPU)
	}

	return nil
}

// validateAWSVMConfig validates AWS specific VM requirements
func validateAWSVMConfig(config *VMConfig) error {
	// AWS specific validation

	// Region must be a valid AWS region
	validRegions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-central-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
	}
	validRegion := false
	for _, region := range validRegions {
		if config.Cloud.Region == region {
			validRegion = true
			break
		}
	}
	if !validRegion {
		return fmt.Errorf("invalid AWS region: %s", config.Cloud.Region)
	}

	// Memory must be at least 512MB
	if config.Resources.Memory < 512 {
		return fmt.Errorf("AWS requires at least 512MB memory, got %d MB", config.Resources.Memory)
	}

	// Disk must be at least 8GB
	if config.Resources.Disk < 8 {
		return fmt.Errorf("AWS requires at least 8GB disk, got %d GB", config.Resources.Disk)
	}

	return nil
}

// validateYandexServerlessConfig validates Yandex Cloud specific serverless requirements
func validateYandexServerlessConfig(config *ServerlessConfig) error {
	// Yandex Cloud Functions specific validation

	// Memory must be in specific increments (128MB, 256MB, 512MB, 1GB, 2GB, 4GB)
	validMemorySizes := []int{128, 256, 512, 1024, 2048, 4096}
	validMemory := false
	for _, size := range validMemorySizes {
		if config.Resources.Memory == size {
			validMemory = true
			break
		}
	}
	if !validMemory {
		return fmt.Errorf("Yandex Cloud Functions memory must be one of %v MB, got %d MB", validMemorySizes, config.Resources.Memory)
	}

	// Timeout must not exceed 60 minutes for Yandex Cloud Functions (updated limit)
	if config.Timeout > 60*time.Minute {
		return fmt.Errorf("Yandex Cloud Functions timeout must not exceed 60 minutes, got %v", config.Timeout)
	}

	return nil
}

// validateAWSServerlessConfig validates AWS Lambda specific requirements
func validateAWSServerlessConfig(config *ServerlessConfig) error {
	// AWS Lambda specific validation

	// Memory must be between 128MB and 10240MB
	if config.Resources.Memory < 128 || config.Resources.Memory > 10240 {
		return fmt.Errorf("AWS Lambda memory must be between 128MB and 10240MB, got %d MB", config.Resources.Memory)
	}

	// Timeout must not exceed 60 minutes for AWS Lambda (updated limit)
	if config.Timeout > 60*time.Minute {
		return fmt.Errorf("AWS Lambda timeout must not exceed 60 minutes, got %v", config.Timeout)
	}

	return nil
}

// TestFlyToCloudSDKConversionWithParserIntegration tests the full pipeline from .fly file to deployment configs
// This validates the end-to-end conversion: .fly file → Parser → Converter → Deployment Config
// The deployment configs are then passed to MotherGoose, which uses OpenTofu to deploy runners.
func TestFlyToCloudSDKConversionWithParserIntegration(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("parsing .fly file and converting to SDK objects produces valid configurations",
		prop.ForAll(
			func(eggType, provider string) bool {
				// Generate a .fly configuration string
				flyConfig := generateFlyConfigString(eggType, provider)

				// Parse the .fly configuration
				p := parser.NewParser()
				parsed, err := p.Parse([]byte(flyConfig), "test.fly")
				if err != nil {
					t.Logf("Parse error: %v\nInput:\n%s", err, flyConfig)
					return false
				}

				// Find the egg block
				if len(parsed.Blocks) == 0 {
					t.Logf("No blocks found in parsed config")
					return false
				}

				eggBlock := &parsed.Blocks[0]
				if eggBlock.Type != "egg" {
					t.Logf("Expected 'egg' block, got '%s'", eggBlock.Type)
					return false
				}

				// Parse the egg block into ParsedEggConfig
				egg, err := ParseEgg(eggBlock)
				if err != nil {
					t.Logf("ParseEgg error: %v", err)
					return false
				}

				// Convert to deployment configuration
				converter := NewConverter()

				if eggType == "vm" {
					vmConfig, convErr := converter.EggToVMConfig(egg)
					if convErr != nil {
						t.Logf("Conversion error: %v", convErr)
						return false
					}

					// Validate that the VMConfig can be used with SDK
					if err := validateVMConfigForSDK(vmConfig, provider); err != nil {
						t.Logf("SDK validation error: %v", err)
						return false
					}
				} else {
					serverlessConfig, convErr := converter.EggToServerlessConfig(egg)
					if convErr != nil {
						t.Logf("Conversion error: %v", convErr)
						return false
					}

					// Validate that the ServerlessConfig can be used with SDK
					if err := validateServerlessConfigForSDK(serverlessConfig, provider); err != nil {
						t.Logf("SDK validation error: %v", err)
						return false
					}
				}

				return true
			},
			gen.OneConstOf("vm", "serverless"),
			gen.OneConstOf("yandex", "aws"),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// generateFlyConfigString generates a .fly configuration string for testing
func generateFlyConfigString(eggType, provider string) string {
	region := "ru-central1-a"
	if provider == "aws" {
		region = "us-east-1"
	}

	return fmt.Sprintf(`
egg "test-app" {
  type = %q

  cloud {
    provider = %q
    region   = %q
  }

  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }

  runner {
    tags = ["docker", "linux"]
    concurrent = 3
    idle_timeout = "10m"
  }

  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }

  environment {
    DOCKER_DRIVER = "overlay2"
    CUSTOM_VAR    = "value"
  }
}
`, eggType, provider, region)
}

// TestSDKClientCreation tests that SDK clients can be created for backend infrastructure deployment
// Note: These clients are for deploying MotherGoose/UglyFox infrastructure, not individual runners
func TestSDKClientCreation(t *testing.T) {
	ctx := context.Background()

	t.Run("AWS client creation for backend infrastructure", func(t *testing.T) {
		// Skip if AWS credentials are not available
		t.Skip("Skipping AWS client test - requires AWS credentials")

		egg := generateValidEggConfig("vm", "aws")
		converter := NewConverter()
		vmConfig, err := converter.EggToVMConfig(egg)
		if err != nil {
			t.Fatalf("Conversion error: %v", err)
		}

		// Validate config before attempting to create client
		if err := validateVMConfigForSDK(vmConfig, "aws"); err != nil {
			t.Fatalf("SDK validation error: %v", err)
		}

		// Attempt to create AWS client for backend infrastructure deployment
		_, err = NewAWSClient(ctx, vmConfig.Cloud.Region)
		if err != nil {
			t.Logf("AWS client creation error (expected without credentials): %v", err)
		}
	})

	t.Run("Yandex Cloud client creation for backend infrastructure", func(t *testing.T) {
		// Skip if Yandex Cloud credentials are not available
		t.Skip("Skipping Yandex Cloud client test - requires YC credentials")

		egg := generateValidEggConfig("vm", "yandex")
		converter := NewConverter()
		vmConfig, err := converter.EggToVMConfig(egg)
		if err != nil {
			t.Fatalf("Conversion error: %v", err)
		}

		// Validate config before attempting to create client
		if err := validateVMConfigForSDK(vmConfig, "yandex"); err != nil {
			t.Fatalf("SDK validation error: %v", err)
		}

		// Attempt to create Yandex Cloud client for backend infrastructure deployment
		_, err = NewYandexCloudClient(ctx)
		if err != nil {
			t.Logf("Yandex Cloud client creation error (expected without credentials): %v", err)
		}
	})
}
