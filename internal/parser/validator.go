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

// validateEggBlock validates an egg configuration block.
// Schema: flat attributes — no nested cloud/resources/runner/gitlab blocks.
func (v *Validator) validateEggBlock(block *Block) {
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"egg block must have exactly one label (the egg name)")
		return
	}
	if !isValidIdentifier(block.Labels[0]) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid egg name %q: must start with a letter and contain only alphanumeric characters, hyphens, and underscores", block.Labels[0]))
	}

	// Required string attributes
	v.validateRequiredString(block, "gitlab_server")
	v.validateRequiredString(block, "region")

	// Required secret URIs
	v.validateRequiredSecretURI(block, "gitlab_token_secret")
	v.validateRequiredSecretURI(block, "gitlab_webhook_secret")
	v.validateRequiredSecretURI(block, "git_repo_url_secret")

	// Required: project_id (number >= 1)
	v.validateRequiredNumberAttribute(block, "project_id", 1, 999999999)

	// Required: cloud_provider
	v.validateRequiredEnum(block, "cloud_provider", []string{"yandex", "aws"})

	// Required: runner_type
	v.validateRequiredEnum(block, "runner_type", []string{"serverless", "apex", "nadir"})

	// Optional: cpu (float > 0)
	if cpuVal, ok := block.GetAttribute("cpu"); ok {
		if _, err := cpuVal.AsNumber(); err != nil {
			v.result.AddError(cpuVal.Position, "cpu", "cpu must be a number (e.g. 0.5, 1, 2)")
		}
	}

	// Optional: memory (string with unit, e.g. "512MB", "1GB")
	if memVal, ok := block.GetAttribute("memory"); ok {
		if s, err := memVal.AsString(); err != nil {
			v.result.AddError(memVal.Position, "memory", "memory must be a string (e.g. \"512MB\", \"1GB\")")
		} else if !isValidMemory(s) {
			v.result.AddError(memVal.Position, "memory",
				fmt.Sprintf("invalid memory value %q: expected format like \"512MB\" or \"1GB\"", s))
		}
	}

	// Optional: max_concurrent_jobs (number >= 1)
	if mjVal, ok := block.GetAttribute("max_concurrent_jobs"); ok {
		if n, err := mjVal.AsNumber(); err != nil {
			v.result.AddError(mjVal.Position, "max_concurrent_jobs", "max_concurrent_jobs must be a number")
		} else if n < 1 {
			v.result.AddError(mjVal.Position, "max_concurrent_jobs", "max_concurrent_jobs must be >= 1")
		}
	}

	// Optional: tags (list of strings)
	if tagsVal, ok := block.GetAttribute("tags"); ok {
		v.validateStringList(tagsVal, "tags")
	}

	// Optional: environment (list of strings)
	if envVal, ok := block.GetAttribute("environment"); ok {
		v.validateStringList(envVal, "environment")
	}
}

// validateEggsBucketBlock validates an eggsbucket configuration block.
// Schema: flat attributes — group_id instead of project_id, no git_repo_url_secret.
func (v *Validator) validateEggsBucketBlock(block *Block) {
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"eggsbucket block must have exactly one label (the bucket name)")
		return
	}
	if !isValidIdentifier(block.Labels[0]) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid eggsbucket name %q: must start with a letter and contain only alphanumeric characters, hyphens, and underscores", block.Labels[0]))
	}

	v.validateRequiredString(block, "gitlab_server")
	v.validateRequiredString(block, "region")
	v.validateRequiredSecretURI(block, "gitlab_token_secret")
	v.validateRequiredSecretURI(block, "gitlab_webhook_secret")
	v.validateRequiredNumberAttribute(block, "group_id", 1, 999999999)
	v.validateRequiredEnum(block, "cloud_provider", []string{"yandex", "aws"})
	v.validateRequiredEnum(block, "runner_type", []string{"serverless", "apex", "nadir"})

	// Optional: cpu, memory, tags, project_ids
	if cpuVal, ok := block.GetAttribute("cpu"); ok {
		if _, err := cpuVal.AsNumber(); err != nil {
			v.result.AddError(cpuVal.Position, "cpu", "cpu must be a number (e.g. 0.5, 1, 2)")
		}
	}
	if memVal, ok := block.GetAttribute("memory"); ok {
		if s, err := memVal.AsString(); err != nil {
			v.result.AddError(memVal.Position, "memory", "memory must be a string (e.g. \"512MB\", \"1GB\")")
		} else if !isValidMemory(s) {
			v.result.AddError(memVal.Position, "memory",
				fmt.Sprintf("invalid memory value %q: expected format like \"512MB\" or \"1GB\"", s))
		}
	}
	if tagsVal, ok := block.GetAttribute("tags"); ok {
		v.validateStringList(tagsVal, "tags")
	}
	if pidsVal, ok := block.GetAttribute("project_ids"); ok {
		if list, err := pidsVal.AsList(); err != nil {
			v.result.AddError(pidsVal.Position, "project_ids", "project_ids must be a list")
		} else {
			for i, item := range list {
				if _, err := item.AsNumber(); err != nil {
					v.result.AddError(item.Position, fmt.Sprintf("project_ids[%d]", i),
						"project_ids entries must be numbers")
				}
			}
		}
	}
}

