package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Position Position
	Message  string
	Field    string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (field: %s)", e.Position, e.Message, e.Field)
}

// ValidationResult contains all validation errors
type ValidationResult struct {
	Errors []*ValidationError
}

// IsValid returns true if there are no validation errors
func (vr *ValidationResult) IsValid() bool {
	return len(vr.Errors) == 0
}

// Error returns a formatted error message with all validation errors
func (vr *ValidationResult) Error() string {
	if vr.IsValid() {
		return ""
	}

	var messages []string
	for _, err := range vr.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("validation failed with %d error(s):\n%s",
		len(vr.Errors), strings.Join(messages, "\n"))
}

// AddError adds a validation error
func (vr *ValidationResult) AddError(pos Position, field, message string) {
	vr.Errors = append(vr.Errors, &ValidationError{
		Position: pos,
		Field:    field,
		Message:  message,
	})
}

// Validator validates .fly configuration files
type Validator struct {
	config *Config
	result *ValidationResult
}

// NewValidator creates a new validator for a config
func NewValidator(config *Config) *Validator {
	return &Validator{
		config: config,
		result: &ValidationResult{
			Errors: make([]*ValidationError, 0),
		},
	}
}

// Validate performs validation on the configuration
func (v *Validator) Validate() *ValidationResult {
	// Validate each top-level block
	for _, block := range v.config.Blocks {
		v.validateBlock(&block)
	}

	return v.result
}

// validateBlock validates a block based on its type
func (v *Validator) validateBlock(block *Block) {
	switch block.Type {
	case "egg":
		v.validateEggBlock(block)
	case "eggsbucket":
		v.validateEggsBucketBlock(block)
	case "job":
		v.validateJobBlock(block)
	case "uglyfox":
		v.validateUglyFoxBlock(block)
	case "mothergoose":
		v.validateMotherGooseBlock(block)
	default:
		v.result.AddError(block.Position, "type",
			fmt.Sprintf("unknown block type: %s", block.Type))
	}
}

// validateEggBlock validates an egg configuration block
func (v *Validator) validateEggBlock(block *Block) {
	// Egg must have exactly one label (the name)
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"egg block must have exactly one label (the egg name)")
		return
	}

	// Validate egg name format (alphanumeric, hyphens, underscores)
	eggName := block.Labels[0]
	if !isValidIdentifier(eggName) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid egg name %q: must contain only alphanumeric characters, hyphens, and underscores", eggName))
	}

	// Validate required attribute: type
	typeVal, ok := block.GetAttribute("type")
	if !ok {
		v.result.AddError(block.Position, "type", "egg block must have a 'type' attribute")
	} else {
		typeStr, err := typeVal.AsString()
		if err != nil {
			v.result.AddError(typeVal.Position, "type", "type must be a string")
		} else if typeStr != "vm" && typeStr != "serverless" {
			v.result.AddError(typeVal.Position, "type",
				fmt.Sprintf("type must be 'vm' or 'serverless', got %q", typeStr))
		}
	}

	// Validate required nested blocks
	v.validateRequiredBlock(block, "cloud")
	v.validateRequiredBlock(block, "resources")
	v.validateRequiredBlock(block, "runner")
	v.validateRequiredBlock(block, "gitlab")

	// Validate cloud block
	if cloudBlock, ok := block.GetBlock("cloud"); ok {
		v.validateCloudBlock(cloudBlock)
	}

	// Validate resources block
	if resourcesBlock, ok := block.GetBlock("resources"); ok {
		v.validateResourcesBlock(resourcesBlock)
	}

	// Validate runner block
	if runnerBlock, ok := block.GetBlock("runner"); ok {
		v.validateRunnerBlock(runnerBlock)
	}

	// Validate gitlab block
	if gitlabBlock, ok := block.GetBlock("gitlab"); ok {
		v.validateGitLabBlock(gitlabBlock)
	}

	// Validate optional environment block
	if envBlock, ok := block.GetBlock("environment"); ok {
		v.validateEnvironmentBlock(envBlock)
	}
}

