package deployer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to write a .fly file into a temp directory
func writeFlyFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

const mgYandexFly = `
mothergoose "yandex_prod" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g1234567890"
    yc_cloud_id  = "b1gabcdefghij"
  }

  api_gateway {
    name            = "mg-gateway"
    description     = "MotherGoose API Gateway"
    spec_path       = "openapi.yaml"
    service_account = "mg-gateway-sa"
  }

  fastapi_app {
    name              = "mg-fastapi"
    image             = "cr.yandex/mg/fastapi:latest"
    memory            = 512
    cores             = 1
    core_fraction     = 100
    execution_timeout = "300s"
    concurrency       = 4
    service_account   = "mg-fastapi-sa"
  }

  celery_workers {
    name              = "mg-celery"
    image             = "cr.yandex/mg/celery:latest"
    memory            = 1024
    cores             = 2
    core_fraction     = 100
    execution_timeout = "600s"
    concurrency       = 1
    service_account   = "mg-celery-sa"
  }

  git_sync_trigger {
    schedule        = "*/5 * * * *"
    service_account = "mg-trigger-sa"
  }

  mothergoose_queues {
    task_queue {
      name               = "mg-tasks"
      visibility_timeout = 30
      message_retention  = 86400
      max_message_size   = 262144
      receive_wait_time  = 20
    }
    dlq {
      name              = "mg-tasks-dlq"
      message_retention = 86400
    }
  }

  database {
    name            = "mg-db"
    type            = "ydb"
    serverless_mode = true
  }

  storage {
    bucket_name = "mg-storage"
    region      = "ru-central1"
  }

  service_account {
    name        = "mg-main-sa"
    description = "MotherGoose main service account"
    roles       = ["lockbox.payloadViewer", "ymq.writer"]
  }
}
`

const mgAWSFly = `
mothergoose "aws_staging" {
  cloud {
    provider       = "aws"
    aws_region     = "us-east-1"
    aws_account_id = "123456789012"
  }

  api_gateway {
    name = "mg-gateway-aws"
  }

  fastapi_app {
    name  = "mg-fastapi-aws"
    image = "123456789012.dkr.ecr.us-east-1.amazonaws.com/mg:latest"
  }

  celery_workers {
    name  = "mg-celery-aws"
    image = "123456789012.dkr.ecr.us-east-1.amazonaws.com/celery:latest"
  }

  database {
    name = "mg-dynamodb"
    type = "dynamodb"
  }

  storage {
    bucket_name = "mg-storage-aws"
    region      = "us-east-1"
  }
}
`

const ufYandexFly = `
uglyfox "yandex_prod" {
  mothergoose = "yandex_prod"

  workers {
    name              = "uf-workers"
    image             = "cr.yandex/uf/worker:latest"
    memory            = 512
    cores             = 1
    core_fraction     = 100
    execution_timeout = "300s"
    concurrency       = 1
    service_account   = "uf-worker-sa"
  }

  service_account {
    name        = "uf-main-sa"
    description = "UglyFox service account"
    roles       = ["lockbox.payloadViewer"]
  }
}
`

