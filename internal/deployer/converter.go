package deployer

import (
	"fmt"
	"time"

	"github.com/polar-gosling/gosling/internal/parser"
)

// Converter converts .fly configuration to deployment configurations
// These configurations are passed to MotherGoose, which uses OpenTofu to deploy runners.
// Gosling CLI does not deploy runners directly - it only parses and converts configurations.
type Converter struct{}

// NewConverter creates a new converter instance
func NewConverter() *Converter {
	return &Converter{}
}

// CloudInfo represents cloud configuration from parser
type CloudInfo struct {
	Provider string
	Region   string
}

// ResourceInfo represents resource configuration from parser
type ResourceInfo struct {
	CPU    int
	Memory int
	Disk   int
}

// RunnerInfo represents runner configuration from parser
type RunnerInfo struct {
	Tags        []string
	Concurrent  int
	IdleTimeout string
}

// GitLabInfo represents GitLab configuration from parser
type GitLabInfo struct {
	ProjectID   int
	TokenSecret string
}

// RepositoryInfo represents a repository in an EggsBucket from parser
type RepositoryInfo struct {
	Name   string
	GitLab GitLabInfo
}

// ParsedEggConfig represents a parsed Egg configuration
type ParsedEggConfig struct {
	Name        string
	Type        string
	Cloud       CloudInfo
	Resources   ResourceInfo
	Runner      RunnerInfo
	GitLab      GitLabInfo
	Environment map[string]string
}

// ParsedEggsBucketConfig represents a parsed EggsBucket configuration
type ParsedEggsBucketConfig struct {
	Name         string
	Type         string
	Cloud        CloudInfo
	Resources    ResourceInfo
	Runner       RunnerInfo
	Repositories []RepositoryInfo
	Environment  map[string]string
}

// ParseEgg parses an Egg block into a ParsedEggConfig
func ParseEgg(block *parser.Block) (*ParsedEggConfig, error) {
	if block.Type != "egg" {
		return nil, fmt.Errorf("expected 'egg' block, got '%s'", block.Type)
	}

	if len(block.Labels) == 0 {
		return nil, fmt.Errorf("egg block must have a name label")
	}

	egg := &ParsedEggConfig{
		Name:        block.Labels[0],
		Environment: make(map[string]string),
	}

	// Parse type
	if typeVal, ok := block.GetAttribute("type"); ok {
		typeStr, err := typeVal.AsString()
		if err != nil {
			return nil, fmt.Errorf("invalid type: %w", err)
		}
		egg.Type = typeStr
	}

	// Parse cloud block
	if cloudBlock, ok := block.GetBlock("cloud"); ok {
		cloud, err := parseCloudBlock(cloudBlock)
		if err != nil {
			return nil, err
		}
		egg.Cloud = cloud
	}

	// Parse resources block
	if resourcesBlock, ok := block.GetBlock("resources"); ok {
		resources, err := parseResourcesBlock(resourcesBlock)
		if err != nil {
			return nil, err
		}
		egg.Resources = resources
	}

	// Parse runner block
	if runnerBlock, ok := block.GetBlock("runner"); ok {
		runner, err := parseRunnerBlock(runnerBlock)
		if err != nil {
			return nil, err
		}
		egg.Runner = runner
	}

	// Parse gitlab block
	if gitlabBlock, ok := block.GetBlock("gitlab"); ok {
		gitlab, err := parseGitLabBlock(gitlabBlock)
		if err != nil {
			return nil, err
		}
		egg.GitLab = gitlab
	}

	// Parse environment block
	if envBlock, ok := block.GetBlock("environment"); ok {
		env, err := parseEnvironmentBlock(envBlock)
		if err != nil {
			return nil, err
		}
		egg.Environment = env
	}

	return egg, nil
}