// validateEggsBucketBlock validates an eggsbucket configuration block
func (v *Validator) validateEggsBucketBlock(block *Block) {
	// EggsBucket must have exactly one label (the name)
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"eggsbucket block must have exactly one label (the bucket name)")
		return
	}

	// Validate bucket name format (alphanumeric, hyphens, underscores)
	bucketName := block.Labels[0]
	if !isValidIdentifier(bucketName) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid eggsbucket name %q: must contain only alphanumeric characters, hyphens, and underscores", bucketName))
	}

	// Validate required attribute: type
	typeVal, ok := block.GetAttribute("type")
	if !ok {
		v.result.AddError(block.Position, "type", "eggsbucket block must have a 'type' attribute")
	} else {
		typeStr, err := typeVal.AsString()
		if err != nil {
			v.result.AddError(typeVal.Position, "type", "type must be a string")
		} else if typeStr != "vm" && typeStr != "serverless" {
			v.result.AddError(typeVal.Position, "type",
				fmt.Sprintf("type must be 'vm' or 'serverless', got %q", typeStr))
		}
	}

	// Validate required nested blocks
	v.validateRequiredBlock(block, "cloud")
	v.validateRequiredBlock(block, "resources")
	v.validateRequiredBlock(block, "runner")
	v.validateRequiredBlock(block, "repositories")

	// Validate cloud block
	if cloudBlock, ok := block.GetBlock("cloud"); ok {
		v.validateCloudBlock(cloudBlock)
	}

	// Validate resources block
	if resourcesBlock, ok := block.GetBlock("resources"); ok {
		v.validateResourcesBlock(resourcesBlock)
	}

	// Validate runner block
	if runnerBlock, ok := block.GetBlock("runner"); ok {
		v.validateRunnerBlock(runnerBlock)
	}

	// Validate repositories block
	if repositoriesBlock, ok := block.GetBlock("repositories"); ok {
		v.validateRepositoriesBlock(repositoriesBlock)
	}

	// Validate optional environment block
	if envBlock, ok := block.GetBlock("environment"); ok {
		v.validateEnvironmentBlock(envBlock)
	}
}

// validateRepositoriesBlock validates a repositories block within an eggsbucket
func (v *Validator) validateRepositoriesBlock(block *Block) {
	// Repositories block must contain at least one repo block
	repoBlocks := block.GetBlocks("repo")
	if len(repoBlocks) == 0 {
		v.result.AddError(block.Position, "repo",
			"repositories block must contain at least one 'repo' block")
		return
	}

	// Validate each repo block
	for _, repoBlock := range repoBlocks {
		v.validateRepoBlock(&repoBlock)
	}
}

// validateRepoBlock validates a single repo block within repositories
func (v *Validator) validateRepoBlock(block *Block) {
	// Repo must have exactly one label (the repo name)
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"repo block must have exactly one label (the repo name)")
		return
	}

	// Validate repo name
	repoName := block.Labels[0]
	if !isValidIdentifier(repoName) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid repo name %q: must contain only alphanumeric characters, hyphens, and underscores", repoName))
	}

	// Validate required nested block: gitlab
	v.validateRequiredBlock(block, "gitlab")

	// Validate gitlab block
	if gitlabBlock, ok := block.GetBlock("gitlab"); ok {
		v.validateGitLabBlock(gitlabBlock)
	}
}

// validateJobBlock validates a job configuration block
func (v *Validator) validateJobBlock(block *Block) {
	// Job must have exactly one label (the name)
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"job block must have exactly one label (the job name)")
		return
	}

	// Validate job name
	jobName := block.Labels[0]
	if !isValidIdentifier(jobName) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid job name %q: must contain only alphanumeric characters, hyphens, and underscores", jobName))
	}

	// Validate required attribute: schedule (cron expression)
	scheduleVal, ok := block.GetAttribute("schedule")
	if !ok {
		v.result.AddError(block.Position, "schedule", "job block must have a 'schedule' attribute")
	} else {
		scheduleStr, err := scheduleVal.AsString()
		if err != nil {
			v.result.AddError(scheduleVal.Position, "schedule", "schedule must be a string")
		} else if !isValidCronExpression(scheduleStr) {
			v.result.AddError(scheduleVal.Position, "schedule",
				fmt.Sprintf("invalid cron expression: %q", scheduleStr))
		}
	}

	// Validate required attribute: script
	scriptVal, ok := block.GetAttribute("script")
	if !ok {
		v.result.AddError(block.Position, "script", "job block must have a 'script' attribute")
	} else {
		_, err := scriptVal.AsString()
		if err != nil {
			v.result.AddError(scriptVal.Position, "script", "script must be a string")
		}
	}

	// Validate required nested block: runner
	v.validateRequiredBlock(block, "runner")
	if runnerBlock, ok := block.GetBlock("runner"); ok {
		v.validateJobRunnerBlock(runnerBlock)
	}
}

