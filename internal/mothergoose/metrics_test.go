package mothergoose

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendHeartbeat(t *testing.T) {
	var received HeartbeatPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/runners/runner-abc/heartbeat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	payload := HeartbeatPayload{EggName: "my-egg", State: "active"}

	if err := client.SendHeartbeat(context.Background(), "runner-abc", payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.EggName != "my-egg" {
		t.Errorf("EggName: got %q, want %q", received.EggName, "my-egg")
	}
	if received.State != "active" {
		t.Errorf("State: got %q, want %q", received.State, "active")
	}
}

func TestSendHeartbeat_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Zero retries so the test is fast.
	client := NewClient(server.URL, "key", WithMaxRetries(0))
	err := client.SendHeartbeat(context.Background(), "r", HeartbeatPayload{})
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

func TestReportRunnerMetrics(t *testing.T) {
	var received RunnerMetricsPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/runners/runner-xyz/metrics" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	payload := RunnerMetricsPayload{
		EggName:          "my-egg",
		State:            "active",
		JobCount:         5,
		CPUUsage:         33.3,
		MemoryUsage:      55.5,
		DiskUsage:        20.0,
		AgentVersion:     "16.0.0",
		FailureCount:     1,
		LastJobTimestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	if err := client.ReportRunnerMetrics(context.Background(), "runner-xyz", payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.EggName != "my-egg" {
		t.Errorf("EggName: got %q, want %q", received.EggName, "my-egg")
	}
	if received.CPUUsage != 33.3 {
		t.Errorf("CPUUsage: got %.1f, want 33.3", received.CPUUsage)
	}
	if received.MemoryUsage != 55.5 {
		t.Errorf("MemoryUsage: got %.1f, want 55.5", received.MemoryUsage)
	}
	if received.DiskUsage != 20.0 {
		t.Errorf("DiskUsage: got %.1f, want 20.0", received.DiskUsage)
	}
	if received.JobCount != 5 {
		t.Errorf("JobCount: got %d, want 5", received.JobCount)
	}
	if received.AgentVersion != "16.0.0" {
		t.Errorf("AgentVersion: got %q, want %q", received.AgentVersion, "16.0.0")
	}
	if received.FailureCount != 1 {
		t.Errorf("FailureCount: got %d, want 1", received.FailureCount)
	}
}

func TestReportRunnerMetrics_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"error":"bad payload"}`)); err != nil {
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "key", WithMaxRetries(0))
	err := client.ReportRunnerMetrics(context.Background(), "r", RunnerMetricsPayload{})
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
}

func TestReportRunnerMetrics_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.ReportRunnerMetrics(ctx, "r", RunnerMetricsPayload{})
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

// TestClientImplementsInterface is a compile-time check that Client still
// satisfies MotherGooseClient after adding the new methods.
func TestClientImplementsInterface(t *testing.T) {
	var _ MotherGooseClient = (*Client)(nil)
}
