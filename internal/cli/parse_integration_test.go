package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestParseIntegration tests the parse command with real .fly files
func TestParseIntegration(t *testing.T) {
	// Skip if gosling binary is not built
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go not available, skipping integration test")
	}

	tests := []struct {
		name       string
		content    string
		configType string
		wantError  bool
	}{
		{
			name: "egg config with all fields",
			content: `
egg "production-app" {
  type = "vm"

  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }

  resources {
    cpu    = 4
    memory = 8192
    disk   = 50
  }

  runner {
    tags = ["docker", "linux", "production"]
    concurrent = 5
    idle_timeout = "15m"
  }

  gitlab {
    project_id = 98765
    server_name = "gitlab.com"
    token_secret = "yc-lockbox://gitlab/runner-token"
  }

  environment {
    DOCKER_DRIVER = "overlay2"
    CI_DEBUG_TRACE = "false"
  }
}
`,
			configType: "egg",
			wantError:  false,
		},
		{
			name: "job config",
			content: `
job "rotate-secrets" {
  schedule = "0 2 * * *"
  script = <<-EOT
    #!/bin/bash
    echo "Rotating secrets..."
    # Secret rotation logic here
  EOT

  runner {
    type = "vm"
    tags = ["privileged", "secrets"]
    concurrent = 1
  }
}
`,
			configType: "job",
			wantError:  false,
		},
		{
			name: "uglyfox config",
			content: `
uglyfox {
  pruning {
    failed_threshold = 3
    check_interval = "5m"
    max_age = "24h"
  }

  runners_condition "default" {
    eggs_entities = ["my-app", "other-app"]

    apex {
      max_count = 5
      min_count = 1
    }

    nadir {
      max_count = 3
      min_count = 0
      idle_timeout = "30m"
    }
  }
}
`,
			configType: "uglyfox",
			wantError:  false,
		},
		{
			name: "eggsbucket config",
			content: `
eggsbucket "microservices-team" {
  type = "vm"

  cloud {
    provider = "aws"
    region   = "us-east-1"
  }

  resources {
    cpu    = 8
    memory = 16384
    disk   = 100
  }

  runner {
    tags = ["docker", "linux", "microservices"]
    concurrent = 10
  }

  repositories {
    repo "auth-service" {
      gitlab {
        project_id = 12345
        server_name = "gitlab.company.com"
        token_secret = "aws-sm://gitlab/auth-token"
      }
    }

    repo "api-gateway" {
      gitlab {
        project_id = 12346
        server_name = "gitlab.company.com"
        token_secret = "aws-sm://gitlab/api-token"
      }
    }

    repo "user-service" {
      gitlab {
        project_id = 12347
        server_name = "gitlab.company.com"
        token_secret = "aws-sm://gitlab/user-token"
      }
    }
  }
}
`,
			configType: "eggsbucket",
			wantError:  false,
		},
		{
			name: "mothergoose config",
			content: `
mothergoose {
  api_gateway {
    name = "polar-gosling-api"
    openapi_spec = "openapi.yaml"
  }

  fastapi_app {
    name = "mothergoose-api"
    runtime = "python312"
    memory = 512
    timeout = 30
  }

  celery_workers {
    name = "mothergoose-celery"
    runtime = "python312"
    memory = 1024
    timeout = 300
  }

  uglyfox_workers {
    name = "uglyfox-celery"
    runtime = "python312"
    memory = 512
    timeout = 180
  }

  message_queues {
    webhook_queue {
      name = "mothergoose-webhooks"
      visibility_timeout = 300
    }

    uglyfox_queue {
      name = "uglyfox-tasks"
      visibility_timeout = 180
    }
  }

  triggers {
    git_sync {
      name = "git-sync-trigger"
      schedule = "*/5 * * * *"
      endpoint = "/internal/sync-git"
    }

    health_check {
      name = "uglyfox-health-trigger"
      schedule = "*/10 * * * *"
      endpoint = "/internal/uglyfox/health-check"
    }
  }

  database {
    type = "ydb"
    name = "polar-gosling-db"
    mode = "serverless"
  }

  storage {
    state_bucket {
      name = "polar-gosling-state"
      versioning = true
    }

    binary_bucket {
      name = "polar-gosling-binaries"
      versioning = false
    }
  }

  service_accounts {
    mothergoose {
      name = "mothergoose-sa"
      roles = ["lockbox.payloadViewer", "ydb.editor"]
    }

    uglyfox {
      name = "uglyfox-sa"
      roles = ["lockbox.payloadViewer", "ydb.viewer"]
    }
  }
}
`,
			configType: "mothergoose",
			wantError:  false,
		},
		{
			name: "invalid syntax",
			content: `
egg "broken" {
  type = "vm"
  cloud {
    provider = "yandex"
    # Missing closing brace
`,
			configType: "egg",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "config.fly")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Run parse command programmatically
			var stdout, stderr bytes.Buffer

			// Save original stdout/stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr

			// Create pipes
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()

			os.Stdout = wOut
			os.Stderr = wErr

			// Run the command
			parseType = tt.configType
			err := runParse(parseCmd, []string{tmpFile})

			// Restore stdout/stderr
			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read output
			stdout.ReadFrom(rOut)
			stderr.ReadFrom(rErr)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v\nStderr: %s", err, stderr.String())
			}

			// Verify JSON output
			var jsonData map[string]interface{}
			if err := json.Unmarshal(stdout.Bytes(), &jsonData); err != nil {
				t.Fatalf("Failed to unmarshal JSON output: %v\nOutput: %s", err, stdout.String())
			}

			// Verify structure
			blocks, ok := jsonData["blocks"].([]interface{})
			if !ok {
				t.Fatal("blocks field is not a slice")
			}

			if len(blocks) == 0 {
				t.Fatal("No blocks in JSON output")
			}

			firstBlock, ok := blocks[0].(map[string]interface{})
			if !ok {
				t.Fatal("First block is not a map")
			}

			// Verify block type
			blockType, ok := firstBlock["type"].(string)
			if !ok {
				t.Fatal("Block type is not a string")
			}

			if blockType != tt.configType {
				t.Errorf("Expected block type %q, got %q", tt.configType, blockType)
			}

			// Verify labels exist (except for uglyfox and mothergoose)
			if tt.configType != "uglyfox" && tt.configType != "mothergoose" {
				labels, ok := firstBlock["labels"].([]interface{})
				if !ok {
					t.Fatal("Labels field is not a slice")
				}

				if len(labels) == 0 {
					t.Error("Expected at least one label")
				}
			}

			// Verify attributes or nested blocks exist
			hasAttributes := firstBlock["attributes"] != nil
			hasBlocks := firstBlock["blocks"] != nil

			if !hasAttributes && !hasBlocks {
				t.Error("Block has neither attributes nor nested blocks")
			}
		})
	}
}

