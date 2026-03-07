package runner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/polar-gosling/gosling/internal/deployer"
	"github.com/polar-gosling/gosling/internal/mothergoose"
)

// mockMGClient records calls to SendHeartbeat and ReportRunnerMetrics.
type mockMGClient struct {
	heartbeatCalls atomic.Int32
	metricsCalls   atomic.Int32

	lastHeartbeatRunnerID string
	lastHeartbeatPayload  mothergoose.HeartbeatPayload

	lastMetricsRunnerID string
	lastMetricsPayload  mothergoose.RunnerMetricsPayload

	heartbeatErr error
	metricsErr   error
}

func (m *mockMGClient) GetEggStatus(_ context.Context, _ string) (*mothergoose.EggStatus, error) {
	return &mothergoose.EggStatus{}, nil
}
func (m *mockMGClient) ListEggs(_ context.Context) ([]*deployer.EggConfig, error) {
	return nil, nil
}
func (m *mockMGClient) CreateOrUpdateEgg(_ context.Context, _ *deployer.EggConfig) error {
	return nil
}
func (m *mockMGClient) GetDeploymentPlan(_ context.Context, _, _ string) (*deployer.DeploymentPlan, error) {
	return nil, nil
}
func (m *mockMGClient) ListDeploymentPlans(_ context.Context, _ string) ([]*deployer.DeploymentPlan, error) {
	return nil, nil
}
func (m *mockMGClient) SendHeartbeat(_ context.Context, runnerID string, payload mothergoose.HeartbeatPayload) error {
	m.heartbeatCalls.Add(1)
	m.lastHeartbeatRunnerID = runnerID
	m.lastHeartbeatPayload = payload
	return m.heartbeatErr
}
func (m *mockMGClient) ReportRunnerMetrics(_ context.Context, runnerID string, payload mothergoose.RunnerMetricsPayload) error {
	m.metricsCalls.Add(1)
	m.lastMetricsRunnerID = runnerID
	m.lastMetricsPayload = payload
	return m.metricsErr
}

// newTestManager builds a Manager wired with a mock client and stub stat reader.
func newTestManager(eggName, runnerID string, mock *mockMGClient) *Manager {
	reader := &stubStatReader{
		stats: SystemStats{
			CPUUsagePercent:    25.0,
			MemoryUsagePercent: 50.0,
			DiskUsagePercent:   10.0,
		},
	}
	cfg := &Config{
		EggName:           eggName,
		RunnerID:          runnerID,
		MetricsInterval:   time.Hour, // disabled in unit tests
		HeartbeatInterval: time.Hour, // disabled in unit tests
	}
	mgr := &Manager{
		Config:    cfg,
		MGClient:  mock,
		Collector: NewMetricsCollector(runnerID, eggName, time.Hour, reader),
	}
	// Populate the collector snapshot once.
	mgr.Collector.collect()
	return mgr
}

// TestReportMetrics_CallsAPIWithCorrectPayload verifies that ReportMetrics
// sends the collector snapshot to the MotherGoose API.
func TestReportMetrics_CallsAPIWithCorrectPayload(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("my-egg", "runner-1", mock)

	mgr.ReportMetrics(context.Background())

	if mock.metricsCalls.Load() != 1 {
		t.Fatalf("expected 1 ReportRunnerMetrics call, got %d", mock.metricsCalls.Load())
	}
	if mock.lastMetricsRunnerID != "runner-1" {
		t.Errorf("runnerID: got %q, want %q", mock.lastMetricsRunnerID, "runner-1")
	}
	p := mock.lastMetricsPayload
	if p.EggName != "my-egg" {
		t.Errorf("EggName: got %q, want %q", p.EggName, "my-egg")
	}
	if p.CPUUsage != 25.0 {
		t.Errorf("CPUUsage: got %.1f, want 25.0", p.CPUUsage)
	}
	if p.MemoryUsage != 50.0 {
		t.Errorf("MemoryUsage: got %.1f, want 50.0", p.MemoryUsage)
	}
	if p.DiskUsage != 10.0 {
		t.Errorf("DiskUsage: got %.1f, want 10.0", p.DiskUsage)
	}
}

// TestReportMetrics_FallbackRunnerID verifies that when RunnerID is empty,
// the manager derives an ID from the egg name.
func TestReportMetrics_FallbackRunnerID(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("my-egg", "", mock)

	mgr.ReportMetrics(context.Background())

	if mock.lastMetricsRunnerID != "runner-my-egg" {
		t.Errorf("fallback runnerID: got %q, want %q", mock.lastMetricsRunnerID, "runner-my-egg")
	}
}