// validateUglyFoxBlock validates an uglyfox configuration block
func (v *Validator) validateUglyFoxBlock(block *Block) {
	// UglyFox should have no labels
	if len(block.Labels) > 0 {
		v.result.AddError(block.Position, "labels",
			"uglyfox block should not have labels")
	}

	// Validate required nested blocks
	v.validateRequiredBlock(block, "pruning")

	// Validate pruning block
	if pruningBlock, ok := block.GetBlock("pruning"); ok {
		v.validatePruningBlock(pruningBlock)
	}

	// Validate runners_condition blocks (at least one required)
	runnersConditions := block.GetBlocks("runners_condition")
	if len(runnersConditions) == 0 {
		v.result.AddError(block.Position, "runners_condition",
			"uglyfox block must have at least one 'runners_condition' block")
	}

	for _, rcBlock := range runnersConditions {
		v.validateRunnersConditionBlock(&rcBlock)
	}

	// Validate optional policies block
	if policiesBlock, ok := block.GetBlock("policies"); ok {
		v.validatePoliciesBlock(policiesBlock)
	}
}

// validateMotherGooseBlock validates a mothergoose configuration block
func (v *Validator) validateMotherGooseBlock(block *Block) {
	// MotherGoose should have no labels
	if len(block.Labels) > 0 {
		v.result.AddError(block.Position, "labels",
			"mothergoose block should not have labels")
	}

	// Validate required nested blocks
	v.validateRequiredBlock(block, "api_gateway")
	v.validateRequiredBlock(block, "fastapi_app")
	v.validateRequiredBlock(block, "celery_workers")
	v.validateRequiredBlock(block, "uglyfox_workers")
	v.validateRequiredBlock(block, "message_queues")
	v.validateRequiredBlock(block, "triggers")
	v.validateRequiredBlock(block, "database")
	v.validateRequiredBlock(block, "storage")
	v.validateRequiredBlock(block, "service_accounts")

	// Note: Detailed validation of nested blocks would be added here
	// For now, we just validate that the required blocks exist
	// Full validation would check specific attributes within each block
}

// validateRunnersConditionBlock validates a runners_condition configuration block
func (v *Validator) validateRunnersConditionBlock(block *Block) {
	// runners_condition must have exactly one label (the condition name)
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"runners_condition block must have exactly one label (the condition name)")
		return
	}

	// Validate condition name
	conditionName := block.Labels[0]
	if !isValidIdentifier(conditionName) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid condition name %q: must contain only alphanumeric characters, hyphens, and underscores", conditionName))
	}

	// Validate required attribute: eggs_entities (list of strings)
	eggsEntitiesVal, ok := block.GetAttribute("eggs_entities")
	if !ok {
		v.result.AddError(block.Position, "eggs_entities",
			"runners_condition block must have an 'eggs_entities' attribute")
	} else {
		eggsEntitiesList, err := eggsEntitiesVal.AsList()
		if err != nil {
			v.result.AddError(eggsEntitiesVal.Position, "eggs_entities",
				"eggs_entities must be a list")
		} else {
			if len(eggsEntitiesList) == 0 {
				v.result.AddError(eggsEntitiesVal.Position, "eggs_entities",
					"eggs_entities must contain at least one egg name")
			}
			for i, entity := range eggsEntitiesList {
				entityStr, err := entity.AsString()
				if err != nil {
					v.result.AddError(entity.Position, fmt.Sprintf("eggs_entities[%d]", i),
						"egg entity must be a string")
				} else if !isValidIdentifier(entityStr) {
					v.result.AddError(entity.Position, fmt.Sprintf("eggs_entities[%d]", i),
						fmt.Sprintf("invalid egg name %q: must contain only alphanumeric characters, hyphens, and underscores", entityStr))
				}
			}
		}
	}

	// Validate required nested blocks
	v.validateRequiredBlock(block, "apex")
	v.validateRequiredBlock(block, "nadir")

	// Validate apex block
	if apexBlock, ok := block.GetBlock("apex"); ok {
		v.validatePoolBlock(apexBlock, "apex")
	}

	// Validate nadir block
	if nadirBlock, ok := block.GetBlock("nadir"); ok {
		v.validatePoolBlock(nadirBlock, "nadir")
	}
}

