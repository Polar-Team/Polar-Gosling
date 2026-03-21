package parser

import (
	"testing"
)

// --- egg ---

func TestValidateEggValid(t *testing.T) {
	content := []byte(`
egg "my-service" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_token_secret   = "yc-lockbox://abc123def456/runner-token"
  gitlab_webhook_secret = "yc-lockbox://abc123def456/webhook-secret"
  git_repo_url_secret   = "yc-lockbox://abc123def456/repo-url"
  cloud_provider        = "yandex"
  region                = "ru-central1"
  runner_type           = "serverless"
  tags                  = ["docker", "linux"]
  cpu                   = 1
  memory                = "512MB"
  max_concurrent_jobs   = 2
}
`)
	assertValid(t, content)
}

func TestValidateEggMissingRequired(t *testing.T) {
	// Missing gitlab_token_secret, git_repo_url_secret, runner_type
	content := []byte(`
egg "my-service" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_webhook_secret = "yc-lockbox://abc123/webhook"
  cloud_provider        = "yandex"
  region                = "ru-central1"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "gitlab_token_secret")
	assertHasErrorField(t, result, "git_repo_url_secret")
	assertHasErrorField(t, result, "runner_type")
}

func TestValidateEggInvalidCloudProvider(t *testing.T) {
	content := []byte(`
egg "my-service" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_token_secret   = "vault://gitlab/token"
  gitlab_webhook_secret = "vault://gitlab/webhook"
  git_repo_url_secret   = "vault://gitlab/repo-url"
  cloud_provider        = "gcp"
  region                = "us-central1"
  runner_type           = "serverless"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "cloud_provider")
}

func TestValidateEggInvalidRunnerType(t *testing.T) {
	content := []byte(`
egg "my-service" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_token_secret   = "aws-sm://prod/token/value"
  gitlab_webhook_secret = "aws-sm://prod/webhook/value"
  git_repo_url_secret   = "aws-sm://prod/repo/value"
  cloud_provider        = "aws"
  region                = "us-east-1"
  runner_type           = "dedicated"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "runner_type")
}

func TestValidateEggInvalidSecretURI(t *testing.T) {
	content := []byte(`
egg "my-service" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_token_secret   = "plain-text-token"
  gitlab_webhook_secret = "yc-lockbox://abc/webhook"
  git_repo_url_secret   = "yc-lockbox://abc/repo"
  cloud_provider        = "yandex"
  region                = "ru-central1"
  runner_type           = "apex"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "gitlab_token_secret")
}

func TestValidateEggInvalidMemory(t *testing.T) {
	content := []byte(`
egg "my-service" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_token_secret   = "vault://gitlab/token"
  gitlab_webhook_secret = "vault://gitlab/webhook"
  git_repo_url_secret   = "vault://gitlab/repo"
  cloud_provider        = "yandex"
  region                = "ru-central1"
  runner_type           = "nadir"
  memory                = "512"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "memory")
}

func TestValidateEggInvalidName(t *testing.T) {
	content := []byte(`
egg "123-invalid" {
  gitlab_server         = "https://gitlab.com"
  project_id            = 12345
  gitlab_token_secret   = "vault://gitlab/token"
  gitlab_webhook_secret = "vault://gitlab/webhook"
  git_repo_url_secret   = "vault://gitlab/repo"
  cloud_provider        = "yandex"
  region                = "ru-central1"
  runner_type           = "serverless"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "name")
}

// --- eggsbucket ---

func TestValidateEggsBucketValid(t *testing.T) {
	content := []byte(`
eggsbucket "platform-team" {
  gitlab_server         = "https://gitlab.example.com"
  group_id              = 999
  gitlab_token_secret   = "aws-sm://prod/platform-runner-token/value"
  gitlab_webhook_secret = "aws-sm://prod/platform-webhook-secret/value"
  cloud_provider        = "aws"
  region                = "us-east-1"
  runner_type           = "apex"
  tags                  = ["docker", "linux", "aws"]
  cpu                   = 2
  memory                = "1GB"
  project_ids           = [101, 102, 103]
}
`)
	assertValid(t, content)
}

func TestValidateEggsBucketMissingGroupID(t *testing.T) {
	content := []byte(`
eggsbucket "platform-team" {
  gitlab_server         = "https://gitlab.example.com"
  gitlab_token_secret   = "aws-sm://prod/token/value"
  gitlab_webhook_secret = "aws-sm://prod/webhook/value"
  cloud_provider        = "aws"
  region                = "us-east-1"
  runner_type           = "apex"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "group_id")
}

// --- job ---

func TestValidateJobValid(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule       = "0 2 * * 0"
  cloud_provider = "yandex"
  region         = "ru-central1"
  runner_image   = "registry.example.com/tools/secret-rotator:latest"
  cpu            = 0.5
  memory         = "256MB"
  timeout        = "30m"
  script         = "#!/bin/bash\nset -euo pipefail\npython3 /app/rotate.py --all"
  secrets        = ["yc-lockbox://abc123/rotation-key"]
}
`)
	assertValid(t, content)
}

func TestValidateJobMissingSchedule(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  cloud_provider = "yandex"
  region         = "ru-central1"
  script         = "echo hello"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "schedule")
}