// TestParseIntegrationWithSecretURIs tests parsing configs with secret URIs
func TestParseIntegrationWithSecretURIs(t *testing.T) {
	content := `
egg "secure-app" {
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
    tags = ["docker", "linux"]
    concurrent = 3
  }

  gitlab {
    project_id = 12345
    server_name = "gitlab.com"
    token_secret = "yc-lockbox://gitlab/runner-token"
  }

  environment {
    API_KEY = "aws-sm://api-keys/production"
    DB_PASSWORD = "vault://database/credentials"
  }
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.fly")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Run parse command
	var stdout bytes.Buffer

	rOut, wOut, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = wOut

	parseType = "egg"
	err := runParse(parseCmd, []string{tmpFile})

	wOut.Close()
	os.Stdout = oldStdout
	stdout.ReadFrom(rOut)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify JSON output
	var jsonData map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &jsonData); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify secret URIs are preserved
	blocks := jsonData["blocks"].([]interface{})
	firstBlock := blocks[0].(map[string]interface{})
	nestedBlocks := firstBlock["blocks"].([]interface{})

	// Find gitlab block
	var gitlabBlock map[string]interface{}
	for _, block := range nestedBlocks {
		b := block.(map[string]interface{})
		if b["type"] == "gitlab" {
			gitlabBlock = b
			break
		}
	}

	if gitlabBlock == nil {
		t.Fatal("GitLab block not found")
	}

	attrs := gitlabBlock["attributes"].(map[string]interface{})
	tokenSecret := attrs["token_secret"].(string)

	if tokenSecret != "yc-lockbox://gitlab/runner-token" {
		t.Errorf("Expected secret URI to be preserved, got %q", tokenSecret)
	}
}

// TestParseIntegrationComplexNesting tests parsing with deeply nested blocks
func TestParseIntegrationComplexNesting(t *testing.T) {
	content := `
egg "complex-app" {
  type = "vm"

  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
    
    network {
      vpc_id = "vpc-12345"
      subnet_id = "subnet-67890"
    }
  }

  resources {
    cpu    = 4
    memory = 8192
    disk   = 50
    
    limits {
      max_cpu = 8
      max_memory = 16384
    }
  }

  runner {
    tags = ["docker", "linux"]
    concurrent = 5
    
    cache {
      type = "s3"
      bucket = "runner-cache"
    }
  }

  gitlab {
    project_id = 12345
    server_name = "gitlab.com"
    token_secret = "yc-lockbox://gitlab/runner-token"
  }
}
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.fly")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Run parse command
	var stdout bytes.Buffer

	rOut, wOut, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = wOut

	parseType = "egg"
	err := runParse(parseCmd, []string{tmpFile})

	wOut.Close()
	os.Stdout = oldStdout
	stdout.ReadFrom(rOut)

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify JSON output
	var jsonData map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &jsonData); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify nested structure is preserved
	blocks := jsonData["blocks"].([]interface{})
	firstBlock := blocks[0].(map[string]interface{})
	nestedBlocks := firstBlock["blocks"].([]interface{})

	// Find cloud block
	var cloudBlock map[string]interface{}
	for _, block := range nestedBlocks {
		b := block.(map[string]interface{})
		if b["type"] == "cloud" {
			cloudBlock = b
			break
		}
	}

	if cloudBlock == nil {
		t.Fatal("Cloud block not found")
	}

	// Verify network block exists within cloud block
	cloudNestedBlocks, ok := cloudBlock["blocks"].([]interface{})
	if !ok || len(cloudNestedBlocks) == 0 {
		t.Fatal("Cloud block should have nested blocks")
	}

	networkBlock := cloudNestedBlocks[0].(map[string]interface{})
	if networkBlock["type"] != "network" {
		t.Errorf("Expected nested block type 'network', got %v", networkBlock["type"])
	}
}
