package runner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// stubStatReader returns fixed values for deterministic tests.
type stubStatReader struct {
	stats SystemStats
	err   error
	calls atomic.Int32
}

func (s *stubStatReader) ReadStats() (SystemStats, error) {
	s.calls.Add(1)
	return s.stats, s.err
}

// TestMetricsCollector_Latest_PopulatedAfterRun verifies that Latest() returns
// a non-zero snapshot after the collector has run at least once.
func TestMetricsCollector_Latest_PopulatedAfterRun(t *testing.T) {
	reader := &stubStatReader{
		stats: SystemStats{
			CPUUsagePercent:    42.5,
			MemoryUsagePercent: 60.0,
			DiskUsagePercent:   75.0,
		},
	}

	c := NewMetricsCollector("runner-1", "my-egg", 100*time.Millisecond, reader)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	go c.Run(ctx)
	<-ctx.Done()

	snap := c.Latest()
	if snap.CPUUsage != 42.5 {
		t.Errorf("CPUUsage: got %.1f, want 42.5", snap.CPUUsage)
	}
	if snap.MemoryUsage != 60.0 {
		t.Errorf("MemoryUsage: got %.1f, want 60.0", snap.MemoryUsage)
	}
	if snap.DiskUsage != 75.0 {
		t.Errorf("DiskUsage: got %.1f, want 75.0", snap.DiskUsage)
	}
	if snap.RunnerID != "runner-1" {
		t.Errorf("RunnerID: got %q, want %q", snap.RunnerID, "runner-1")
	}
	if snap.EggName != "my-egg" {
		t.Errorf("EggName: got %q, want %q", snap.EggName, "my-egg")
	}
}

// TestMetricsCollector_CollectsImmediately verifies the first collection happens
// before the first tick (i.e., Latest() is non-zero right after Run starts).
func TestMetricsCollector_CollectsImmediately(t *testing.T) {
	reader := &stubStatReader{
		stats: SystemStats{CPUUsagePercent: 10.0},
	}

	c := NewMetricsCollector("r", "egg", time.Hour, reader) // long interval — only immediate collect fires

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		c.Run(ctx)
		close(done)
	}()

	// Give the goroutine a moment to execute the immediate collect.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	snap := c.Latest()
	if snap.CPUUsage != 10.0 {
		t.Errorf("expected immediate collect: CPUUsage=10.0, got %.1f", snap.CPUUsage)
	}
	if reader.calls.Load() < 1 {
		t.Error("expected at least one ReadStats call")
	}
}

// TestMetricsCollector_UpdateJobCount verifies the job count is updated correctly.
func TestMetricsCollector_UpdateJobCount(t *testing.T) {
	c := NewMetricsCollector("r", "egg", time.Hour, &stubStatReader{})
	c.UpdateJobCount(7)
	if got := c.Latest().JobCount; got != 7 {
		t.Errorf("JobCount: got %d, want 7", got)
	}
}

// TestMetricsCollector_IncrementFailures verifies failure counting is cumulative.
func TestMetricsCollector_IncrementFailures(t *testing.T) {
	c := NewMetricsCollector("r", "egg", time.Hour, &stubStatReader{})
	c.IncrementFailures()
	c.IncrementFailures()
	c.IncrementFailures()
	if got := c.Latest().FailureCount; got != 3 {
		t.Errorf("FailureCount: got %d, want 3", got)
	}
}

// TestMetricsCollector_SetState verifies the state string is updated.
func TestMetricsCollector_SetState(t *testing.T) {
	c := NewMetricsCollector("r", "egg", time.Hour, &stubStatReader{})
	c.SetState("idle")
	if got := c.Latest().State; got != "idle" {
		t.Errorf("State: got %q, want %q", got, "idle")
	}
}

// TestMetricsCollector_JobCountPreservedAcrossCollects verifies that job count
// set via UpdateJobCount is not reset when collect() runs again.
func TestMetricsCollector_JobCountPreservedAcrossCollects(t *testing.T) {
	reader := &stubStatReader{stats: SystemStats{CPUUsagePercent: 5.0}}
	c := NewMetricsCollector("r", "egg", 50*time.Millisecond, reader)

	c.UpdateJobCount(42)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go c.Run(ctx)
	<-ctx.Done()

	if got := c.Latest().JobCount; got != 42 {
		t.Errorf("JobCount after re-collect: got %d, want 42", got)
	}
}

// TestMetricsCollector_DefaultInterval verifies that a zero interval falls back
// to the default (no panic, collector is usable).
func TestMetricsCollector_DefaultInterval(t *testing.T) {
	c := NewMetricsCollector("r", "egg", 0, &stubStatReader{})
	if c.interval != defaultMetricsInterval {
		t.Errorf("interval: got %v, want %v", c.interval, defaultMetricsInterval)
	}
}

// TestMetricsCollector_NilReaderFallsBackToOS verifies that passing nil reader
// uses the real OS reader (no panic).
func TestMetricsCollector_NilReaderFallsBackToOS(t *testing.T) {
	c := NewMetricsCollector("r", "egg", time.Hour, nil)
	if c.reader == nil {
		t.Error("expected non-nil reader when nil is passed")
	}
	if _, ok := c.reader.(*OSStatReader); !ok {
		t.Errorf("expected *OSStatReader, got %T", c.reader)
	}
}

// TestMetricsCollector_HeartbeatTimestampUpdated verifies LastHeartbeat is set
// to a recent time after collection.
func TestMetricsCollector_HeartbeatTimestampUpdated(t *testing.T) {
	before := time.Now().UTC()
	c := NewMetricsCollector("r", "egg", time.Hour, &stubStatReader{})
	c.collect()
	after := time.Now().UTC()

	snap := c.Latest()
	if snap.LastHeartbeat.Before(before) || snap.LastHeartbeat.After(after) {
		t.Errorf("LastHeartbeat %v not in [%v, %v]", snap.LastHeartbeat, before, after)
	}
}

// TestMetricsCollector_ConcurrentSafety exercises concurrent reads and writes
// to verify no data races (run with -race).
func TestMetricsCollector_ConcurrentSafety(t *testing.T) {
	reader := &stubStatReader{stats: SystemStats{CPUUsagePercent: 1.0}}
	c := NewMetricsCollector("r", "egg", 10*time.Millisecond, reader)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go c.Run(ctx)

	// Concurrent writers
	go func() {
		for i := 0; i < 50; i++ {
			c.UpdateJobCount(i)
			time.Sleep(time.Millisecond)
		}
	}()
	go func() {
		for i := 0; i < 50; i++ {
			c.IncrementFailures()
			time.Sleep(time.Millisecond)
		}
	}()

	// Concurrent readers
	for i := 0; i < 50; i++ {
		_ = c.Latest()
		time.Sleep(time.Millisecond)
	}

	<-ctx.Done()
}
