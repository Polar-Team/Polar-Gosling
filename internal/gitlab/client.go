package gitlab

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Client wraps the GitLab Go SDK
type Client struct {
	client *gitlab.Client
}

// NewClient creates a new GitLab client
func NewClient(baseURL, token string) (*Client, error) {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

// RunnerConfig represents GitLab runner configuration
type RunnerConfig struct {
	ProjectID   int
	Token       string
	Description string
	Tags        []string
	RunUntagged bool
	Locked      bool
}

// Runner represents a registered GitLab runner
type Runner struct {
	ID          int
	Token       string
	Description string
	Active      bool
	Tags        []string
}

// WebhookConfig represents GitLab webhook configuration
type WebhookConfig struct {
	URL                   string
	Token                 string
	PushEvents            bool
	MergeRequestsEvents   bool
	PipelineEvents        bool
	EnableSSLVerification bool
}

// RegisterRunner registers a new runner with GitLab
func (c *Client) RegisterRunner(ctx context.Context, config *RunnerConfig) (*Runner, error) {
	// Register the runner using the GitLab API
	options := &gitlab.RegisterNewRunnerOptions{
		Token:       gitlab.Ptr(config.Token),
		Description: gitlab.Ptr(config.Description),
		TagList:     &config.Tags,
		RunUntagged: gitlab.Ptr(config.RunUntagged),
		Locked:      gitlab.Ptr(config.Locked),
	}

	runner, _, err := c.client.Runners.RegisterNewRunner(options)
	if err != nil {
		return nil, fmt.Errorf("failed to register runner: %w", err)
	}

	// Extract tags - the API may return tags in different formats
	tags := config.Tags
	if len(tags) == 0 {
		tags = []string{}
	}

	return &Runner{
		ID:          int(runner.ID),
		Token:       runner.Token,
		Description: runner.Description,
		Active:      !runner.Paused,
		Tags:        tags,
	}, nil
}

// UnregisterRunner removes a runner from GitLab
func (c *Client) UnregisterRunner(ctx context.Context, runnerID int) error {
	options := &gitlab.DeleteRegisteredRunnerOptions{
		Token: gitlab.Ptr(""), // Token will be provided if needed
	}
	_, err := c.client.Runners.DeleteRegisteredRunner(options)
	if err != nil {
		return fmt.Errorf("failed to unregister runner: %w", err)
	}
	return nil
}

// GetRunner retrieves runner details from GitLab
func (c *Client) GetRunner(ctx context.Context, runnerID int) (*Runner, error) {
	runner, _, err := c.client.Runners.GetRunnerDetails(runnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get runner details: %w", err)
	}

	// Extract tags - may not be available in all API versions
	tags := []string{}

	return &Runner{
		ID:          int(runner.ID),
		Token:       runner.Token,
		Description: runner.Description,
		Active:      !runner.Paused,
		Tags:        tags,
	}, nil
}

// CreateWebhook creates a webhook for a GitLab project
func (c *Client) CreateWebhook(ctx context.Context, projectID int, config *WebhookConfig) (int, error) {
	options := &gitlab.AddProjectHookOptions{
		URL:                   gitlab.Ptr(config.URL),
		Token:                 gitlab.Ptr(config.Token),
		PushEvents:            gitlab.Ptr(config.PushEvents),
		MergeRequestsEvents:   gitlab.Ptr(config.MergeRequestsEvents),
		PipelineEvents:        gitlab.Ptr(config.PipelineEvents),
		EnableSSLVerification: gitlab.Ptr(config.EnableSSLVerification),
	}

	hook, _, err := c.client.Projects.AddProjectHook(projectID, options)
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook: %w", err)
	}

	return int(hook.ID), nil
}

// DeleteWebhook removes a webhook from a GitLab project
func (c *Client) DeleteWebhook(ctx context.Context, projectID, hookID int) error {
	_, err := c.client.Projects.DeleteProjectHook(projectID, int64(hookID))
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	return nil
}

// GetWebhook retrieves webhook details from GitLab
func (c *Client) GetWebhook(ctx context.Context, projectID, hookID int) (*gitlab.ProjectHook, error) {
	hook, _, err := c.client.Projects.GetProjectHook(projectID, int64(hookID))
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return hook, nil
}

// ListProjectRunners lists all runners for a project
func (c *Client) ListProjectRunners(ctx context.Context, projectID int) ([]*Runner, error) {
	options := &gitlab.ListProjectRunnersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	runners, _, err := c.client.Runners.ListProjectRunners(projectID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list project runners: %w", err)
	}

	result := make([]*Runner, len(runners))
	for i, r := range runners {
		// Extract tags - may not be available in all API versions
		tags := []string{}

		result[i] = &Runner{
			ID:          int(r.ID),
			Description: r.Description,
			Active:      !r.Paused,
			Tags:        tags,
		}
	}

	return result, nil
}

// UpdateRunner updates runner configuration
func (c *Client) UpdateRunner(ctx context.Context, runnerID int, description string, tags []string) error {
	options := &gitlab.UpdateRunnerDetailsOptions{
		Description: gitlab.Ptr(description),
		TagList:     &tags,
	}

	_, _, err := c.client.Runners.UpdateRunnerDetails(runnerID, options)
	if err != nil {
		return fmt.Errorf("failed to update runner: %w", err)
	}

	return nil
}

// VerifyRunner checks if a runner is still registered and active
func (c *Client) VerifyRunner(ctx context.Context, runnerID int) (bool, error) {
	runner, _, err := c.client.Runners.GetRunnerDetails(runnerID)
	if err != nil {
		return false, fmt.Errorf("failed to verify runner: %w", err)
	}

	return !runner.Paused, nil
}
