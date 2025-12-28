package parser

import (
	"testing"
)

func TestParseEggConfig(t *testing.T) {
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
	type = vm
  }

  runner {
    tags = ["docker", "linux"]
    concurrent = 3
    idle_timeout = "10m"
  }

  gitlab {
    project_id = 12345
  	server_name = "https://exmple.com"
    token_secret = "vault://gitlab/runner-token"
  }

  environment {
    DOCKER_DRIVER = "overlay2"
    CUSTOM_VAR    = "value"
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(config.Blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(config.Blocks))
	}

	eggBlock := config.Blocks[0]
	if eggBlock.Type != "egg" {
		t.Errorf("Expected block type 'egg', got %q", eggBlock.Type)
	}

	if len(eggBlock.Labels) != 1 || eggBlock.Labels[0] != "my-app" {
		t.Errorf("Expected label 'my-app', got %v", eggBlock.Labels)
	}

	// Check type attribute
	typeVal, ok := eggBlock.GetAttribute("type")
	if !ok {
		t.Fatal("Missing 'type' attribute")
	}
	typeStr, err := typeVal.AsString()
	if err != nil {
		t.Fatalf("Type attribute error: %v", err)
	}
	if typeStr != "vm" {
		t.Errorf("Expected type 'vm', got %q", typeStr)
	}

	// Check cloud block
	cloudBlock, ok := eggBlock.GetBlock("cloud")
	if !ok {
		t.Fatal("Missing 'cloud' block")
	}

	providerVal, ok := cloudBlock.GetAttribute("provider")
	if !ok {
		t.Fatal("Missing 'provider' attribute in cloud block")
	}
	providerStr, _ := providerVal.AsString()
	if providerStr != "yandex" {
		t.Errorf("Expected provider 'yandex', got %q", providerStr)
	}

	// Check resources block
	resourcesBlock, ok := eggBlock.GetBlock("resources")
	if !ok {
		t.Fatal("Missing 'resources' block")
	}

	cpuVal, ok := resourcesBlock.GetAttribute("cpu")
	if !ok {
		t.Fatal("Missing 'cpu' attribute in resources block")
	}
	cpuNum, _ := cpuVal.AsInt()
	if cpuNum != 2 {
		t.Errorf("Expected cpu 2, got %d", cpuNum)
	}

	// Check runner block
	runnerBlock, ok := eggBlock.GetBlock("runner")
	if !ok {
		t.Fatal("Missing 'runner' block")
	}

	tagsVal, ok := runnerBlock.GetAttribute("tags")
	if !ok {
		t.Fatal("Missing 'tags' attribute in runner block")
	}
	tagsList, _ := tagsVal.AsList()
	if len(tagsList) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tagsList))
	}

	// Check gitlab block
	gitlabBlock, ok := eggBlock.GetBlock("gitlab")
	if !ok {
		t.Fatal("Missing 'gitlab' block")
	}

	projectIDVal, ok := gitlabBlock.GetAttribute("project_id")
	if !ok {
		t.Fatal("Missing 'project_id' attribute in gitlab block")
	}
	projectID, _ := projectIDVal.AsInt()
	if projectID != 12345 {
		t.Errorf("Expected project_id 12345, got %d", projectID)
	}

	// Check environment block
	envBlock, ok := eggBlock.GetBlock("environment")
	if !ok {
		t.Fatal("Missing 'environment' block")
	}

	dockerDriverVal, ok := envBlock.GetAttribute("DOCKER_DRIVER")
	if !ok {
		t.Fatal("Missing 'DOCKER_DRIVER' attribute in environment block")
	}
	dockerDriver, _ := dockerDriverVal.AsString()
	if dockerDriver != "overlay2" {
		t.Errorf("Expected DOCKER_DRIVER 'overlay2', got %q", dockerDriver)
	}
}