func TestValidateJobMissingScript(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule       = "0 2 * * *"
  cloud_provider = "yandex"
  region         = "ru-central1"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "script")
}

func TestValidateJobInvalidCron(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule       = "not a cron"
  cloud_provider = "yandex"
  region         = "ru-central1"
  script         = "echo hello"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "schedule")
}

func TestValidateJobInvalidSecretURI(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule       = "0 2 * * *"
  cloud_provider = "yandex"
  region         = "ru-central1"
  script         = "echo hello"
  secrets        = ["plain-secret"]
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "secrets[0]")
}

func TestValidateJobInvalidTimeout(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule       = "0 2 * * *"
  cloud_provider = "yandex"
  region         = "ru-central1"
  script         = "echo hello"
  timeout        = "30minutes"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "timeout")
}

// --- uglyfox ---

func TestValidateUglyFoxValid(t *testing.T) {
	content := []byte(`
uglyfox {
  pruning {
    max_age_hours          = 72
    max_failures           = 5
    idle_timeout_minutes   = 30
    check_interval_seconds = 60
  }

  apex_pool {
    min_size           = 1
    max_size           = 10
    scale_up_threshold = 5
  }

  nadir_pool {
    min_size            = 0
    max_size            = 5
    warmup_time_seconds = 30
  }

  runners_condition {
    egg_name      = "my-service"
    max_failures  = 3
    max_age_hours = 24
  }
}
`)
	assertValid(t, content)
}

