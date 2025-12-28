package mothergoose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/polar-gosling/gosling/internal/deployer"
)

// Compile-time check to ensure Client implements MotherGooseClient interface
var _ MotherGooseClient = (*Client)(nil)

// Client implements the MotherGooseClient interface for communicating with MotherGoose API
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	maxRetries int
}

// ClientOption is a functional option for configuring the Client
type ClientOption func(*Client)

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new MotherGoose API client
func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	client := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: 3,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// EggStatus represents the deployment status of an Egg
type EggStatus struct {
	EggName           string                     `json:"egg_name"`
	LatestPlan        *deployer.DeploymentPlan   `json:"latest_plan"`
	DeploymentHistory []*deployer.DeploymentPlan `json:"deployment_history"`
	ActiveRunners     []*Runner                  `json:"active_runners"`
	ConfigHash        string                     `json:"config_hash"`
}

// Runner represents a runner instance
type Runner struct {
	ID            string    `json:"id"`
	EggName       string    `json:"egg_name"`
	Type          string    `json:"type"`
	State         string    `json:"state"`
	CloudProvider string    `json:"cloud_provider"`
	Region        string    `json:"region"`
	CreatedAt     time.Time `json:"created_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// GetEggStatus retrieves deployment status for an Egg
func (c *Client) GetEggStatus(ctx context.Context, eggName string) (*EggStatus, error) {
	url := fmt.Sprintf("%s/eggs/%s/status", c.baseURL, eggName)

	var status EggStatus
	err := c.doRequestWithRetry(ctx, "GET", url, nil, &status)
	if err != nil {
		return nil, fmt.Errorf("failed to get egg status: %w", err)
	}

	return &status, nil
}

// ListEggs lists all Egg configurations
func (c *Client) ListEggs(ctx context.Context) ([]*deployer.EggConfig, error) {
	url := fmt.Sprintf("%s/eggs", c.baseURL)

	var eggs []*deployer.EggConfig
	err := c.doRequestWithRetry(ctx, "GET", url, nil, &eggs)
	if err != nil {
		return nil, fmt.Errorf("failed to list eggs: %w", err)
	}

	return eggs, nil
}

// CreateOrUpdateEgg creates or updates an Egg configuration
func (c *Client) CreateOrUpdateEgg(ctx context.Context, config *deployer.EggConfig) error {
	url := fmt.Sprintf("%s/eggs", c.baseURL)

	err := c.doRequestWithRetry(ctx, "POST", url, config, nil)
	if err != nil {
		return fmt.Errorf("failed to create or update egg: %w", err)
	}

	return nil
}

// GetDeploymentPlan retrieves a specific deployment plan
func (c *Client) GetDeploymentPlan(ctx context.Context, eggName, planID string) (*deployer.DeploymentPlan, error) {
	url := fmt.Sprintf("%s/eggs/%s/plans/%s", c.baseURL, eggName, planID)

	var plan deployer.DeploymentPlan
	err := c.doRequestWithRetry(ctx, "GET", url, nil, &plan)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment plan: %w", err)
	}

	return &plan, nil
}

// ListDeploymentPlans lists all deployment plans for an Egg
func (c *Client) ListDeploymentPlans(ctx context.Context, eggName string) ([]*deployer.DeploymentPlan, error) {
	url := fmt.Sprintf("%s/eggs/%s/plans", c.baseURL, eggName)

	var plans []*deployer.DeploymentPlan
	err := c.doRequestWithRetry(ctx, "GET", url, nil, &plans)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployment plans: %w", err)
	}

	return plans, nil
}

// doRequestWithRetry performs an HTTP request with retry logic
func (c *Client) doRequestWithRetry(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, etc.
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.doRequest(ctx, method, url, body, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on client errors (4xx) except 429 (rate limit)
		if httpErr, ok := err.(*HTTPError); ok {
			if httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && httpErr.StatusCode != 429 {
				return err
			}
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}

// doRequest performs a single HTTP request
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	// Decode response if result is provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.Status, e.Body)
}