func TestParseMGDirectory_SingleFile(t *testing.T) {
	mgDir := t.TempDir()
	writeFlyFile(t, mgDir, "config.fly", mgYandexFly)

	configs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 MGConfig, got %d", len(configs))
	}

	mg := configs[0]
	if mg.Name != "yandex_prod" {
		t.Errorf("expected name 'yandex_prod', got %q", mg.Name)
	}
	if mg.Cloud.Provider != CloudProviderYandex {
		t.Errorf("expected provider yandex, got %q", mg.Cloud.Provider)
	}
	if mg.Cloud.YCFolderID != "b1g1234567890" {
		t.Errorf("expected yc_folder_id 'b1g1234567890', got %q", mg.Cloud.YCFolderID)
	}
	if mg.Cloud.YCCloudID != "b1gabcdefghij" {
		t.Errorf("expected yc_cloud_id 'b1gabcdefghij', got %q", mg.Cloud.YCCloudID)
	}
	if mg.APIGateway.Name != "mg-gateway" {
		t.Errorf("expected api_gateway name 'mg-gateway', got %q", mg.APIGateway.Name)
	}
	if mg.FastAPIApp.Memory != 512 {
		t.Errorf("expected fastapi_app memory 512, got %d", mg.FastAPIApp.Memory)
	}
	if mg.CeleryWorkers.Cores != 2 {
		t.Errorf("expected celery_workers cores 2, got %d", mg.CeleryWorkers.Cores)
	}
	if mg.Queues.TaskQueue.Name != "mg-tasks" {
		t.Errorf("expected task queue name 'mg-tasks', got %q", mg.Queues.TaskQueue.Name)
	}
	if mg.Queues.DLQ.Name != "mg-tasks-dlq" {
		t.Errorf("expected dlq name 'mg-tasks-dlq', got %q", mg.Queues.DLQ.Name)
	}
	if mg.GitSyncTrigger.Schedule != "*/5 * * * *" {
		t.Errorf("expected git_sync_trigger schedule '*/5 * * * *', got %q", mg.GitSyncTrigger.Schedule)
	}
	if mg.GitSyncTrigger.ServiceAccount != "mg-trigger-sa" {
		t.Errorf("expected git_sync_trigger service_account 'mg-trigger-sa', got %q", mg.GitSyncTrigger.ServiceAccount)
	}
	if mg.Database.ServerlessMode != true {
		t.Error("expected database serverless_mode true")
	}
	if mg.Storage.BucketName != "mg-storage" {
		t.Errorf("expected bucket name 'mg-storage', got %q", mg.Storage.BucketName)
	}
	if mg.Storage.Region != "ru-central1" {
		t.Errorf("expected region 'ru-central1', got %q", mg.Storage.Region)
	}
	if len(mg.ServiceAccounts) != 1 {
		t.Fatalf("expected 1 service account, got %d", len(mg.ServiceAccounts))
	}
	if len(mg.ServiceAccounts[0].Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(mg.ServiceAccounts[0].Roles))
	}
}

func TestParseMGDirectory_MultipleInstances(t *testing.T) {
	mgDir := t.TempDir()
	writeFlyFile(t, mgDir, "yandex.fly", mgYandexFly)
	writeFlyFile(t, mgDir, "aws.fly", mgAWSFly)

	configs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 MGConfigs, got %d", len(configs))
	}

	names := map[string]bool{}
	for _, c := range configs {
		names[c.Name] = true
	}
	if !names["yandex_prod"] {
		t.Error("missing yandex_prod config")
	}
	if !names["aws_staging"] {
		t.Error("missing aws_staging config")
	}
}

func TestParseMGDirectory_MultipleBlocksInOneFile(t *testing.T) {
	mgDir := t.TempDir()
	content := mgYandexFly + "\n" + mgAWSFly
	writeFlyFile(t, mgDir, "all.fly", content)

	configs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 MGConfigs from single file, got %d", len(configs))
	}
}

func TestParseUFDirectory_SingleFile(t *testing.T) {
	mgDir := t.TempDir()
	ufDir := t.TempDir()
	writeFlyFile(t, mgDir, "config.fly", mgYandexFly)
	writeFlyFile(t, ufDir, "config.fly", ufYandexFly)

	mgConfigs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed: %v", err)
	}

	ufConfigs, err := ParseUFDirectory(ufDir, mgConfigs)
	if err != nil {
		t.Fatalf("ParseUFDirectory failed: %v", err)
	}
	if len(ufConfigs) != 1 {
		t.Fatalf("expected 1 UFConfig, got %d", len(ufConfigs))
	}

	uf := ufConfigs[0]
	if uf.Name != "yandex_prod" {
		t.Errorf("expected name 'yandex_prod', got %q", uf.Name)
	}
	if uf.MotherGooseRef != "yandex_prod" {
		t.Errorf("expected mothergoose ref 'yandex_prod', got %q", uf.MotherGooseRef)
	}
	// Cloud should be inherited from MG
	if uf.Cloud.Provider != CloudProviderYandex {
		t.Errorf("expected inherited provider yandex, got %q", uf.Cloud.Provider)
	}
	if uf.Cloud.YCFolderID != "b1g1234567890" {
		t.Errorf("expected inherited yc_folder_id, got %q", uf.Cloud.YCFolderID)
	}
	if uf.Workers.Name != "uf-workers" {
		t.Errorf("expected workers name 'uf-workers', got %q", uf.Workers.Name)
	}
	if uf.ServiceAccount.Name != "uf-main-sa" {
		t.Errorf("expected sa name 'uf-main-sa', got %q", uf.ServiceAccount.Name)
	}
}

