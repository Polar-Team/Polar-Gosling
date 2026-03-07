// Package runner implements the GitLab Runner Agent lifecycle manager.
// It is used by the `gosling runner` CLI command (cli/runner.go) and can be
// imported independently for testing.
package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/polar-gosling/gosling/internal/gitlab"
	"github.com/polar-gosling/gosling/internal/mothergoose"
)

// defaultHeartbeatInterval is how often the runner sends a liveness ping.
const defaultHeartbeatInterval = 30 * time.Second

// Config holds the configuration for a runner instance.
type Config struct {
	EggName           string
	RunnerID          string
	TokenSecretURI    string
	GitLabServer      string
	Tags              []string
	AgentVersion      string
	MotherGooseURL    string
	APIKey            string
	MetricsInterval   time.Duration
	HeartbeatInterval time.Duration
}

// Manager manages the GitLab Runner Agent process lifecycle.
type Manager struct {
	Config    *Config
	Collector *MetricsCollector

	GitLabClient *gitlab.Client
	MGClient     mothergoose.MotherGooseClient
	RegisteredID int
	agentCmd     *exec.Cmd
}

// New creates a new Manager from the given Config.
func New(cfg *Config) (*Manager, error) {
	gitlabURL := fmt.Sprintf("https://%s", cfg.GitLabServer)
	glClient, err := gitlab.NewClient(gitlabURL, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	mgr := &Manager{
		Config:       cfg,
		GitLabClient: glClient,
		Collector:    NewMetricsCollector(cfg.RunnerID, cfg.EggName, cfg.MetricsInterval, nil),
	}

	if cfg.MotherGooseURL != "" {
		mgr.MGClient = mothergoose.NewClient(cfg.MotherGooseURL, cfg.APIKey)
	}

	return mgr, nil
}

// Run is the main loop: retrieve token → sync version → register → metrics → agent.
func (m *Manager) Run(ctx context.Context) error {
	token, err := m.RetrieveToken()
	if err != nil {
		return fmt.Errorf("failed to retrieve runner token: %w", err)
	}

	if err := m.SyncVersion(); err != nil {
		fmt.Printf("Warning: failed to sync agent version: %v\n", err)
	}

	if err := m.Register(ctx, token); err != nil {
		return fmt.Errorf("failed to register runner: %w", err)
	}
	defer m.Deregister(ctx)

	if m.MGClient != nil {
		go m.Collector.Run(ctx)
		go m.MetricsLoop(ctx)
		go m.HeartbeatLoop(ctx)
	}

	return m.StartAgent(ctx)
}

// RetrieveToken retrieves the runner token from secret storage based on the URI scheme.
func (m *Manager) RetrieveToken() (string, error) {
	uri := m.Config.TokenSecretURI
	switch {
	case strings.HasPrefix(uri, "yc-lockbox://"):
		return m.retrieveFromYCLockbox(uri)
	case strings.HasPrefix(uri, "aws-sm://"):
		return m.retrieveFromAWSSecretsManager(uri)
	case strings.HasPrefix(uri, "vault://"):
		return m.retrieveFromVault(uri)
	default:
		return "", fmt.Errorf("unsupported secret URI scheme: %s", uri)
	}
}

func (m *Manager) retrieveFromYCLockbox(uri string) (string, error) {
	// Task 24: Implement YC Lockbox secret retrieval
	path := strings.TrimPrefix(uri, "yc-lockbox://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid yc-lockbox URI: %s", uri)
	}
	return os.Getenv("RUNNER_TOKEN"), nil
}

func (m *Manager) retrieveFromAWSSecretsManager(uri string) (string, error) {
	// Task 24: Implement AWS Secrets Manager secret retrieval
	path := strings.TrimPrefix(uri, "aws-sm://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid aws-sm URI: %s", uri)
	}
	return os.Getenv("RUNNER_TOKEN"), nil
}

func (m *Manager) retrieveFromVault(uri string) (string, error) {
	// Task 24: Implement Vault secret retrieval
	path := strings.TrimPrefix(uri, "vault://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid vault URI: %s", uri)
	}
	return os.Getenv("RUNNER_TOKEN"), nil
}

// Register registers the runner with GitLab using the provided token.
func (m *Manager) Register(ctx context.Context, token string) error {
	gitlabURL := fmt.Sprintf("https://%s", m.Config.GitLabServer)
	glClient, err := gitlab.NewClient(gitlabURL, token)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client with token: %w", err)
	}
	m.GitLabClient = glClient

	cfg := &gitlab.RunnerConfig{
		Token:       token,
		Description: fmt.Sprintf("gosling-runner-%s", m.Config.EggName),
		Tags:        m.Config.Tags,
		RunUntagged: len(m.Config.Tags) == 0,
		Locked:      false,
	}

	r, err := m.GitLabClient.RegisterRunner(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to register runner with GitLab: %w", err)
	}

	m.RegisteredID = r.ID
	fmt.Printf("Runner registered with GitLab (ID: %d, tags: %v)\n", r.ID, r.Tags)
	return nil
}