// validateJobBlock validates a job configuration block.
// Schema: flat attributes — schedule, script, cloud_provider, region (required).
func (v *Validator) validateJobBlock(block *Block) {
	if len(block.Labels) != 1 {
		v.result.AddError(block.Position, "labels",
			"job block must have exactly one label (the job name)")
		return
	}
	if !isValidIdentifier(block.Labels[0]) {
		v.result.AddError(block.Position, "name",
			fmt.Sprintf("invalid job name %q: must start with a letter and contain only alphanumeric characters, hyphens, and underscores", block.Labels[0]))
	}

	// Required: schedule (cron)
	if schedVal, ok := block.GetAttribute("schedule"); !ok {
		v.result.AddError(block.Position, "schedule", "job block must have a 'schedule' attribute")
	} else if s, err := schedVal.AsString(); err != nil {
		v.result.AddError(schedVal.Position, "schedule", "schedule must be a string (cron expression)")
	} else if !isValidCronExpression(s) {
		v.result.AddError(schedVal.Position, "schedule",
			fmt.Sprintf("invalid cron expression: %q", s))
	}

	// Required: script
	v.validateRequiredString(block, "script")

	// Required: cloud_provider, region
	v.validateRequiredEnum(block, "cloud_provider", []string{"yandex", "aws"})
	v.validateRequiredString(block, "region")

	// Optional: runner_image, cpu, memory, timeout, environment, secrets
	if memVal, ok := block.GetAttribute("memory"); ok {
		if s, err := memVal.AsString(); err != nil {
			v.result.AddError(memVal.Position, "memory", "memory must be a string (e.g. \"256MB\")")
		} else if !isValidMemory(s) {
			v.result.AddError(memVal.Position, "memory",
				fmt.Sprintf("invalid memory value %q: expected format like \"256MB\" or \"1GB\"", s))
		}
	}
	if timeoutVal, ok := block.GetAttribute("timeout"); ok {
		if s, err := timeoutVal.AsString(); err != nil {
			v.result.AddError(timeoutVal.Position, "timeout", "timeout must be a duration string (e.g. \"30m\", \"1h\")")
		} else if !isValidDuration(s) {
			v.result.AddError(timeoutVal.Position, "timeout",
				fmt.Sprintf("invalid duration %q: expected format like \"30s\", \"5m\", \"2h\"", s))
		}
	}
	if secretsVal, ok := block.GetAttribute("secrets"); ok {
		if list, err := secretsVal.AsList(); err != nil {
			v.result.AddError(secretsVal.Position, "secrets", "secrets must be a list")
		} else {
			for i, item := range list {
				if s, err := item.AsString(); err != nil {
					v.result.AddError(item.Position, fmt.Sprintf("secrets[%d]", i), "secret must be a string")
				} else if !isValidSecretURI(s) {
					v.result.AddError(item.Position, fmt.Sprintf("secrets[%d]", i),
						fmt.Sprintf("invalid secret URI %q: must use yc-lockbox://, aws-sm://, or vault:// scheme", s))
				}
			}
		}
	}
}