func TestParseUFDirectory_ReferencesSpecificMG(t *testing.T) {
	mgDir := t.TempDir()
	ufDir := t.TempDir()

	// Two MG instances
	writeFlyFile(t, mgDir, "yandex.fly", mgYandexFly)
	writeFlyFile(t, mgDir, "aws.fly", mgAWSFly)

	// UF references the AWS one
	ufContent := `
uglyfox "aws_staging" {
  mothergoose = "aws_staging"

  workers {
    name  = "uf-aws-workers"
    image = "123456789012.dkr.ecr.us-east-1.amazonaws.com/uf:latest"
  }
}
`
	writeFlyFile(t, ufDir, "config.fly", ufContent)

	mgConfigs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed: %v", err)
	}

	ufConfigs, err := ParseUFDirectory(ufDir, mgConfigs)
	if err != nil {
		t.Fatalf("ParseUFDirectory failed: %v", err)
	}
	if len(ufConfigs) != 1 {
		t.Fatalf("expected 1 UFConfig, got %d", len(ufConfigs))
	}

	uf := ufConfigs[0]
	if uf.Cloud.Provider != CloudProviderAWS {
		t.Errorf("expected inherited provider aws, got %q", uf.Cloud.Provider)
	}
	if uf.Cloud.AWSRegion != "us-east-1" {
		t.Errorf("expected inherited aws_region 'us-east-1', got %q", uf.Cloud.AWSRegion)
	}
}

func TestParseUFDirectory_MissingMGReference(t *testing.T) {
	mgDir := t.TempDir()
	ufDir := t.TempDir()

	writeFlyFile(t, mgDir, "config.fly", mgYandexFly)

	ufContent := `
uglyfox "broken" {
  mothergoose = "nonexistent_mg"

  workers {
    name  = "uf-workers"
    image = "some-image:latest"
  }
}
`
	writeFlyFile(t, ufDir, "config.fly", ufContent)

	mgConfigs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed: %v", err)
	}

	_, err = ParseUFDirectory(ufDir, mgConfigs)
	if err == nil {
		t.Fatal("expected error for missing MG reference, got nil")
	}
}

func TestParseUFDirectory_MultipleFiles(t *testing.T) {
	mgDir := t.TempDir()
	ufDir := t.TempDir()

	writeFlyFile(t, mgDir, "yandex.fly", mgYandexFly)
	writeFlyFile(t, mgDir, "aws.fly", mgAWSFly)

	writeFlyFile(t, ufDir, "yandex.fly", ufYandexFly)
	writeFlyFile(t, ufDir, "aws.fly", `
uglyfox "aws_staging" {
  mothergoose = "aws_staging"

  workers {
    name  = "uf-aws"
    image = "uf-aws:latest"
  }
}
`)

	mgConfigs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed: %v", err)
	}

	ufConfigs, err := ParseUFDirectory(ufDir, mgConfigs)
	if err != nil {
		t.Fatalf("ParseUFDirectory failed: %v", err)
	}
	if len(ufConfigs) != 2 {
		t.Fatalf("expected 2 UFConfigs, got %d", len(ufConfigs))
	}
}

func TestParseMGDirectory_YandexMissingFolderID(t *testing.T) {
	mgDir := t.TempDir()
	content := `
mothergoose "bad_yandex" {
  cloud {
    provider    = "yandex"
    yc_cloud_id = "b1gabcdefghij"
  }
}
`
	writeFlyFile(t, mgDir, "config.fly", content)

	_, err := ParseMGDirectory(mgDir)
	if err == nil {
		t.Fatal("expected error for missing yc_folder_id, got nil")
	}
}