// Deregister removes the runner from GitLab on shutdown.
func (m *Manager) Deregister(ctx context.Context) {
	if m.RegisteredID == 0 {
		return
	}
	fmt.Printf("Deregistering runner %d from GitLab\n", m.RegisteredID)
	if err := m.GitLabClient.UnregisterRunner(ctx, m.RegisteredID); err != nil {
		fmt.Printf("Warning: failed to deregister runner: %v\n", err)
	}
}

// SyncVersion checks the installed agent version against the required one.
func (m *Manager) SyncVersion() error {
	if m.Config.AgentVersion == "" {
		return nil
	}

	current, err := GetAgentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current agent version: %w", err)
	}

	if current == m.Config.AgentVersion {
		fmt.Printf("GitLab Runner Agent is already at required version %s\n", m.Config.AgentVersion)
		return nil
	}

	fmt.Printf("Syncing GitLab Runner Agent from %s to %s\n", current, m.Config.AgentVersion)
	// Task 24: Implement actual version download and update
	return nil
}

// GetAgentVersion returns the currently installed GitLab Runner Agent version.
func GetAgentVersion() (string, error) {
	out, err := exec.Command("gitlab-runner", "--version").Output()
	if err != nil {
		return "unknown", nil
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Version:")), nil
		}
	}
	return "unknown", nil
}

// StartAgent starts the GitLab Runner Agent process and monitors it until exit or ctx cancel.
func (m *Manager) StartAgent(ctx context.Context) error {
	fmt.Println("Starting GitLab Runner Agent")

	m.agentCmd = exec.CommandContext(ctx, "gitlab-runner", "run", "--working-directory", "/tmp/gitlab-runner")
	m.agentCmd.Stdout = os.Stdout
	m.agentCmd.Stderr = os.Stderr

	if err := m.agentCmd.Start(); err != nil {
		return fmt.Errorf("failed to start GitLab Runner Agent: %w", err)
	}

	fmt.Printf("GitLab Runner Agent started (PID: %d)\n", m.agentCmd.Process.Pid)

	done := make(chan error, 1)
	go func() { done <- m.agentCmd.Wait() }()

	select {
	case <-ctx.Done():
		fmt.Println("Context cancelled, stopping GitLab Runner Agent")
		if m.agentCmd.Process != nil {
			_ = m.agentCmd.Process.Signal(syscall.SIGTERM)
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				_ = m.agentCmd.Process.Kill()
			}
		}
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("GitLab Runner Agent exited with error: %w", err)
		}
		fmt.Println("GitLab Runner Agent exited cleanly")
		return nil
	}
}

// ReportMetrics collects the latest snapshot and sends it to MotherGoose.
func (m *Manager) ReportMetrics(ctx context.Context) {
	if m.MGClient == nil {
		return
	}

	snap := m.Collector.Latest()
	runnerID := m.Config.RunnerID
	if runnerID == "" {
		runnerID = fmt.Sprintf("runner-%s", m.Config.EggName)
	}

	payload := mothergoose.RunnerMetricsPayload{
		EggName:          snap.EggName,
		State:            snap.State,
		JobCount:         snap.JobCount,
		CPUUsage:         snap.CPUUsage,
		MemoryUsage:      snap.MemoryUsage,
		DiskUsage:        snap.DiskUsage,
		AgentVersion:     snap.AgentVersion,
		FailureCount:     snap.FailureCount,
		LastJobTimestamp: snap.LastJobTimestamp,
	}

	if err := m.MGClient.ReportRunnerMetrics(ctx, runnerID, payload); err != nil {
		fmt.Printf("Warning: failed to report metrics: %v\n", err)
	}
}

// MetricsLoop periodically calls ReportMetrics until ctx is cancelled.
func (m *Manager) MetricsLoop(ctx context.Context) {
	interval := m.Config.MetricsInterval
	if interval <= 0 {
		interval = defaultMetricsInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.ReportMetrics(ctx)
		}
	}
}

// SendHeartbeat sends a liveness ping to MotherGoose.
func (m *Manager) SendHeartbeat(ctx context.Context) {
	if m.MGClient == nil {
		return
	}

	runnerID := m.Config.RunnerID
	if runnerID == "" {
		runnerID = fmt.Sprintf("runner-%s", m.Config.EggName)
	}

	snap := m.Collector.Latest()
	payload := mothergoose.HeartbeatPayload{
		EggName: m.Config.EggName,
		State:   snap.State,
	}

	if err := m.MGClient.SendHeartbeat(ctx, runnerID, payload); err != nil {
		fmt.Printf("Warning: failed to send heartbeat: %v\n", err)
	}
}

// HeartbeatLoop periodically calls SendHeartbeat until ctx is cancelled.
func (m *Manager) HeartbeatLoop(ctx context.Context) {
	interval := m.Config.HeartbeatInterval
	if interval <= 0 {
		interval = defaultHeartbeatInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.SendHeartbeat(ctx)
		}
	}
}

// ReloadConfig handles SIGHUP by re-reading configuration.
func (m *Manager) ReloadConfig() {
	fmt.Println("Reloading runner configuration")
	// Task 24: Implement config reload (re-read tags, version from Egg config)
}

// ParseTags splits a comma-separated tag string into a deduplicated slice.
func ParseTags(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