// validateCloudBlock validates a cloud configuration block
func (v *Validator) validateCloudBlock(block *Block) {
	// Validate required attribute: provider
	providerVal, ok := block.GetAttribute("provider")
	if !ok {
		v.result.AddError(block.Position, "provider", "cloud block must have a 'provider' attribute")
	} else {
		providerStr, err := providerVal.AsString()
		if err != nil {
			v.result.AddError(providerVal.Position, "provider", "provider must be a string")
		} else if providerStr != "yandex" && providerStr != "aws" {
			v.result.AddError(providerVal.Position, "provider",
				fmt.Sprintf("provider must be 'yandex' or 'aws', got %q", providerStr))
		}
	}

	// Validate required attribute: region
	regionVal, ok := block.GetAttribute("region")
	if !ok {
		v.result.AddError(block.Position, "region", "cloud block must have a 'region' attribute")
	} else {
		_, err := regionVal.AsString()
		if err != nil {
			v.result.AddError(regionVal.Position, "region", "region must be a string")
		}
	}
}

// validateResourcesBlock validates a resources configuration block
func (v *Validator) validateResourcesBlock(block *Block) {
	// Validate required attributes
	v.validateRequiredNumberAttribute(block, "cpu", 1, 128)
	v.validateRequiredNumberAttribute(block, "memory", 512, 524288) // 512 MB to 512 GB
	v.validateRequiredNumberAttribute(block, "disk", 10, 10240)     // 10 GB to 10 TB

	typeVal, ok := block.GetAttribute("type")
	if ok {
		typeStr, err := typeVal.AsString()
		if err != nil {
			v.result.AddError(typeVal.Position, "type", "type must be a string")
		} else if typeStr != "vm" && typeStr != "serverless" {
			v.result.AddError(typeVal.Position, "type",
				fmt.Sprintf("type must be 'vm' or 'serverless', got %q", typeStr))
		}
	}
}

// validateRunnerBlock validates a runner configuration block
func (v *Validator) validateRunnerBlock(block *Block) {
	// Validate required attribute: tags (list of strings)
	tagsVal, ok := block.GetAttribute("tags")
	if !ok {
		v.result.AddError(block.Position, "tags", "runner block must have a 'tags' attribute")
	} else {
		tagsList, err := tagsVal.AsList()
		if err != nil {
			v.result.AddError(tagsVal.Position, "tags", "tags must be a list")
		} else {
			for i, tag := range tagsList {
				_, err := tag.AsString()
				if err != nil {
					v.result.AddError(tag.Position, fmt.Sprintf("tags[%d]", i),
						"tag must be a string")
				}
			}
		}
	}

	// Validate required attribute: concurrent
	v.validateRequiredNumberAttribute(block, "concurrent", 1, 100)

	// Validate optional attribute: idle_timeout
	if idleTimeoutVal, ok := block.GetAttribute("idle_timeout"); ok {
		_, err := idleTimeoutVal.AsString()
		if err != nil {
			v.result.AddError(idleTimeoutVal.Position, "idle_timeout",
				"idle_timeout must be a string (duration)")
		}
	}
}

// validateGitLabBlock validates a gitlab configuration block
func (v *Validator) validateGitLabBlock(block *Block) {
	// Validate required attribute: project_id
	v.validateRequiredNumberAttribute(block, "project_id", 1, 999999999)

	gitServer, ok := block.GetAttribute("server_name")
	if !ok {
		v.result.AddError(block.Position, "server_name",
			"gitlab block must have a 'server_name' attribute")
	} else {
		_, err := gitServer.AsString()
		if err != nil {
			v.result.AddError(gitServer.Position, "server_name",
				"server_name must be a string")
		}
	}

	// Validate required attribute: token_secret
	tokenSecretVal, ok := block.GetAttribute("token_secret")
	if !ok {
		v.result.AddError(block.Position, "token_secret",
			"gitlab block must have a 'token_secret' attribute")
	} else {
		_, err := tokenSecretVal.AsString()
		if err != nil {
			v.result.AddError(tokenSecretVal.Position, "token_secret",
				"token_secret must be a string")
		}
	}
}

