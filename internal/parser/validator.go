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
	case "job":
		v.validateJobBlock(block)
	case "uglyfox":
		v.validateUglyFoxBlock(block)
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
	v.validateRequiredBlock(block, "apex")
	v.validateRequiredBlock(block, "nadir")

	// Validate pruning block
	if pruningBlock, ok := block.GetBlock("pruning"); ok {
		v.validatePruningBlock(pruningBlock)
	}

	// Validate apex block
	if apexBlock, ok := block.GetBlock("apex"); ok {
		v.validatePoolBlock(apexBlock, "apex")
	}

	// Validate nadir block
	if nadirBlock, ok := block.GetBlock("nadir"); ok {
		v.validatePoolBlock(nadirBlock, "nadir")
	}

	// Validate optional policies block
	if policiesBlock, ok := block.GetBlock("policies"); ok {
		v.validatePoliciesBlock(policiesBlock)
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
