package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/polar-gosling/gosling/internal/parser"
)

// Feature: gitops-runner-orchestration, Property 6: Egg Configuration Validation
// Validates: Requirements 3.6
func TestEggConfigurationValidation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("invalid .fly configuration files are rejected before committing",
		prop.ForAll(
			func(invalidationType string) bool {
				// Create a temporary directory for testing
				tempDir, err := os.MkdirTemp("", "validate-test-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tempDir)

				// Create Nest structure
				eggsDir := filepath.Join(tempDir, "Eggs", "test-app")
				if err := os.MkdirAll(eggsDir, 0755); err != nil {
					t.Logf("Failed to create Eggs directory: %v", err)
					return false
				}

				// Generate an invalid configuration based on invalidationType
				invalidConfig := generateInvalidConfig(invalidationType)
				configPath := filepath.Join(eggsDir, "config.fly")
				if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
					t.Logf("Failed to write config file: %v", err)
					return false
				}

				// Parse the configuration
				p := parser.NewParser()
				config, parseErr := p.ParseFile(configPath)

				// Some invalidations cause parse errors, others cause validation errors
				if parseErr != nil {
					// Parse error is acceptable for some invalid configurations
					// The important thing is that the error is detected
					return true
				}

				// Validate the configuration
				validationErr := validateConfig(config, configPath)

				// Invalid configurations should fail validation
				if validationErr == nil {
					t.Logf("Expected validation to fail for invalidation type: %s\nConfig:\n%s",
						invalidationType, invalidConfig)
					return false
				}

				// Validation error should be descriptive
				errMsg := validationErr.Error()
				if len(errMsg) == 0 {
					t.Logf("Validation error message is empty")
					return false
				}

				return true
			},
			genInvalidationType(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genInvalidationType generates different types of invalid configurations
func genInvalidationType() gopter.Gen {
	return gen.OneConstOf(
		"missing_type",             // Missing required 'type' attribute
		"invalid_type_value",       // type is not 'vm' or 'serverless'
		"missing_cloud_block",      // Missing required 'cloud' nested block
		"missing_resources_block",  // Missing required 'resources' nested block
		"missing_runner_block",     // Missing required 'runner' nested block
		"missing_gitlab_block",     // Missing required 'gitlab' nested block
		"invalid_provider",         // provider is not 'yandex' or 'aws'
		"missing_provider",         // Missing 'provider' in cloud block
		"missing_region",           // Missing 'region' in cloud block
		"invalid_cpu_range",        // cpu value out of valid range
		"invalid_memory_range",     // memory value out of valid range
		"invalid_disk_range",       // disk value out of valid range
		"missing_tags",             // Missing 'tags' in runner block
		"invalid_tags_type",        // tags is not a list
		"missing_concurrent",       // Missing 'concurrent' in runner block
		"invalid_concurrent_range", // concurrent value out of valid range
		"missing_project_id",       // Missing 'project_id' in gitlab block
		"invalid_project_id_type",  // project_id is not a number
		"missing_token_secret",     // Missing 'token_secret' in gitlab block
		"no_labels",                // Egg block has no labels
		"multiple_labels",          // Egg block has multiple labels
		"invalid_name_format",      // Egg name contains invalid characters
		"type_wrong_type",          // type attribute is a number instead of string
		"empty_config",             // Configuration file is empty
	)
}

// generateInvalidConfig creates an invalid .fly configuration based on the invalidation type
func generateInvalidConfig(invalidationType string) string {
	switch invalidationType {
	case "missing_type":
		return `
egg "test-app" {
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_type_value":
		return `
egg "test-app" {
  type = "container"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_cloud_block":
		return `
egg "test-app" {
  type = "vm"
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_resources_block":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_runner_block":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_gitlab_block":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
}
`

	case "invalid_provider":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "gcp"
    region   = "us-central1"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_provider":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_region":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_cpu_range":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 256
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_memory_range":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 1000000
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_disk_range":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20000
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_tags":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_tags_type":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = "docker"
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_concurrent":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_concurrent_range":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 200
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_project_id":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_project_id_type":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = "12345"
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "missing_token_secret":
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
  }
}
`

	case "no_labels":
		return `
egg {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "multiple_labels":
		return `
egg "test-app" "extra-label" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "invalid_name_format":
		return `
egg "test app!" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "type_wrong_type":
		return `
egg "test-app" {
  type = 123
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`

	case "empty_config":
		return ""

	default:
		// Return a valid configuration as fallback
		return `
egg "test-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 2
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`
	}
}