func TestParseJobConfig(t *testing.T) {
	content := []byte(`
job "rotate-secrets" {
  schedule = "0 2 * * *"

  runner {
    type = "vm"
    tags = ["privileged"]
  }

  script = <<-EOT
    #!/bin/bash
    # Rotate GitLab runner tokens
    gosling rotate-tokens --all
  EOT
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(config.Blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(config.Blocks))
	}

	jobBlock := config.Blocks[0]
	if jobBlock.Type != "job" {
		t.Errorf("Expected block type 'job', got %q", jobBlock.Type)
	}

	if len(jobBlock.Labels) != 1 || jobBlock.Labels[0] != "rotate-secrets" {
		t.Errorf("Expected label 'rotate-secrets', got %v", jobBlock.Labels)
	}

	// Check schedule attribute
	scheduleVal, ok := jobBlock.GetAttribute("schedule")
	if !ok {
		t.Fatal("Missing 'schedule' attribute")
	}
	scheduleStr, _ := scheduleVal.AsString()
	if scheduleStr != "0 2 * * *" {
		t.Errorf("Expected schedule '0 2 * * *', got %q", scheduleStr)
	}

	// Check script attribute
	scriptVal, ok := jobBlock.GetAttribute("script")
	if !ok {
		t.Fatal("Missing 'script' attribute")
	}
	scriptStr, _ := scriptVal.AsString()
	if scriptStr == "" {
		t.Error("Expected non-empty script")
	}

	// Check runner block
	runnerBlock, ok := jobBlock.GetBlock("runner")
	if !ok {
		t.Fatal("Missing 'runner' block")
	}

	typeVal, ok := runnerBlock.GetAttribute("type")
	if !ok {
		t.Fatal("Missing 'type' attribute in runner block")
	}
	typeStr, _ := typeVal.AsString()
	if typeStr != "vm" {
		t.Errorf("Expected type 'vm', got %q", typeStr)
	}
}

func TestParseUglyFoxConfig(t *testing.T) {
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

  runners_condition "high-performance" {
    eggs_entities = ["Egg3", "EggsBucket4"]

    apex {
      max_count = 20
      min_count = 5
      cpu_threshold = 90
      memory_threshold = 80
    }

    nadir {
      max_count = 10
      min_count = 2
      idle_timeout = "15m"
    }
  }

  policies {
    rule "terminate_old_failed" {
      condition = "failed_count >= 3 AND age > 1h"
      action    = "terminate"
    }

    rule "demote_idle" {
      condition = "state == 'apex' AND idle_time > 30m"
      action    = "demote_to_nadir"
    }
  }
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(config.Blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(config.Blocks))
	}

	uglyFoxBlock := config.Blocks[0]
	if uglyFoxBlock.Type != "uglyfox" {
		t.Errorf("Expected block type 'uglyfox', got %q", uglyFoxBlock.Type)
	}

	// Check pruning block
	pruningBlock, ok := uglyFoxBlock.GetBlock("pruning")
	if !ok {
		t.Fatal("Missing 'pruning' block")
	}

	failedThresholdVal, ok := pruningBlock.GetAttribute("failed_threshold")
	if !ok {
		t.Fatal("Missing 'failed_threshold' attribute")
	}
	failedThreshold, _ := failedThresholdVal.AsInt()
	if failedThreshold != 3 {
		t.Errorf("Expected failed_threshold 3, got %d", failedThreshold)
	}

	// Check runners_condition blocks
	runnersConditions := uglyFoxBlock.GetBlocks("runners_condition")
	if len(runnersConditions) != 2 {
		t.Fatalf("Expected 2 runners_condition blocks, got %d", len(runnersConditions))
	}

	// Check first runners_condition block (default)
	defaultCondition := runnersConditions[0]
	if len(defaultCondition.Labels) != 1 || defaultCondition.Labels[0] != "default" {
		t.Errorf("Expected label 'default', got %v", defaultCondition.Labels)
	}

	eggsEntitiesVal, ok := defaultCondition.GetAttribute("eggs_entities")
	if !ok {
		t.Fatal("Missing 'eggs_entities' attribute in runners_condition")
	}
	eggsEntitiesList, _ := eggsEntitiesVal.AsList()
	if len(eggsEntitiesList) != 2 {
		t.Errorf("Expected 2 eggs_entities, got %d", len(eggsEntitiesList))
	}

	// Check apex block within runners_condition
	apexBlock, ok := defaultCondition.GetBlock("apex")
	if !ok {
		t.Fatal("Missing 'apex' block in runners_condition")
	}

	maxCountVal, ok := apexBlock.GetAttribute("max_count")
	if !ok {
		t.Fatal("Missing 'max_count' attribute in apex block")
	}
	maxCount, _ := maxCountVal.AsInt()
	if maxCount != 10 {
		t.Errorf("Expected max_count 10, got %d", maxCount)
	}

	cpuThresholdVal, ok := apexBlock.GetAttribute("cpu_threshold")
	if !ok {
		t.Fatal("Missing 'cpu_threshold' attribute in apex block")
	}
	cpuThreshold, _ := cpuThresholdVal.AsInt()
	if cpuThreshold != 80 {
		t.Errorf("Expected cpu_threshold 80, got %d", cpuThreshold)
	}

	// Check nadir block within runners_condition
	nadirBlock, ok := defaultCondition.GetBlock("nadir")
	if !ok {
		t.Fatal("Missing 'nadir' block in runners_condition")
	}

	idleTimeoutVal, ok := nadirBlock.GetAttribute("idle_timeout")
	if !ok {
		t.Fatal("Missing 'idle_timeout' attribute in nadir block")
	}
	idleTimeout, _ := idleTimeoutVal.AsString()
	if idleTimeout != "30m" {
		t.Errorf("Expected idle_timeout '30m', got %q", idleTimeout)
	}

	// Check second runners_condition block (high-performance)
	hpCondition := runnersConditions[1]
	if len(hpCondition.Labels) != 1 || hpCondition.Labels[0] != "high-performance" {
		t.Errorf("Expected label 'high-performance', got %v", hpCondition.Labels)
	}

	hpApexBlock, ok := hpCondition.GetBlock("apex")
	if !ok {
		t.Fatal("Missing 'apex' block in high-performance runners_condition")
	}

	hpMaxCountVal, ok := hpApexBlock.GetAttribute("max_count")
	if !ok {
		t.Fatal("Missing 'max_count' attribute in high-performance apex block")
	}
	hpMaxCount, _ := hpMaxCountVal.AsInt()
	if hpMaxCount != 20 {
		t.Errorf("Expected max_count 20, got %d", hpMaxCount)
	}

	// Check policies block
	policiesBlock, ok := uglyFoxBlock.GetBlock("policies")
	if !ok {
		t.Fatal("Missing 'policies' block")
	}

	rules := policiesBlock.GetBlocks("rule")
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	// Check first rule
	if len(rules) > 0 {
		rule := rules[0]
		if len(rule.Labels) != 1 || rule.Labels[0] != "terminate_old_failed" {
			t.Errorf("Expected rule label 'terminate_old_failed', got %v", rule.Labels)
		}

		actionVal, ok := rule.GetAttribute("action")
		if !ok {
			t.Fatal("Missing 'action' attribute in rule")
		}
		action, _ := actionVal.AsString()
		if action != "terminate" {
			t.Errorf("Expected action 'terminate', got %q", action)
		}
	}
}

func TestParseSyntaxError(t *testing.T) {
	content := []byte(`
egg "my-app" {
  type = "vm"
  invalid syntax here
}
`)

	parser := NewParser()
	_, err := parser.Parse(content, "test.fly")
	if err == nil {
		t.Fatal("Expected parse error for invalid syntax")
	}
}

func TestParseTypeError(t *testing.T) {
	content := []byte(`
egg "my-app" {
  type = 123
}
`)

	parser := NewParser()
	config, err := parser.Parse(content, "test.fly")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Parsing should succeed, but validation should fail
	validator := NewValidator(config)
	result := validator.Validate()
	if result.IsValid() {
		t.Error("Expected validation to fail for type mismatch")
	}
}