// ParseEggsBucket parses an EggsBucket block into a ParsedEggsBucketConfig
func ParseEggsBucket(block *parser.Block) (*ParsedEggsBucketConfig, error) {
	if block.Type != "eggsbucket" {
		return nil, fmt.Errorf("expected 'eggsbucket' block, got '%s'", block.Type)
	}

	if len(block.Labels) == 0 {
		return nil, fmt.Errorf("eggsbucket block must have a name label")
	}

	bucket := &ParsedEggsBucketConfig{
		Name:        block.Labels[0],
		Environment: make(map[string]string),
	}

	// Parse type
	if typeVal, ok := block.GetAttribute("type"); ok {
		typeStr, err := typeVal.AsString()
		if err != nil {
			return nil, fmt.Errorf("invalid type: %w", err)
		}
		bucket.Type = typeStr
	}

	// Parse cloud block
	if cloudBlock, ok := block.GetBlock("cloud"); ok {
		cloud, err := parseCloudBlock(cloudBlock)
		if err != nil {
			return nil, err
		}
		bucket.Cloud = cloud
	}

	// Parse resources block
	if resourcesBlock, ok := block.GetBlock("resources"); ok {
		resources, err := parseResourcesBlock(resourcesBlock)
		if err != nil {
			return nil, err
		}
		bucket.Resources = resources
	}

	// Parse runner block
	if runnerBlock, ok := block.GetBlock("runner"); ok {
		runner, err := parseRunnerBlock(runnerBlock)
		if err != nil {
			return nil, err
		}
		bucket.Runner = runner
	}

	// Parse repositories block
	if repositoriesBlock, ok := block.GetBlock("repositories"); ok {
		repos, err := parseRepositoriesBlock(repositoriesBlock)
		if err != nil {
			return nil, err
		}
		bucket.Repositories = repos
	}

	// Parse environment block
	if envBlock, ok := block.GetBlock("environment"); ok {
		env, err := parseEnvironmentBlock(envBlock)
		if err != nil {
			return nil, err
		}
		bucket.Environment = env
	}

	return bucket, nil
}

// Helper functions to parse nested blocks

func parseCloudBlock(block *parser.Block) (CloudInfo, error) {
	cloud := CloudInfo{}

	if providerVal, ok := block.GetAttribute("provider"); ok {
		provider, err := providerVal.AsString()
		if err != nil {
			return cloud, fmt.Errorf("invalid provider: %w", err)
		}
		cloud.Provider = provider
	}

	if regionVal, ok := block.GetAttribute("region"); ok {
		region, err := regionVal.AsString()
		if err != nil {
			return cloud, fmt.Errorf("invalid region: %w", err)
		}
		cloud.Region = region
	}

	return cloud, nil
}

func parseResourcesBlock(block *parser.Block) (ResourceInfo, error) {
	resources := ResourceInfo{}

	if cpuVal, ok := block.GetAttribute("cpu"); ok {
		cpu, err := cpuVal.AsInt()
		if err != nil {
			return resources, fmt.Errorf("invalid cpu: %w", err)
		}
		resources.CPU = cpu
	}

	if memoryVal, ok := block.GetAttribute("memory"); ok {
		memory, err := memoryVal.AsInt()
		if err != nil {
			return resources, fmt.Errorf("invalid memory: %w", err)
		}
		resources.Memory = memory
	}

	if diskVal, ok := block.GetAttribute("disk"); ok {
		disk, err := diskVal.AsInt()
		if err != nil {
			return resources, fmt.Errorf("invalid disk: %w", err)
		}
		resources.Disk = disk
	}

	return resources, nil
}

func parseRunnerBlock(block *parser.Block) (RunnerInfo, error) {
	runner := RunnerInfo{}

	if tagsVal, ok := block.GetAttribute("tags"); ok {
		tagsList, err := tagsVal.AsList()
		if err != nil {
			return runner, fmt.Errorf("invalid tags: %w", err)
		}
		tags := make([]string, len(tagsList))
		for i, tagVal := range tagsList {
			tag, err := tagVal.AsString()
			if err != nil {
				return runner, fmt.Errorf("invalid tag at index %d: %w", i, err)
			}
			tags[i] = tag
		}
		runner.Tags = tags
	}

	if concurrentVal, ok := block.GetAttribute("concurrent"); ok {
		concurrent, err := concurrentVal.AsInt()
		if err != nil {
			return runner, fmt.Errorf("invalid concurrent: %w", err)
		}
		runner.Concurrent = concurrent
	}

	if idleTimeoutVal, ok := block.GetAttribute("idle_timeout"); ok {
		idleTimeout, err := idleTimeoutVal.AsString()
		if err != nil {
			return runner, fmt.Errorf("invalid idle_timeout: %w", err)
		}
		runner.IdleTimeout = idleTimeout
	}

	return runner, nil
}