// validateUglyFoxBlock validates an uglyfox configuration block.
// Schema: pruning {}, apex_pool {}, nadir_pool {}, runners_condition {} nested blocks.
func (v *Validator) validateUglyFoxBlock(block *Block) {
	if len(block.Labels) > 0 {
		v.result.AddError(block.Position, "labels", "uglyfox block should not have labels")
	}

	// Required: pruning block
	if pruningBlock, ok := block.GetBlock("pruning"); ok {
		v.validateUFPruningBlock(pruningBlock)
	} else {
		v.result.AddError(block.Position, "pruning", "uglyfox block must have a 'pruning' nested block")
	}

	// Required: apex_pool block
	if apexBlock, ok := block.GetBlock("apex_pool"); ok {
		v.validateUFPoolBlock(apexBlock, "apex_pool")
	} else {
		v.result.AddError(block.Position, "apex_pool", "uglyfox block must have an 'apex_pool' nested block")
	}

	// Required: nadir_pool block
	if nadirBlock, ok := block.GetBlock("nadir_pool"); ok {
		v.validateUFPoolBlock(nadirBlock, "nadir_pool")
	} else {
		v.result.AddError(block.Position, "nadir_pool", "uglyfox block must have a 'nadir_pool' nested block")
	}

	// Optional: runners_condition blocks (zero or more, no label)
	for i := range block.GetBlocks("runners_condition") {
		rc := block.GetBlocks("runners_condition")[i]
		v.validateUFRunnersConditionBlock(&rc)
	}
}

// validateUFPruningBlock validates the pruning block inside uglyfox.
func (v *Validator) validateUFPruningBlock(block *Block) {
	v.validateRequiredNumberAttribute(block, "max_age_hours", 1, 8760)
	v.validateRequiredNumberAttribute(block, "max_failures", 1, 1000)
	v.validateRequiredNumberAttribute(block, "idle_timeout_minutes", 1, 10080)
	v.validateRequiredNumberAttribute(block, "check_interval_seconds", 1, 86400)
}

// validateUFPoolBlock validates apex_pool or nadir_pool blocks inside uglyfox.
func (v *Validator) validateUFPoolBlock(block *Block, poolType string) {
	v.validateRequiredNumberAttribute(block, "min_size", 0, 1000)
	v.validateRequiredNumberAttribute(block, "max_size", 0, 1000)

	minVal, minOk := block.GetAttribute("min_size")
	maxVal, maxOk := block.GetAttribute("max_size")
	if minOk && maxOk {
		minN, minErr := minVal.AsInt()
		maxN, maxErr := maxVal.AsInt()
		if minErr == nil && maxErr == nil && minN > maxN {
			v.result.AddError(block.Position, "min_size",
				fmt.Sprintf("min_size (%d) cannot be greater than max_size (%d)", minN, maxN))
		}
	}

	if poolType == "apex_pool" {
		v.validateRequiredNumberAttribute(block, "scale_up_threshold", 1, 1000)
	}
	if poolType == "nadir_pool" {
		v.validateRequiredNumberAttribute(block, "warmup_time_seconds", 0, 86400)
	}
}

// validateUFRunnersConditionBlock validates a runners_condition block inside uglyfox.
func (v *Validator) validateUFRunnersConditionBlock(block *Block) {
	if len(block.Labels) > 0 {
		v.result.AddError(block.Position, "labels", "runners_condition block should not have labels")
	}
	v.validateRequiredString(block, "egg_name")
	if maxFail, ok := block.GetAttribute("max_failures"); ok {
		if n, err := maxFail.AsNumber(); err != nil || n < 1 {
			v.result.AddError(maxFail.Position, "max_failures", "max_failures must be a number >= 1")
		}
	}
	if maxAge, ok := block.GetAttribute("max_age_hours"); ok {
		if _, err := maxAge.AsNumber(); err != nil {
			v.result.AddError(maxAge.Position, "max_age_hours", "max_age_hours must be a number")
		}
	}
}