func TestParseMGDirectory_AWSMissingRegion(t *testing.T) {
	mgDir := t.TempDir()
	content := `
mothergoose "bad_aws" {
  cloud {
    provider = "aws"
  }
}
`
	writeFlyFile(t, mgDir, "config.fly", content)

	_, err := ParseMGDirectory(mgDir)
	if err == nil {
		t.Fatal("expected error for missing aws_region, got nil")
	}
}

func TestParseMGDirectory_EmptyDir(t *testing.T) {
	mgDir := t.TempDir()

	configs, err := ParseMGDirectory(mgDir)
	if err != nil {
		t.Fatalf("ParseMGDirectory failed on empty dir: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs from empty dir, got %d", len(configs))
	}
}

func TestParseMGDirectory_LegacyStateBucketRejected(t *testing.T) {
	mgDir := t.TempDir()
	writeFlyFile(t, mgDir, "config.fly", `
mothergoose "legacy_test" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g1234567890"
    yc_cloud_id  = "b1gabcdefghij"
  }

  storage {
    state_bucket {
      name       = "old-state"
      versioning = true
    }
  }
}
`)

	_, err := ParseMGDirectory(mgDir)
	if err == nil {
		t.Fatal("expected error for legacy state_bucket sub-block, got nil")
	}
	if !strings.Contains(err.Error(), "state_bucket") || !strings.Contains(err.Error(), "no longer supported") {
		t.Errorf("expected legacy migration error mentioning state_bucket, got: %v", err)
	}
}

func TestParseMGDirectory_LegacyBinaryBucketOnlyRejected(t *testing.T) {
	mgDir := t.TempDir()
	writeFlyFile(t, mgDir, "config.fly", `
mothergoose "legacy_bin" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g1234567890"
    yc_cloud_id  = "b1gabcdefghij"
  }

  storage {
    binary_bucket {
      name = "old-bin"
    }
  }
}
`)

	_, err := ParseMGDirectory(mgDir)
	if err == nil {
		t.Fatal("expected error for legacy binary_bucket sub-block, got nil")
	}
	if !strings.Contains(err.Error(), "binary_bucket") || !strings.Contains(err.Error(), "no longer supported") {
		t.Errorf("expected legacy migration error mentioning binary_bucket, got: %v", err)
	}
}

func TestParseMGDirectory_LegacyBothBucketsRejected(t *testing.T) {
	mgDir := t.TempDir()
	writeFlyFile(t, mgDir, "config.fly", `
mothergoose "legacy_both" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g1234567890"
    yc_cloud_id  = "b1gabcdefghij"
  }

  storage {
    state_bucket {
      name = "old-state"
    }
    binary_bucket {
      name = "old-bin"
    }
  }
}
`)

	_, err := ParseMGDirectory(mgDir)
	if err == nil {
		t.Fatal("expected error for legacy sub-blocks, got nil")
	}
	if !strings.Contains(err.Error(), "state_bucket") {
		t.Errorf("expected error to mention state_bucket, got: %v", err)
	}
	if !strings.Contains(err.Error(), "binary_bucket") {
		t.Errorf("expected error to mention binary_bucket, got: %v", err)
	}
}

func TestParseMGDirectory_StorageBucketNameWithoutRegion(t *testing.T) {
	mgDir := t.TempDir()
	writeFlyFile(t, mgDir, "config.fly", `
mothergoose "missing_region" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g1234567890"
    yc_cloud_id  = "b1gabcdefghij"
  }

  storage {
    bucket_name = "my-bucket"
  }
}
`)

	_, err := ParseMGDirectory(mgDir)
	if err == nil {
		t.Fatal("expected error for bucket_name without region, got nil")
	}
	if !strings.Contains(err.Error(), "both") {
		t.Errorf("expected cross-field validation error, got: %v", err)
	}
}