func parseGitLabBlock(block *parser.Block) (GitLabInfo, error) {
	gitlab := GitLabInfo{}

	if projectIDVal, ok := block.GetAttribute("project_id"); ok {
		projectID, err := projectIDVal.AsInt()
		if err != nil {
			return gitlab, fmt.Errorf("invalid project_id: %w", err)
		}
		gitlab.ProjectID = projectID
	}

	if tokenSecretVal, ok := block.GetAttribute("token_secret"); ok {
		tokenSecret, err := tokenSecretVal.AsString()
		if err != nil {
			return gitlab, fmt.Errorf("invalid token_secret: %w", err)
		}
		gitlab.TokenSecret = tokenSecret
	}

	return gitlab, nil
}

func parseEnvironmentBlock(block *parser.Block) (map[string]string, error) {
	env := make(map[string]string)

	for key, val := range block.Attributes {
		strVal, err := val.AsString()
		if err != nil {
			return nil, fmt.Errorf("invalid environment variable %s: %w", key, err)
		}
		env[key] = strVal
	}

	return env, nil
}

func parseRepositoriesBlock(block *parser.Block) ([]RepositoryInfo, error) {
	repoBlocks := block.GetBlocks("repo")
	repos := make([]RepositoryInfo, len(repoBlocks))

	for i, repoBlock := range repoBlocks {
		if len(repoBlock.Labels) == 0 {
			return nil, fmt.Errorf("repo block must have a name label")
		}

		repo := RepositoryInfo{
			Name: repoBlock.Labels[0],
		}

		// Parse gitlab block
		if gitlabBlock, ok := repoBlock.GetBlock("gitlab"); ok {
			gitlab, err := parseGitLabBlock(gitlabBlock)
			if err != nil {
				return nil, fmt.Errorf("invalid gitlab block in repo %s: %w", repo.Name, err)
			}
			repo.GitLab = gitlab
		}

		repos[i] = repo
	}

	return repos, nil
}

// EggToVMConfig converts a parsed Egg configuration to a VM deployment configuration
func (c *Converter) EggToVMConfig(egg *ParsedEggConfig) (*VMConfig, error) {
	if egg.Type != "vm" {
		return nil, fmt.Errorf("egg type must be 'vm', got '%s'", egg.Type)
	}

	// Parse cloud provider
	provider, err := parseCloudProvider(egg.Cloud.Provider)
	if err != nil {
		return nil, err
	}

	// Parse idle timeout
	idleTimeout, err := time.ParseDuration(egg.Runner.IdleTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid idle timeout: %w", err)
	}

	return &VMConfig{
		EggName: egg.Name,
		Cloud: CloudConfig{
			Provider: provider,
			Region:   egg.Cloud.Region,
		},
		Resources: ResourceConfig{
			CPU:    egg.Resources.CPU,
			Memory: egg.Resources.Memory,
			Disk:   egg.Resources.Disk,
		},
		Runner: RunnerConfig{
			Tags:        egg.Runner.Tags,
			Concurrent:  egg.Runner.Concurrent,
			IdleTimeout: idleTimeout,
		},
		GitLab: GitLabConfig{
			ProjectID:   egg.GitLab.ProjectID,
			TokenSecret: egg.GitLab.TokenSecret,
		},
		Environment: egg.Environment,
	}, nil
}

// EggToServerlessConfig converts a parsed Egg configuration to a serverless deployment configuration
func (c *Converter) EggToServerlessConfig(egg *ParsedEggConfig) (*ServerlessConfig, error) {
	if egg.Type != "serverless" {
		return nil, fmt.Errorf("egg type must be 'serverless', got '%s'", egg.Type)
	}

	// Parse cloud provider
	provider, err := parseCloudProvider(egg.Cloud.Provider)
	if err != nil {
		return nil, err
	}

	// Parse idle timeout
	idleTimeout, err := time.ParseDuration(egg.Runner.IdleTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid idle timeout: %w", err)
	}

	// Serverless runners have a maximum timeout of 60 minutes
	timeout := 60 * time.Minute

	return &ServerlessConfig{
		EggName: egg.Name,
		Cloud: CloudConfig{
			Provider: provider,
			Region:   egg.Cloud.Region,
		},
		Resources: ResourceConfig{
			CPU:    egg.Resources.CPU,
			Memory: egg.Resources.Memory,
			Disk:   egg.Resources.Disk,
		},
		Runner: RunnerConfig{
			Tags:        egg.Runner.Tags,
			Concurrent:  egg.Runner.Concurrent,
			IdleTimeout: idleTimeout,
		},
		GitLab: GitLabConfig{
			ProjectID:   egg.GitLab.ProjectID,
			TokenSecret: egg.GitLab.TokenSecret,
		},
		Environment: egg.Environment,
		Timeout:     timeout,
	}, nil
}