// validateMotherGooseBlock validates a mothergoose configuration block.
// Schema: api_gateway {}, message_queue {}, cloud_trigger {}, container {} nested blocks.
func (v *Validator) validateMotherGooseBlock(block *Block) {
	if len(block.Labels) > 0 {
		v.result.AddError(block.Position, "labels", "mothergoose block should not have labels")
	}

	// Required: api_gateway
	if agBlock, ok := block.GetBlock("api_gateway"); ok {
		v.validateMGAPIGatewayBlock(agBlock)
	} else {
		v.result.AddError(block.Position, "api_gateway", "mothergoose block must have an 'api_gateway' nested block")
	}

	// Required: message_queue
	if mqBlock, ok := block.GetBlock("message_queue"); ok {
		v.validateMGMessageQueueBlock(mqBlock)
	} else {
		v.result.AddError(block.Position, "message_queue", "mothergoose block must have a 'message_queue' nested block")
	}

	// Required: cloud_trigger
	if ctBlock, ok := block.GetBlock("cloud_trigger"); ok {
		v.validateMGCloudTriggerBlock(ctBlock)
	} else {
		v.result.AddError(block.Position, "cloud_trigger", "mothergoose block must have a 'cloud_trigger' nested block")
	}

	// Required: container
	if cBlock, ok := block.GetBlock("container"); ok {
		v.validateMGContainerBlock(cBlock)
	} else {
		v.result.AddError(block.Position, "container", "mothergoose block must have a 'container' nested block")
	}
}

// validateMGAPIGatewayBlock validates the api_gateway block inside mothergoose.
func (v *Validator) validateMGAPIGatewayBlock(block *Block) {
	v.validateRequiredEnum(block, "cloud_provider", []string{"yandex", "aws"})
	v.validateRequiredString(block, "region")
	v.validateRequiredString(block, "domain")

	if tlsVal, ok := block.GetAttribute("tls"); ok {
		if _, err := tlsVal.AsBool(); err != nil {
			v.result.AddError(tlsVal.Position, "tls", "tls must be a bool")
		}
	}
}

// validateMGMessageQueueBlock validates the message_queue block inside mothergoose.
func (v *Validator) validateMGMessageQueueBlock(block *Block) {
	v.validateRequiredEnum(block, "cloud_provider", []string{"yandex", "aws"})
	v.validateRequiredString(block, "queue_name")
}

// validateMGCloudTriggerBlock validates the cloud_trigger block inside mothergoose.
func (v *Validator) validateMGCloudTriggerBlock(block *Block) {
	v.validateRequiredEnum(block, "type", []string{"timer", "webhook"})
	v.validateRequiredString(block, "target")

	if schedVal, ok := block.GetAttribute("schedule"); ok {
		if s, err := schedVal.AsString(); err != nil {
			v.result.AddError(schedVal.Position, "schedule", "schedule must be a string (cron expression)")
		} else if !isValidCronExpression(s) {
			v.result.AddError(schedVal.Position, "schedule",
				fmt.Sprintf("invalid cron expression: %q", s))
		}
	}
}

// validateMGContainerBlock validates the container block inside mothergoose.
func (v *Validator) validateMGContainerBlock(block *Block) {
	v.validateRequiredString(block, "image")

	if cpuVal, ok := block.GetAttribute("cpu"); ok {
		if _, err := cpuVal.AsNumber(); err != nil {
			v.result.AddError(cpuVal.Position, "cpu", "cpu must be a number")
		}
	}
	if memVal, ok := block.GetAttribute("memory"); ok {
		if s, err := memVal.AsString(); err != nil {
			v.result.AddError(memVal.Position, "memory", "memory must be a string (e.g. \"512MB\")")
		} else if !isValidMemory(s) {
			v.result.AddError(memVal.Position, "memory",
				fmt.Sprintf("invalid memory value %q: expected format like \"512MB\" or \"1GB\"", s))
		}
	}
	if minVal, ok := block.GetAttribute("min_instances"); ok {
		if n, err := minVal.AsNumber(); err != nil || n < 0 {
			v.result.AddError(minVal.Position, "min_instances", "min_instances must be a number >= 0")
		}
	}
	if maxVal, ok := block.GetAttribute("max_instances"); ok {
		if n, err := maxVal.AsNumber(); err != nil || n < 1 {
			v.result.AddError(maxVal.Position, "max_instances", "max_instances must be a number >= 1")
		}
	}
}

