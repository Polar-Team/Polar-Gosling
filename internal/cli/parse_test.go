package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/polar-gosling/gosling/internal/parser"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		configType  string
		expectError bool
	}{
		{
			name: "parse egg config",
			content: `
egg "my-app" {
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
}
`,
			configType:  "egg",
			expectError: false,
		},
		{
			name: "parse job config",
			content: `
job "rotate-secrets" {
  schedule = "0 2 * * *"
  script = "#!/bin/bash\necho test"

  runner {
    type = "vm"
    tags = ["privileged"]
  }
}
`,
			configType:  "job",
			expectError: false,
		},
		{
			name: "parse uglyfox config",
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
			configType:  "uglyfox",
			expectError: false,
		},
		{
			name: "parse eggsbucket config",
			content: `
eggsbucket "microservices-team" {
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
    tags = ["docker", "linux", "microservices"]
    concurrent = 10
  }

  repositories {
    repo "auth-service" {
      gitlab {
        project_id = 12345
        server_name = "gitlab.com"
        token_secret = "yc-lockbox://gitlab/auth-token"
      }
    }

    repo "api-gateway" {
      gitlab {
        project_id = 12346
        server_name = "gitlab.com"
        token_secret = "yc-lockbox://gitlab/api-token"
      }
    }
  }
}
`,
			configType:  "eggsbucket",
			expectError: false,
		},
		{
			name: "parse mothergoose config",
			content: `
mothergoose {
  api_gateway {
    name = "polar-gosling-api"
  }

  fastapi_app {
    name = "mothergoose-api"
    runtime = "python312"
    memory = 512
  }

  celery_workers {
    name = "mothergoose-celery"
    runtime = "python312"
    memory = 1024
  }

  uglyfox_workers {
    name = "uglyfox-celery"
    runtime = "python312"
    memory = 512
  }

  message_queues {
    webhook_queue {
      name = "mothergoose-webhooks"
    }
  }

  triggers {
    git_sync {
      name = "git-sync-trigger"
      schedule = "*/5 * * * *"
    }
  }

  database {
    type = "ydb"
    name = "polar-gosling-db"
  }

  storage {
    state_bucket {
      name = "polar-gosling-state"
    }
  }

  service_accounts {
    mothergoose {
      name = "mothergoose-sa"
    }
  }
}
`,
			configType:  "mothergoose",
			expectError: false,
		},
		{
			name: "wrong type specified",
			content: `
egg "my-app" {
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
}
`,
			configType:  "job",
			expectError: true,
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

			// Parse the file
			config, err := parser.ParseAndValidate(tmpFile)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Parse failed: %v", err)
				}
				return
			}

			// Validate type if specified
			if tt.configType != "" {
				err := validateConfigType(config, tt.configType)
				if tt.expectError && err == nil {
					t.Fatal("Expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if tt.expectError {
					return
				}
			}

			// Convert to JSON
			jsonData := configToJSON(config)

			// Verify JSON structure
			if jsonData == nil {
				t.Fatal("JSON data is nil")
			}

			blocks, ok := jsonData["blocks"].([]map[string]interface{})
			if !ok {
				t.Fatal("blocks field is not a slice of maps")
			}

			if len(blocks) == 0 {
				t.Fatal("No blocks in JSON output")
			}

			// Verify first block has expected type
			firstBlock := blocks[0]
			blockType, ok := firstBlock["type"].(string)
			if !ok {
				t.Fatal("Block type is not a string")
			}

			if blockType != tt.configType {
				t.Errorf("Expected block type %q, got %q", tt.configType, blockType)
			}

			// For egg, job, eggsbucket, and mothergoose, we expect labels (except uglyfox and mothergoose)
			if tt.configType != "uglyfox" && tt.configType != "mothergoose" {
				labels, ok := firstBlock["labels"].([]string)
				if !ok {
					t.Fatal("Labels field is not a string slice")
				}
				if len(labels) == 0 {
					t.Error("Expected at least one label")
				}
			}

			// Verify JSON can be marshaled
			jsonBytes, err := json.Marshal(jsonData)
			if err != nil {
				t.Fatalf("Failed to marshal JSON: %v", err)
			}

			// Verify JSON can be unmarshaled
			var unmarshaled map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}
		})
	}
}

func TestValueToJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    *parser.Value
		expected interface{}
	}{
		{
			name: "string value",
			value: &parser.Value{
				Type: parser.StringType,
				Raw:  "test-string",
			},
			expected: "test-string",
		},
		{
			name: "number value",
			value: &parser.Value{
				Type: parser.NumberType,
				Raw:  float64(42),
			},
			expected: float64(42),
		},
		{
			name: "bool value",
			value: &parser.Value{
				Type: parser.BoolType,
				Raw:  true,
			},
			expected: true,
		},
		{
			name: "list value",
			value: &parser.Value{
				Type: parser.ListType,
				Raw: []parser.Value{
					{Type: parser.StringType, Raw: "item1"},
					{Type: parser.StringType, Raw: "item2"},
				},
			},
			expected: []interface{}{"item1", "item2"},
		},
		{
			name: "map value",
			value: &parser.Value{
				Type: parser.MapType,
				Raw: map[string]parser.Value{
					"key1": {Type: parser.StringType, Raw: "value1"},
					"key2": {Type: parser.NumberType, Raw: float64(123)},
				},
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": float64(123),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valueToJSON(tt.value)

			// Compare based on type
			switch expected := tt.expected.(type) {
			case string:
				if result != expected {
					t.Errorf("Expected %q, got %q", expected, result)
				}
			case float64:
				if result != expected {
					t.Errorf("Expected %v, got %v", expected, result)
				}
			case bool:
				if result != expected {
					t.Errorf("Expected %v, got %v", expected, result)
				}
			case []interface{}:
				resultList, ok := result.([]interface{})
				if !ok {
					t.Fatalf("Result is not a slice")
				}
				if len(resultList) != len(expected) {
					t.Errorf("Expected list length %d, got %d", len(expected), len(resultList))
				}
			case map[string]interface{}:
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Fatalf("Result is not a map")
				}
				if len(resultMap) != len(expected) {
					t.Errorf("Expected map length %d, got %d", len(expected), len(resultMap))
				}
			}
		})
	}
}