// validateEnvironmentBlock validates an environment configuration block
func (v *Validator) validateEnvironmentBlock(block *Block) {
	// Environment block should only contain string attributes
	for name, val := range block.Attributes {
		_, err := val.AsString()
		if err != nil {
			v.result.AddError(val.Position, name,
				"environment variables must be strings")
		}
	}
}

// validateJobRunnerBlock validates a runner block within a job
func (v *Validator) validateJobRunnerBlock(block *Block) {
	// Validate required attribute: type
	typeVal, ok := block.GetAttribute("type")
	if !ok {
		v.result.AddError(block.Position, "type", "runner block must have a 'type' attribute")
	} else {
		typeStr, err := typeVal.AsString()
		if err != nil {
			v.result.AddError(typeVal.Position, "type", "type must be a string")
		} else if typeStr != "vm" && typeStr != "serverless" {
			v.result.AddError(typeVal.Position, "type",
				fmt.Sprintf("type must be 'vm' or 'serverless', got %q", typeStr))
		}
	}

	// Validate required attribute: tags
	tagsVal, ok := block.GetAttribute("tags")
	if !ok {
		v.result.AddError(block.Position, "tags", "runner block must have a 'tags' attribute")
	} else {
		tagsList, err := tagsVal.AsList()
		if err != nil {
			v.result.AddError(tagsVal.Position, "tags", "tags must be a list")
		} else {
			for i, tag := range tagsList {
				_, err := tag.AsString()
				if err != nil {
					v.result.AddError(tag.Position, fmt.Sprintf("tags[%d]", i),
						"tag must be a string")
				}
			}
		}
	}
}

// validatePruningBlock validates a pruning configuration block
func (v *Validator) validatePruningBlock(block *Block) {
	v.validateRequiredNumberAttribute(block, "failed_threshold", 1, 100)

	maxAgeVal, ok := block.GetAttribute("max_age")
	if !ok {
		v.result.AddError(block.Position, "max_age",
			"pruning block must have a 'max_age' attribute")
	} else {
		_, err := maxAgeVal.AsString()
		if err != nil {
			v.result.AddError(maxAgeVal.Position, "max_age",
				"max_age must be a string (duration)")
		}
	}

	checkIntervalVal, ok := block.GetAttribute("check_interval")
	if !ok {
		v.result.AddError(block.Position, "check_interval",
			"pruning block must have a 'check_interval' attribute")
	} else {
		_, err := checkIntervalVal.AsString()
		if err != nil {
			v.result.AddError(checkIntervalVal.Position, "check_interval",
				"check_interval must be a string (duration)")
		}
	}
}

// validatePoolBlock validates an apex or nadir pool configuration block
func (v *Validator) validatePoolBlock(block *Block, poolType string) {
	v.validateRequiredNumberAttribute(block, "max_count", 0, 1000)
	v.validateRequiredNumberAttribute(block, "min_count", 0, 1000)

	// Validate that min_count <= max_count
	minVal, minOk := block.GetAttribute("min_count")
	maxVal, maxOk := block.GetAttribute("max_count")
	if minOk && maxOk {
		minNum, minErr := minVal.AsInt()
		maxNum, maxErr := maxVal.AsInt()
		if minErr == nil && maxErr == nil && minNum > maxNum {
			v.result.AddError(block.Position, "min_count",
				fmt.Sprintf("min_count (%d) cannot be greater than max_count (%d)", minNum, maxNum))
		}
	}

	// Validate optional cpu_threshold for apex pools
	if poolType == "apex" {
		if cpuThresholdVal, ok := block.GetAttribute("cpu_threshold"); ok {
			cpuThreshold, err := cpuThresholdVal.AsNumber()
			if err != nil {
				v.result.AddError(cpuThresholdVal.Position, "cpu_threshold",
					"cpu_threshold must be a number")
			} else if cpuThreshold < 0 || cpuThreshold > 100 {
				v.result.AddError(cpuThresholdVal.Position, "cpu_threshold",
					fmt.Sprintf("cpu_threshold must be between 0 and 100, got %v", cpuThreshold))
			}
		}

		// Validate optional memory_threshold for apex pools
		if memoryThresholdVal, ok := block.GetAttribute("memory_threshold"); ok {
			memoryThreshold, err := memoryThresholdVal.AsNumber()
			if err != nil {
				v.result.AddError(memoryThresholdVal.Position, "memory_threshold",
					"memory_threshold must be a number")
			} else if memoryThreshold < 0 || memoryThreshold > 100 {
				v.result.AddError(memoryThresholdVal.Position, "memory_threshold",
					fmt.Sprintf("memory_threshold must be between 0 and 100, got %v", memoryThreshold))
			}
		}
	}

	// Nadir pool requires idle_timeout
	if poolType == "nadir" {
		idleTimeoutVal, ok := block.GetAttribute("idle_timeout")
		if !ok {
			v.result.AddError(block.Position, "idle_timeout",
				"nadir block must have an 'idle_timeout' attribute")
		} else {
			_, err := idleTimeoutVal.AsString()
			if err != nil {
				v.result.AddError(idleTimeoutVal.Position, "idle_timeout",
					"idle_timeout must be a string (duration)")
			}
		}
	}
}

