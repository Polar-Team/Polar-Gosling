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
    server_name = "example.com"
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
    server_name = "example.com"
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
    server_name = "example.com"
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
    server_name = "example.com"
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
    server_name = "example.com"
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

  runners_condition "default" {
    eggs_entities = ["Egg1", "EggsBucket2"]

    apex {
      max_count = 10
      min_count = 2
      cpu_threshold = 80
      memory_threshold = 70
    }

    nadir {
      max_count = 5
      min_count = 0
      idle_timeout = "30m"
    }
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

  runners_condition "default" {
    eggs_entities = ["Egg1"]

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

  runners_condition "default" {
    eggs_entities = ["Egg1"]

    apex {
      max_count = 10
      min_count = 2
    }

    nadir {
      max_count = 5
      min_count = 0
      idle_timeout = "30m"
    }
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
    server_name = "example.com"
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

// ---------------------------------------------------------------------------
// MotherGoose block validation tests
// ---------------------------------------------------------------------------

// validMotherGooseConfig is the canonical valid mothergoose .fly config used
// as a baseline for all mothergoose validator tests.
const validMotherGooseConfig = `
mothergoose "yandex-prod" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1gxxxxxxxxxxxxxxx"
    yc_cloud_id  = "b1gyyyyyyyyyyyyyyy"
  }

  api_gateway {
    name            = "polar-gosling-gw"
    description     = "Main API gateway"
    service_account = "mg-sa"
  }

  fastapi_app {
    name              = "mg-fastapi"
    image             = "ghcr.io/polar-team/mothergoose:latest"
    memory            = 512
    cores             = 1
    execution_timeout = "60s"
    concurrency       = 4
    service_account   = "mg-sa"
  }

  celery_workers {
    name              = "mg-celery"
    image             = "ghcr.io/polar-team/mothergoose:latest"
    memory            = 1024
    cores             = 2
    execution_timeout = "300s"
    concurrency       = 2
    service_account   = "mg-sa"
  }

  git_sync_trigger {
    schedule        = "*/5 * * * *"
    service_account = "mg-sa"
  }

  mothergoose_queues {
    task_queue {
      name               = "mg-tasks"
      visibility_timeout = 30
      message_retention  = 86400
    }
    dlq {
      name              = "mg-tasks-dlq"
      message_retention = 86400
    }
  }

  database {
    name            = "polar-gosling-db"
    type            = "ydb"
    serverless_mode = true
  }

  storage {
    bucket_name = "polar-gosling-storage"
    region      = "ru-central1"
  }

  service_account "mg-sa" {
    description = "MotherGoose service account"
    roles       = ["lockbox.payloadViewer", "ydb.editor"]
  }
}
`

func TestValidateMotherGooseConfig(t *testing.T) {
	p := NewParser()
	config, err := p.Parse([]byte(validMotherGooseConfig), "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if !result.IsValid() {
		t.Errorf("Expected valid mothergoose config, got errors: %v", result.Error())
	}
}

func TestValidateMotherGooseMissingAPIGateway(t *testing.T) {
	content := `
mothergoose "yandex-prod" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1gxxxxxxxxxxxxxxx"
    yc_cloud_id  = "b1gyyyyyyyyyyyyyyy"
  }

  fastapi_app {
    name  = "mg-fastapi"
    image = "ghcr.io/polar-team/mothergoose:latest"
  }

  celery_workers {
    name  = "mg-celery"
    image = "ghcr.io/polar-team/mothergoose:latest"
  }

  uglyfox_workers {
    name  = "uf-worker"
    image = "ghcr.io/polar-team/uglyfox:latest"
  }

  message_queues {
    name = "mg-tasks"
  }

  triggers {
    name     = "git-sync"
    schedule = "*/5 * * * *"
    endpoint = "/internal/sync-git"
  }

  database {
    name = "polar-gosling-db"
    type = "ydb"
  }

  storage {
    bucket_name = "polar-gosling-storage"
  }

  service_accounts {
    name  = "mg-sa"
    roles = ["lockbox.payloadViewer"]
  }
}
`
	p := NewParser()
	config, err := p.Parse([]byte(content), "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := NewValidator(config).Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail when api_gateway block is missing")
	}

	found := false
	for _, e := range result.Errors {
		if e.Field == "api_gateway" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for missing 'api_gateway' field")
	}
}

func TestValidateMotherGooseMissingDatabase(t *testing.T) {
	content := `
mothergoose "yandex-prod" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1gxxxxxxxxxxxxxxx"
    yc_cloud_id  = "b1gyyyyyyyyyyyyyyy"
  }

  api_gateway {
    name = "polar-gosling-gw"
  }

  fastapi_app {
    name  = "mg-fastapi"
    image = "ghcr.io/polar-team/mothergoose:latest"
  }

  celery_workers {
    name  = "mg-celery"
    image = "ghcr.io/polar-team/mothergoose:latest"
  }

  uglyfox_workers {
    name  = "uf-worker"
    image = "ghcr.io/polar-team/uglyfox:latest"
  }

  message_queues {
    name = "mg-tasks"
  }

  triggers {
    name     = "git-sync"
    schedule = "*/5 * * * *"
    endpoint = "/internal/sync-git"
  }

  storage {
    bucket_name = "polar-gosling-storage"
  }

  service_accounts {
    name  = "mg-sa"
    roles = ["lockbox.payloadViewer"]
  }
}
`
	p := NewParser()
	config, err := p.Parse([]byte(content), "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := NewValidator(config).Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail when database block is missing")
	}

	found := false
	for _, e := range result.Errors {
		if e.Field == "database" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for missing 'database' field")
	}
}

func TestValidateMotherGooseMissingStorage(t *testing.T) {
	content := `
mothergoose "yandex-prod" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1gxxxxxxxxxxxxxxx"
    yc_cloud_id  = "b1gyyyyyyyyyyyyyyy"
  }

  api_gateway {
    name = "gw"
  }

  fastapi_app {
    name  = "mg-fastapi"
    image = "img:latest"
  }

  celery_workers {
    name  = "mg-celery"
    image = "img:latest"
  }

  git_sync_trigger {
    schedule = "*/5 * * * *"
  }

  mothergoose_queues {
    task_queue {
      name = "mg-tasks"
    }
    dlq {
      name = "mg-tasks-dlq"
    }
  }

  database {
    name = "db"
    type = "ydb"
  }

  service_account "mg-sa" {
    roles = ["lockbox.payloadViewer"]
  }
}
`
	p := NewParser()
	config, err := p.Parse([]byte(content), "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := NewValidator(config).Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail when storage block is missing")
	}

	found := false
	for _, e := range result.Errors {
		if e.Field == "storage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for missing 'storage' field")
	}
}

func TestValidateMotherGooseMissingServiceAccounts(t *testing.T) {
	content := `
mothergoose "yandex-prod" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1gxxxxxxxxxxxxxxx"
    yc_cloud_id  = "b1gyyyyyyyyyyyyyyy"
  }

  api_gateway {
    name = "gw"
  }

  fastapi_app {
    name  = "mg-fastapi"
    image = "img:latest"
  }

  celery_workers {
    name  = "mg-celery"
    image = "img:latest"
  }

  git_sync_trigger {
    schedule = "*/5 * * * *"
  }

  mothergoose_queues {
    task_queue {
      name = "mg-tasks"
    }
    dlq {
      name = "mg-tasks-dlq"
    }
  }

  database {
    name = "db"
    type = "ydb"
  }

  storage {
    bucket_name = "polar-gosling-storage"
  }
}
`
	p := NewParser()
	config, err := p.Parse([]byte(content), "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	result := NewValidator(config).Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail when service_accounts block is missing")
	}

	found := false
	for _, e := range result.Errors {
		if e.Field == "service_account" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validation error for missing 'service_account' field")
	}
}
