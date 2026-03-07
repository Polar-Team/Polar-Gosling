// Package runner implements the GitLab Runner Agent lifecycle manager.
package runner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// defaultMetricsInterval is how often the collector samples system stats.
const defaultMetricsInterval = 30 * time.Second

// MetricsCollector periodically samples system resource usage and exposes
// the latest snapshot via Latest(). It is safe for concurrent use.
type MetricsCollector struct {
	runnerID string
	eggName  string
	interval time.Duration
	reader   StatReader

	mu       sync.RWMutex
	snapshot RunnerMetrics
}

// NewMetricsCollector creates a MetricsCollector with the given parameters.
// Pass a nil reader to use the real OS reader.
func NewMetricsCollector(runnerID, eggName string, interval time.Duration, reader StatReader) *MetricsCollector {
	if interval <= 0 {
		interval = defaultMetricsInterval
	}
	if reader == nil {
		reader = &OSStatReader{}
	}
	return &MetricsCollector{
		runnerID: runnerID,
		eggName:  eggName,
		interval: interval,
		reader:   reader,
	}
}

// Run starts the periodic collection loop. It blocks until ctx is cancelled.
func (c *MetricsCollector) Run(ctx context.Context) {
	// Collect once immediately so Latest() is populated before the first tick.
	c.collect()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// Latest returns the most recently collected metrics snapshot.
func (c *MetricsCollector) Latest() RunnerMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshot
}

// collect samples system stats and updates the internal snapshot.
func (c *MetricsCollector) collect() {
	stats, err := c.reader.ReadStats()
	if err != nil {
		fmt.Printf("Warning: failed to read system stats: %v\n", err)
	}

	agentVersion, _ := GetAgentVersion()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.snapshot = RunnerMetrics{
		RunnerID:      c.runnerID,
		EggName:       c.eggName,
		State:         "active",
		LastHeartbeat: time.Now().UTC(),
		CPUUsage:      stats.CPUUsagePercent,
		MemoryUsage:   stats.MemoryUsagePercent,
		DiskUsage:     stats.DiskUsagePercent,
		AgentVersion:  agentVersion,
		// JobCount and FailureCount are updated externally via UpdateJobCount / IncrementFailures.
		JobCount:     c.snapshot.JobCount,
		FailureCount: c.snapshot.FailureCount,
	}
}

// UpdateJobCount sets the current job execution count on the snapshot.
func (c *MetricsCollector) UpdateJobCount(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.JobCount = count
}

// IncrementFailures increments the failure counter on the snapshot.
func (c *MetricsCollector) IncrementFailures() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.FailureCount++
}

// SetState updates the runner state string on the snapshot.
func (c *MetricsCollector) SetState(state string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot.State = state
}