// EggsBucketToVMConfigs converts a parsed EggsBucket configuration to multiple VM deployment configurations
func (c *Converter) EggsBucketToVMConfigs(bucket *ParsedEggsBucketConfig) ([]*VMConfig, error) {
	if bucket.Type != "vm" {
		return nil, fmt.Errorf("eggsbucket type must be 'vm', got '%s'", bucket.Type)
	}

	// Parse cloud provider
	provider, err := parseCloudProvider(bucket.Cloud.Provider)
	if err != nil {
		return nil, err
	}

	// Parse idle timeout
	idleTimeout, err := time.ParseDuration(bucket.Runner.IdleTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid idle timeout: %w", err)
	}

	// Create a VM config for each repository in the bucket
	configs := make([]*VMConfig, len(bucket.Repositories))
	for i, repo := range bucket.Repositories {
		configs[i] = &VMConfig{
			EggName: fmt.Sprintf("%s-%s", bucket.Name, repo.Name),
			Cloud: CloudConfig{
				Provider: provider,
				Region:   bucket.Cloud.Region,
			},
			Resources: ResourceConfig{
				CPU:    bucket.Resources.CPU,
				Memory: bucket.Resources.Memory,
				Disk:   bucket.Resources.Disk,
			},
			Runner: RunnerConfig{
				Tags:        bucket.Runner.Tags,
				Concurrent:  bucket.Runner.Concurrent,
				IdleTimeout: idleTimeout,
			},
			GitLab: GitLabConfig{
				ProjectID:   repo.GitLab.ProjectID,
				TokenSecret: repo.GitLab.TokenSecret,
			},
			Environment: bucket.Environment,
		}
	}

	return configs, nil
}

// EggsBucketToServerlessConfigs converts a parsed EggsBucket configuration to multiple serverless deployment configurations
func (c *Converter) EggsBucketToServerlessConfigs(bucket *ParsedEggsBucketConfig) ([]*ServerlessConfig, error) {
	if bucket.Type != "serverless" {
		return nil, fmt.Errorf("eggsbucket type must be 'serverless', got '%s'", bucket.Type)
	}

	// Parse cloud provider
	provider, err := parseCloudProvider(bucket.Cloud.Provider)
	if err != nil {
		return nil, err
	}

	// Parse idle timeout
	idleTimeout, err := time.ParseDuration(bucket.Runner.IdleTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid idle timeout: %w", err)
	}

	// Serverless runners have a maximum timeout of 60 minutes
	timeout := 60 * time.Minute

	// Create a serverless config for each repository in the bucket
	configs := make([]*ServerlessConfig, len(bucket.Repositories))
	for i, repo := range bucket.Repositories {
		configs[i] = &ServerlessConfig{
			EggName: fmt.Sprintf("%s-%s", bucket.Name, repo.Name),
			Cloud: CloudConfig{
				Provider: provider,
				Region:   bucket.Cloud.Region,
			},
			Resources: ResourceConfig{
				CPU:    bucket.Resources.CPU,
				Memory: bucket.Resources.Memory,
				Disk:   bucket.Resources.Disk,
			},
			Runner: RunnerConfig{
				Tags:        bucket.Runner.Tags,
				Concurrent:  bucket.Runner.Concurrent,
				IdleTimeout: idleTimeout,
			},
			GitLab: GitLabConfig{
				ProjectID:   repo.GitLab.ProjectID,
				TokenSecret: repo.GitLab.TokenSecret,
			},
			Environment: bucket.Environment,
			Timeout:     timeout,
		}
	}

	return configs, nil
}

// parseCloudProvider converts a string cloud provider to CloudProvider type
func parseCloudProvider(provider string) (CloudProvider, error) {
	switch provider {
	case "yandex":
		return CloudProviderYandex, nil
	case "aws":
		return CloudProviderAWS, nil
	default:
		return "", fmt.Errorf("unsupported cloud provider: %s", provider)
	}
}