// --- helpers ---

func (v *Validator) validateRequiredString(block *Block, name string) {
	val, ok := block.GetAttribute(name)
	if !ok {
		v.result.AddError(block.Position, name,
			fmt.Sprintf("%s block must have a '%s' attribute", block.Type, name))
		return
	}
	if _, err := val.AsString(); err != nil {
		v.result.AddError(val.Position, name, fmt.Sprintf("%s must be a string", name))
	}
}

func (v *Validator) validateRequiredSecretURI(block *Block, name string) {
	val, ok := block.GetAttribute(name)
	if !ok {
		v.result.AddError(block.Position, name,
			fmt.Sprintf("%s block must have a '%s' attribute", block.Type, name))
		return
	}
	s, err := val.AsString()
	if err != nil {
		v.result.AddError(val.Position, name, fmt.Sprintf("%s must be a string", name))
		return
	}
	if !isValidSecretURI(s) {
		v.result.AddError(val.Position, name,
			fmt.Sprintf("invalid secret URI %q: must use yc-lockbox://, aws-sm://, or vault:// scheme", s))
	}
}

func (v *Validator) validateRequiredEnum(block *Block, name string, allowed []string) {
	val, ok := block.GetAttribute(name)
	if !ok {
		v.result.AddError(block.Position, name,
			fmt.Sprintf("%s block must have a '%s' attribute", block.Type, name))
		return
	}
	s, err := val.AsString()
	if err != nil {
		v.result.AddError(val.Position, name, fmt.Sprintf("%s must be a string", name))
		return
	}
	for _, a := range allowed {
		if s == a {
			return
		}
	}
	v.result.AddError(val.Position, name,
		fmt.Sprintf("%s must be one of %v, got %q", name, allowed, s))
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
		v.result.AddError(val.Position, name, fmt.Sprintf("%s must be a number", name))
		return
	}
	if num < min || num > max {
		v.result.AddError(val.Position, name,
			fmt.Sprintf("%s must be between %v and %v, got %v", name, min, max, num))
	}
}

func (v *Validator) validateStringList(val Value, name string) {
	list, err := val.AsList()
	if err != nil {
		v.result.AddError(val.Position, name, fmt.Sprintf("%s must be a list", name))
		return
	}
	for i, item := range list {
		if _, err := item.AsString(); err != nil {
			v.result.AddError(item.Position, fmt.Sprintf("%s[%d]", name, i),
				fmt.Sprintf("%s entries must be strings", name))
		}
	}
}

// isValidIdentifier checks name starts with a letter and contains only alphanumeric, hyphens, underscores.
func isValidIdentifier(s string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_-]*$`, s)
	return matched
}

// isValidCronExpression does a basic 5-field cron check.
func isValidCronExpression(s string) bool {
	parts := strings.Fields(s)
	return len(parts) == 5 || len(parts) == 6
}

// isValidSecretURI checks for yc-lockbox://, aws-sm://, or vault:// prefix.
func isValidSecretURI(s string) bool {
	return strings.HasPrefix(s, "yc-lockbox://") ||
		strings.HasPrefix(s, "aws-sm://") ||
		strings.HasPrefix(s, "vault://")
}

// isValidMemory checks for a number followed by MB or GB (case-insensitive).
func isValidMemory(s string) bool {
	matched, _ := regexp.MatchString(`(?i)^\d+(\.\d+)?(MB|GB)$`, s)
	return matched
}

// isValidDuration checks for a number followed by s, m, h, or d.
func isValidDuration(s string) bool {
	matched, _ := regexp.MatchString(`^\d+(\.\d+)?(s|m|h|d)$`, s)
	return matched
}