func TestValidateUglyFoxMissingPruning(t *testing.T) {
	content := []byte(`
uglyfox {
  apex_pool {
    min_size           = 1
    max_size           = 10
    scale_up_threshold = 5
  }
  nadir_pool {
    min_size            = 0
    max_size            = 5
    warmup_time_seconds = 30
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "pruning")
}

func TestValidateUglyFoxMissingApexPool(t *testing.T) {
	content := []byte(`
uglyfox {
  pruning {
    max_age_hours          = 72
    max_failures           = 5
    idle_timeout_minutes   = 30
    check_interval_seconds = 60
  }
  nadir_pool {
    min_size            = 0
    max_size            = 5
    warmup_time_seconds = 30
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "apex_pool")
}

func TestValidateUglyFoxPoolMinGreaterThanMax(t *testing.T) {
	content := []byte(`
uglyfox {
  pruning {
    max_age_hours          = 72
    max_failures           = 5
    idle_timeout_minutes   = 30
    check_interval_seconds = 60
  }
  apex_pool {
    min_size           = 10
    max_size           = 5
    scale_up_threshold = 3
  }
  nadir_pool {
    min_size            = 0
    max_size            = 5
    warmup_time_seconds = 30
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "min_size")
}

// --- mothergoose ---

func TestValidateMotherGooseValid(t *testing.T) {
	content := []byte(`
mothergoose {
  api_gateway {
    cloud_provider = "yandex"
    region         = "ru-central1"
    domain         = "mg.example.com"
    tls            = true
  }

  message_queue {
    cloud_provider = "yandex"
    queue_name     = "polar-gosling-tasks"
  }

  cloud_trigger {
    type     = "timer"
    schedule = "*/5 * * * *"
    target   = "/internal/sync-git"
  }

  container {
    image         = "registry.example.com/polar-gosling/mothergoose:latest"
    cpu           = 1
    memory        = "512MB"
    min_instances = 1
    max_instances = 5
  }
}
`)
	assertValid(t, content)
}

func TestValidateMotherGooseMissingAPIGateway(t *testing.T) {
	content := []byte(`
mothergoose {
  message_queue {
    cloud_provider = "yandex"
    queue_name     = "polar-gosling-tasks"
  }
  cloud_trigger {
    type   = "timer"
    target = "/internal/sync-git"
  }
  container {
    image = "registry.example.com/mg:latest"
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "api_gateway")
}

func TestValidateMotherGooseMissingContainer(t *testing.T) {
	content := []byte(`
mothergoose {
  api_gateway {
    cloud_provider = "yandex"
    region         = "ru-central1"
    domain         = "mg.example.com"
  }
  message_queue {
    cloud_provider = "yandex"
    queue_name     = "tasks"
  }
  cloud_trigger {
    type   = "timer"
    target = "/internal/sync-git"
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "container")
}

func TestValidateMotherGooseInvalidCloudTriggerType(t *testing.T) {
	content := []byte(`
mothergoose {
  api_gateway {
    cloud_provider = "yandex"
    region         = "ru-central1"
    domain         = "mg.example.com"
  }
  message_queue {
    cloud_provider = "yandex"
    queue_name     = "tasks"
  }
  cloud_trigger {
    type   = "cron"
    target = "/internal/sync-git"
  }
  container {
    image = "registry.example.com/mg:latest"
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "type")
}

func TestValidateMotherGooseInvalidContainerMemory(t *testing.T) {
	content := []byte(`
mothergoose {
  api_gateway {
    cloud_provider = "yandex"
    region         = "ru-central1"
    domain         = "mg.example.com"
  }
  message_queue {
    cloud_provider = "yandex"
    queue_name     = "tasks"
  }
  cloud_trigger {
    type   = "timer"
    target = "/internal/sync-git"
  }
  container {
    image  = "registry.example.com/mg:latest"
    memory = "512"
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "memory")
}

func TestValidateMotherGooseInvalidAPIGatewayProvider(t *testing.T) {
	content := []byte(`
mothergoose {
  api_gateway {
    cloud_provider = "azure"
    region         = "eastus"
    domain         = "mg.example.com"
  }
  message_queue {
    cloud_provider = "yandex"
    queue_name     = "tasks"
  }
  cloud_trigger {
    type   = "timer"
    target = "/internal/sync-git"
  }
  container {
    image = "registry.example.com/mg:latest"
  }
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "cloud_provider")
}

// --- unknown block ---

func TestValidateUnknownBlock(t *testing.T) {
	content := []byte(`
unknown_block "foo" {
  bar = "baz"
}
`)
	result := validate(t, content)
	assertHasErrorField(t, result, "type")
}

// --- test helpers ---

func parse(t *testing.T, content []byte) *Config {
	t.Helper()
	p := NewParser()
	config, err := p.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	return config
}

func validate(t *testing.T, content []byte) *ValidationResult {
	t.Helper()
	return NewValidator(parse(t, content)).Validate()
}

func assertValid(t *testing.T, content []byte) {
	t.Helper()
	result := validate(t, content)
	if !result.IsValid() {
		t.Errorf("Expected valid config, got errors: %v", result.Error())
	}
}

func assertHasErrorField(t *testing.T, result *ValidationResult, field string) {
	t.Helper()
	if result.IsValid() {
		t.Errorf("Expected validation errors but got none (looking for field %q)", field)
		return
	}
	for _, e := range result.Errors {
		if e.Field == field {
			return
		}
	}
	t.Errorf("Expected validation error for field %q, got: %v", field, result.Error())
}
