package parser

import (
	"testing"
)

func TestValidateEggConfig(t *testing.T) {
	content := []byte(`
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
    idle_timeout = "10m"
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if !result.IsValid() {
		t.Errorf("Validation failed: %v", result.Error())
	}
}

func TestValidateEggConfigMissingType(t *testing.T) {
	content := []byte(`
egg "my-app" {
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
    token_secret = "vault://gitlab/runner-token"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for missing type attribute")
	}

	// Check that error mentions 'type'
	found := false
	for _, err := range result.Errors {
		if err.Field == "type" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for 'type' field")
	}
}

func TestValidateEggConfigInvalidType(t *testing.T) {
	content := []byte(`
egg "my-app" {
  type = "invalid"
  
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
    token_secret = "vault://gitlab/runner-token"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for invalid type value")
	}
}

func TestValidateEggConfigInvalidProvider(t *testing.T) {
	content := []byte(`
egg "my-app" {
  type = "vm"
  
  cloud {
    provider = "invalid"
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
    token_secret = "vault://gitlab/runner-token"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for invalid provider value")
	}
}

func TestValidateEggConfigResourceOutOfRange(t *testing.T) {
	content := []byte(`
egg "my-app" {
  type = "vm"
  
  cloud {
    provider = "yandex"
    region   = "ru-central1-a"
  }
  
  resources {
    cpu    = 200
    memory = 4096
    disk   = 20
  }
  
  runner {
    tags = ["docker", "linux"]
    concurrent = 3
  }
  
  gitlab {
    project_id = 12345
    token_secret = "vault://gitlab/runner-token"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for CPU out of range")
	}
}

func TestValidateJobConfig(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule = "0 2 * * *"
  
  runner {
    type = "vm"
    tags = ["privileged"]
  }
  
  script = "#!/bin/bash\necho 'test'"
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if !result.IsValid() {
		t.Errorf("Validation failed: %v", result.Error())
	}
}

func TestValidateJobConfigMissingSchedule(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  runner {
    type = "vm"
    tags = ["privileged"]
  }
  
  script = "#!/bin/bash\necho 'test'"
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for missing schedule")
	}
}

func TestValidateJobConfigInvalidCron(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule = "invalid cron"
  
  runner {
    type = "vm"
    tags = ["privileged"]
  }
  
  script = "#!/bin/bash\necho 'test'"
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for invalid cron expression")
	}
}

func TestValidateUglyFoxConfig(t *testing.T) {
	content := []byte(`
uglyfox {
  pruning {
    failed_threshold = 3
    max_age = "24h"
    check_interval = "5m"
  }
  
  apex {
    max_count = 10
    min_count = 2
  }
  
  nadir {
    max_count = 5
    min_count = 0
    idle_timeout = "30m"
  }
  
  policies {
    rule "terminate_old_failed" {
      condition = "failed_count >= 3 AND age > 1h"
      action    = "terminate"
    }
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if !result.IsValid() {
		t.Errorf("Validation failed: %v", result.Error())
	}
}

func TestValidateUglyFoxConfigMinGreaterThanMax(t *testing.T) {
	content := []byte(`
uglyfox {
  pruning {
    failed_threshold = 3
    max_age = "24h"
    check_interval = "5m"
  }
  
  apex {
    max_count = 5
    min_count = 10
  }
  
  nadir {
    max_count = 5
    min_count = 0
    idle_timeout = "30m"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail when min_count > max_count")
	}
}

func TestValidateUglyFoxConfigInvalidAction(t *testing.T) {
	content := []byte(`
uglyfox {
  pruning {
    failed_threshold = 3
    max_age = "24h"
    check_interval = "5m"
  }
  
  apex {
    max_count = 10
    min_count = 2
  }
  
  nadir {
    max_count = 5
    min_count = 0
    idle_timeout = "30m"
  }
  
  policies {
    rule "test_rule" {
      condition = "failed_count >= 3"
      action    = "invalid_action"
    }
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for invalid action")
	}
}

func TestValidateInvalidEggName(t *testing.T) {
	content := []byte(`
egg "123-invalid" {
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
    token_secret = "vault://gitlab/runner-token"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for egg name starting with number")
	}
}
