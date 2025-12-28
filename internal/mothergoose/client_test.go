package mothergoose

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/polar-gosling/gosling/internal/deployer"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com", "test-api-key")

	if client.baseURL != "https://api.example.com" {
		t.Errorf("expected baseURL to be 'https://api.example.com', got '%s'", client.baseURL)
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("expected apiKey to be 'test-api-key', got '%s'", client.apiKey)
	}

	if client.maxRetries != 3 {
		t.Errorf("expected maxRetries to be 3, got %d", client.maxRetries)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected timeout to be 30s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClientWithOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 10 * time.Second}
	client := NewClient(
		"https://api.example.com",
		"test-api-key",
		WithTimeout(60*time.Second),
		WithMaxRetries(5),
		WithHTTPClient(customClient),
	)

	if client.maxRetries != 5 {
		t.Errorf("expected maxRetries to be 5, got %d", client.maxRetries)
	}

	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestGetEggStatus(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/eggs/test-egg/status" {
			t.Errorf("expected path '/eggs/test-egg/status', got '%s'", r.URL.Path)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
		}

		// Send response
		status := EggStatus{
			EggName:    "test-egg",
			ConfigHash: "abc123",
			ActiveRunners: []*Runner{
				{
					ID:      "runner-1",
					EggName: "test-egg",
					Type:    "vm",
					State:   "active",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	ctx := context.Background()

	status, err := client.GetEggStatus(ctx, "test-egg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.EggName != "test-egg" {
		t.Errorf("expected EggName 'test-egg', got '%s'", status.EggName)
	}

	if status.ConfigHash != "abc123" {
		t.Errorf("expected ConfigHash 'abc123', got '%s'", status.ConfigHash)
	}

	if len(status.ActiveRunners) != 1 {
		t.Errorf("expected 1 active runner, got %d", len(status.ActiveRunners))
	}
}

func TestListEggs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/eggs" {
			t.Errorf("expected path '/eggs', got '%s'", r.URL.Path)
		}

		eggs := []*deployer.EggConfig{
			{
				Name: "egg-1",
				Type: deployer.RunnerTypeVM,
			},
			{
				Name: "egg-2",
				Type: deployer.RunnerTypeServerless,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(eggs)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	ctx := context.Background()

	eggs, err := client.ListEggs(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(eggs) != 2 {
		t.Errorf("expected 2 eggs, got %d", len(eggs))
	}

	if eggs[0].Name != "egg-1" {
		t.Errorf("expected first egg name 'egg-1', got '%s'", eggs[0].Name)
	}
}

func TestCreateOrUpdateEgg(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/eggs" {
			t.Errorf("expected path '/eggs', got '%s'", r.URL.Path)
		}

		// Verify request body
		var config deployer.EggConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if config.Name != "test-egg" {
			t.Errorf("expected egg name 'test-egg', got '%s'", config.Name)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	ctx := context.Background()

	config := &deployer.EggConfig{
		Name: "test-egg",
		Type: deployer.RunnerTypeVM,
	}

	err := client.CreateOrUpdateEgg(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDeploymentPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/eggs/test-egg/plans/plan-123" {
			t.Errorf("expected path '/eggs/test-egg/plans/plan-123', got '%s'", r.URL.Path)
		}

		plan := deployer.DeploymentPlan{
			ID:         "plan-123",
			EggName:    "test-egg",
			PlanType:   "runner",
			ConfigHash: "abc123",
			Status:     "applied",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(plan)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	ctx := context.Background()

	plan, err := client.GetDeploymentPlan(ctx, "test-egg", "plan-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.ID != "plan-123" {
		t.Errorf("expected plan ID 'plan-123', got '%s'", plan.ID)
	}

	if plan.EggName != "test-egg" {
		t.Errorf("expected egg name 'test-egg', got '%s'", plan.EggName)
	}
}

func TestListDeploymentPlans(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/eggs/test-egg/plans" {
			t.Errorf("expected path '/eggs/test-egg/plans', got '%s'", r.URL.Path)
		}

		plans := []*deployer.DeploymentPlan{
			{
				ID:      "plan-1",
				EggName: "test-egg",
				Status:  "applied",
			},
			{
				ID:      "plan-2",
				EggName: "test-egg",
				Status:  "pending",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(plans)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	ctx := context.Background()

	plans, err := client.ListDeploymentPlans(ctx, "test-egg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}

	if plans[0].ID != "plan-1" {
		t.Errorf("expected first plan ID 'plan-1', got '%s'", plans[0].ID)
	}
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "egg not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	ctx := context.Background()

	_, err := client.GetEggStatus(ctx, "nonexistent-egg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The error is wrapped, so we need to check if it contains an HTTPError
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError in error chain, got %T: %v", err, err)
	}

	if httpErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status code 404, got %d", httpErr.StatusCode)
	}
}

func TestRetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail the first 2 attempts with a retryable error
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Succeed on the 3rd attempt
		status := EggStatus{
			EggName: "test-egg",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", WithMaxRetries(3))
	ctx := context.Background()

	status, err := client.GetEggStatus(ctx, "test-egg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.EggName != "test-egg" {
		t.Errorf("expected EggName 'test-egg', got '%s'", status.EggName)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestNoRetryOn4xxErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key", WithMaxRetries(3))
	ctx := context.Background()

	_, err := client.GetEggStatus(ctx, "test-egg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should only attempt once for 4xx errors (except 429)
	if attempts != 1 {
		t.Errorf("expected 1 attempt for 4xx error, got %d", attempts)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GetEggStatus(ctx, "test-egg")
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}