// TestReportMetrics_NoClientIsNoop verifies that ReportMetrics is a no-op
// when no MotherGoose client is configured.
func TestReportMetrics_NoClientIsNoop(t *testing.T) {
	mgr := &Manager{
		Config:    &Config{EggName: "egg"},
		Collector: NewMetricsCollector("", "egg", time.Hour, &stubStatReader{}),
	}
	// Should not panic.
	mgr.ReportMetrics(context.Background())
}

// TestSendHeartbeat_CallsAPIWithCorrectPayload verifies that SendHeartbeat
// sends the current state to the MotherGoose API.
func TestSendHeartbeat_CallsAPIWithCorrectPayload(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("my-egg", "runner-1", mock)
	mgr.Collector.SetState("idle")

	mgr.SendHeartbeat(context.Background())

	if mock.heartbeatCalls.Load() != 1 {
		t.Fatalf("expected 1 SendHeartbeat call, got %d", mock.heartbeatCalls.Load())
	}
	if mock.lastHeartbeatRunnerID != "runner-1" {
		t.Errorf("runnerID: got %q, want %q", mock.lastHeartbeatRunnerID, "runner-1")
	}
	p := mock.lastHeartbeatPayload
	if p.EggName != "my-egg" {
		t.Errorf("EggName: got %q, want %q", p.EggName, "my-egg")
	}
	if p.State != "idle" {
		t.Errorf("State: got %q, want %q", p.State, "idle")
	}
}

// TestSendHeartbeat_NoClientIsNoop verifies that SendHeartbeat is a no-op
// when no MotherGoose client is configured.
func TestSendHeartbeat_NoClientIsNoop(t *testing.T) {
	mgr := &Manager{
		Config:    &Config{EggName: "egg"},
		Collector: NewMetricsCollector("", "egg", time.Hour, &stubStatReader{}),
	}
	mgr.SendHeartbeat(context.Background())
}

// TestMetricsLoop_FiresMultipleTimes verifies that MetricsLoop calls
// ReportRunnerMetrics repeatedly at the configured interval.
func TestMetricsLoop_FiresMultipleTimes(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("egg", "r", mock)
	mgr.Config.MetricsInterval = 30 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go mgr.MetricsLoop(ctx)
	<-ctx.Done()

	calls := mock.metricsCalls.Load()
	if calls < 2 {
		t.Errorf("expected at least 2 metrics calls in 200ms with 30ms interval, got %d", calls)
	}
}

// TestHeartbeatLoop_FiresMultipleTimes verifies that HeartbeatLoop calls
// SendHeartbeat repeatedly at the configured interval.
func TestHeartbeatLoop_FiresMultipleTimes(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("egg", "r", mock)
	mgr.Config.HeartbeatInterval = 30 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go mgr.HeartbeatLoop(ctx)
	<-ctx.Done()

	calls := mock.heartbeatCalls.Load()
	if calls < 2 {
		t.Errorf("expected at least 2 heartbeat calls in 200ms with 30ms interval, got %d", calls)
	}
}

// TestMetricsLoop_StopsOnContextCancel verifies that MetricsLoop exits cleanly
// when the context is cancelled.
func TestMetricsLoop_StopsOnContextCancel(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("egg", "r", mock)
	mgr.Config.MetricsInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		mgr.MetricsLoop(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// good
	case <-time.After(500 * time.Millisecond):
		t.Error("MetricsLoop did not stop after context cancellation")
	}
}

// TestHeartbeatLoop_StopsOnContextCancel verifies that HeartbeatLoop exits
// cleanly when the context is cancelled.
func TestHeartbeatLoop_StopsOnContextCancel(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("egg", "r", mock)
	mgr.Config.HeartbeatInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		mgr.HeartbeatLoop(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// good
	case <-time.After(500 * time.Millisecond):
		t.Error("HeartbeatLoop did not stop after context cancellation")
	}
}

// TestMetricsLoop_DefaultIntervalUsedWhenZero verifies that a zero
// MetricsInterval in Config falls back to the package default.
func TestMetricsLoop_DefaultIntervalUsedWhenZero(t *testing.T) {
	mock := &mockMGClient{}
	mgr := newTestManager("egg", "r", mock)
	mgr.Config.MetricsInterval = 0

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// With the default 30s interval, no ticks fire in 50ms — that's fine,
	// we just verify the loop starts and stops without panic.
	go mgr.MetricsLoop(ctx)
	<-ctx.Done()
}