func TestBlockToJSON(t *testing.T) {
	block := &parser.Block{
		Type:   "egg",
		Labels: []string{"my-app"},
		Attributes: map[string]parser.Value{
			"type": {
				Type: parser.StringType,
				Raw:  "vm",
			},
		},
		Blocks: []parser.Block{
			{
				Type:   "cloud",
				Labels: []string{},
				Attributes: map[string]parser.Value{
					"provider": {
						Type: parser.StringType,
						Raw:  "yandex",
					},
				},
			},
		},
	}

	result := blockToJSON(block)

	// Verify type
	if result["type"] != "egg" {
		t.Errorf("Expected type 'egg', got %v", result["type"])
	}

	// Verify labels
	labels, ok := result["labels"].([]string)
	if !ok {
		t.Fatal("Labels is not a string slice")
	}
	if len(labels) != 1 || labels[0] != "my-app" {
		t.Errorf("Expected labels ['my-app'], got %v", labels)
	}

	// Verify attributes
	attrs, ok := result["attributes"].(map[string]interface{})
	if !ok {
		t.Fatal("Attributes is not a map")
	}
	if attrs["type"] != "vm" {
		t.Errorf("Expected type attribute 'vm', got %v", attrs["type"])
	}

	// Verify nested blocks
	blocks, ok := result["blocks"].([]map[string]interface{})
	if !ok {
		t.Fatal("Blocks is not a slice of maps")
	}
	if len(blocks) != 1 {
		t.Errorf("Expected 1 nested block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "cloud" {
		t.Errorf("Expected nested block type 'cloud', got %v", blocks[0]["type"])
	}
}

func TestConfigToJSON(t *testing.T) {
	config := &parser.Config{
		Blocks: []parser.Block{
			{
				Type:   "egg",
				Labels: []string{"my-app"},
				Attributes: map[string]parser.Value{
					"type": {
						Type: parser.StringType,
						Raw:  "vm",
					},
				},
			},
		},
	}

	result := configToJSON(config)

	// Verify blocks field exists
	blocks, ok := result["blocks"].([]map[string]interface{})
	if !ok {
		t.Fatal("blocks field is not a slice of maps")
	}

	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}

	if blocks[0]["type"] != "egg" {
		t.Errorf("Expected block type 'egg', got %v", blocks[0]["type"])
	}
}

func TestValidateConfigType(t *testing.T) {
	tests := []struct {
		name         string
		config       *parser.Config
		expectedType string
		expectError  bool
	}{
		{
			name: "valid egg type",
			config: &parser.Config{
				Blocks: []parser.Block{
					{Type: "egg", Labels: []string{"my-app"}},
				},
			},
			expectedType: "egg",
			expectError:  false,
		},
		{
			name: "invalid type mismatch",
			config: &parser.Config{
				Blocks: []parser.Block{
					{Type: "egg", Labels: []string{"my-app"}},
				},
			},
			expectedType: "job",
			expectError:  true,
		},
		{
			name: "empty config",
			config: &parser.Config{
				Blocks: []parser.Block{},
			},
			expectedType: "egg",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigType(tt.config, tt.expectedType)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParseCommandOutput(t *testing.T) {
	// Create a temporary file with egg config
	content := `
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
    tags = ["docker", "linux"]
    concurrent = 3
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

	// Parse and convert to JSON
	config, err := parser.ParseAndValidate(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	jsonData := configToJSON(config)

	// Marshal to JSON bytes
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(jsonData); err != nil {
		t.Fatalf("Failed to encode JSON: %v", err)
	}

	// Verify JSON is valid
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify structure
	blocks, ok := unmarshaled["blocks"].([]interface{})
	if !ok {
		t.Fatal("blocks field is not a slice")
	}

	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}

	firstBlock, ok := blocks[0].(map[string]interface{})
	if !ok {
		t.Fatal("First block is not a map")
	}

	if firstBlock["type"] != "egg" {
		t.Errorf("Expected type 'egg', got %v", firstBlock["type"])
	}

	// Verify nested blocks exist
	nestedBlocks, ok := firstBlock["blocks"].([]interface{})
	if !ok {
		t.Fatal("Nested blocks field is not a slice")
	}

	// Should have cloud, resources, runner, gitlab blocks
	if len(nestedBlocks) < 4 {
		t.Errorf("Expected at least 4 nested blocks, got %d", len(nestedBlocks))
	}
}