// validatePoliciesBlock validates a policies configuration block
func (v *Validator) validatePoliciesBlock(block *Block) {
	// Policies block should contain rule blocks
	rules := block.GetBlocks("rule")
	if len(rules) == 0 {
		v.result.AddError(block.Position, "rules",
			"policies block must contain at least one rule")
	}

	for _, rule := range rules {
		v.validateRuleBlock(&rule)
	}
}

// validateRuleBlock validates a rule block within policies
func (v *Validator) validateRuleBlock(block *Block) {
	// Rule must have exactly one label (the rule name)
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"rule block must have exactly one label (the rule name)")
		return
	}

	// Validate required attribute: condition
	conditionVal, ok := block.GetAttribute("condition")
	if !ok {
		v.result.AddError(block.Position, "condition",
			"rule block must have a 'condition' attribute")
	} else {
		_, err := conditionVal.AsString()
		if err != nil {
			v.result.AddError(conditionVal.Position, "condition",
				"condition must be a string")
		}
	}

	// Validate required attribute: action
	actionVal, ok := block.GetAttribute("action")
	if !ok {
		v.result.AddError(block.Position, "action",
			"rule block must have an 'action' attribute")
	} else {
		actionStr, err := actionVal.AsString()
		if err != nil {
			v.result.AddError(actionVal.Position, "action",
				"action must be a string")
		} else {
			validActions := []string{"terminate", "demote_to_nadir", "promote_to_apex"}
			if !contains(validActions, actionStr) {
				v.result.AddError(actionVal.Position, "action",
					fmt.Sprintf("action must be one of %v, got %q", validActions, actionStr))
			}
		}
	}
}

// Helper functions

func (v *Validator) validateRequiredBlock(block *Block, blockType string) {
	if _, ok := block.GetBlock(blockType); !ok {
		v.result.AddError(block.Position, blockType,
			fmt.Sprintf("%s block must have a '%s' nested block", block.Type, blockType))
	}
}

func (v *Validator) validateRequiredNumberAttribute(block *Block, name string, min, max float64) {
	val, ok := block.GetAttribute(name)
	if !ok {
		v.result.AddError(block.Position, name,
			fmt.Sprintf("%s block must have a '%s' attribute", block.Type, name))
		return
	}

	num, err := val.AsNumber()
	if err != nil {
		v.result.AddError(val.Position, name,
			fmt.Sprintf("%s must be a number", name))
		return
	}

	if num < min || num > max {
		v.result.AddError(val.Position, name,
			fmt.Sprintf("%s must be between %v and %v, got %v", name, min, max, num))
	}
}

func isValidIdentifier(s string) bool {
	// Must contain only alphanumeric characters, hyphens, and underscores
	// Must start with a letter
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_-]*$`, s)
	return matched
}

func isValidCronExpression(s string) bool {
	// Basic cron validation: 5 or 6 fields separated by spaces
	// This is a simplified check; a full implementation would validate each field
	parts := strings.Fields(s)
	return len(parts) == 5 || len(parts) == 6
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
