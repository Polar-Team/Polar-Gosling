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

	// Check apex block
	apexBlock, ok := uglyFoxBlock.GetBlock("apex")
	if !ok {
		t.Fatal("Missing 'apex' block")
	}

	maxCountVal, ok := apexBlock.GetAttribute("max_count")
	if !ok {
		t.Fatal("Missing 'max_count' attribute in apex block")
	}
	maxCount, _ := maxCountVal.AsInt()
	if maxCount != 10 {
		t.Errorf("Expected max_count 10, got %d", maxCount)
	}

	// Check nadir block
	nadirBlock, ok := uglyFoxBlock.GetBlock("nadir")
	if !ok {
		t.Fatal("Missing 'nadir' block")
	}

	idleTimeoutVal, ok := nadirBlock.GetAttribute("idle_timeout")
	if !ok {
		t.Fatal("Missing 'idle_timeout' attribute in nadir block")
	}
	idleTimeout, _ := idleTimeoutVal.AsString()
	if idleTimeout != "30m" {
		t.Errorf("Expected idle_timeout '30m', got %q", idleTimeout)
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
