package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/polar-gosling/gosling/internal/lockbox"
)

// mockSecretStore implements lockbox.SecretStore for testing.
type mockSecretStore struct {
	createResult *lockbox.CreateResult
	createErr    error
}

func (m *mockSecretStore) Create(_ context.Context, params lockbox.CreateParams) (*lockbox.CreateResult, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createResult, nil
}

func (m *mockSecretStore) List(_ context.Context) ([]lockbox.SecretInfo, error) {
	return nil, nil
}

func (m *mockSecretStore) Verify(_ context.Context, _ string) (*lockbox.VerifyResult, error) {
	return nil, nil
}

// withMockStoreFactory temporarily replaces the package-level storeFactory
// with one that returns the given mock, restoring the original on cleanup.
func withMockStoreFactory(t *testing.T, mock *mockSecretStore) {
	t.Helper()
	original := storeFactory
	storeFactory = func(_ context.Context, _, _, _ string) (lockbox.SecretStore, error) {
		return mock, nil
	}
	t.Cleanup(func() { storeFactory = original })
}

// TestInteractiveFlow_SecretExists_YandexProvider tests the interactive flow when
// the user says a secret store already exists and provides a Yandex secret ID.
// The flow should prompt for the secret ID and generate correct yc-lockbox:// URIs.
// Requirements: 7.1, 7.2
func TestInteractiveFlow_SecretExists_YandexProvider(t *testing.T) {
	// Simulate: "y\n" (secret exists) + "e6q-abc-123\n" (secret ID)
	input := "y\ne6q-abc-123\n"
	reader := strings.NewReader(input)

	uris, err := runInteractiveSecretFlowWithReader(reader, "my-app", "yandex", "ru-central1-a", "folder-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uris == nil {
		t.Fatal("expected non-nil URIs, got nil")
	}

	// Verify all three required entries are present with correct format
	for _, key := range lockbox.RequiredEntries() {
		uri, ok := uris[key]
		if !ok {
			t.Errorf("missing URI for key %q", key)
			continue
		}
		expected := fmt.Sprintf("yc-lockbox://e6q-abc-123/%s", key)
		if uri != expected {
			t.Errorf("key %q: got %q, want %q", key, uri, expected)
		}
	}
}

// TestInteractiveFlow_SecretExists_AWSProvider tests the interactive flow when
// the user says a secret store already exists and provides an AWS secret name.
// The flow should prompt for the secret name and generate correct aws-sm:// URIs.
// Requirements: 7.1, 7.2
func TestInteractiveFlow_SecretExists_AWSProvider(t *testing.T) {
	// Simulate: "yes\n" (secret exists) + "polar-gosling/my-app\n" (secret name)
	input := "yes\npolar-gosling/my-app\n"
	reader := strings.NewReader(input)

	uris, err := runInteractiveSecretFlowWithReader(reader, "my-app", "aws", "us-east-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uris == nil {
		t.Fatal("expected non-nil URIs, got nil")
	}

	for _, key := range lockbox.RequiredEntries() {
		uri, ok := uris[key]
		if !ok {
			t.Errorf("missing URI for key %q", key)
			continue
		}
		expected := fmt.Sprintf("aws-sm://polar-gosling/my-app/%s", key)
		if uri != expected {
			t.Errorf("key %q: got %q, want %q", key, uri, expected)
		}
	}
}

// TestInteractiveFlow_SecretExists_EmptyIdentifier tests that providing an empty
// secret identifier returns an error.
// Requirements: 7.2
func TestInteractiveFlow_SecretExists_EmptyIdentifier(t *testing.T) {
	// Simulate: "y\n" (secret exists) + "\n" (empty identifier)
	input := "y\n\n"
	reader := strings.NewReader(input)

	_, err := runInteractiveSecretFlowWithReader(reader, "my-app", "yandex", "ru-central1-a", "folder-xyz")
	if err == nil {
		t.Fatal("expected error for empty identifier, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error about empty identifier, got: %v", err)
	}
}

// TestInteractiveFlow_NoSecret_DeclineCreation tests the interactive flow when
// the user says no secret exists and declines to create one.
// The flow should return nil URIs (placeholder mode).
// Requirements: 7.5
func TestInteractiveFlow_NoSecret_DeclineCreation(t *testing.T) {
	// Simulate: "n\n" (no secret) + "n\n" (decline creation)
	input := "n\nn\n"
	reader := strings.NewReader(input)

	uris, err := runInteractiveSecretFlowWithReader(reader, "my-app", "yandex", "ru-central1-a", "folder-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uris != nil {
		t.Errorf("expected nil URIs for declined creation, got: %v", uris)
	}
}

// TestInteractiveFlow_NoSecret_AcceptCreation tests the interactive flow when
// the user says no secret exists and accepts creation.
// The flow should call SecretStore.Create and return the resulting URIs.
// Requirements: 7.3, 7.4
func TestInteractiveFlow_NoSecret_AcceptCreation(t *testing.T) {
	mockURIs := lockbox.GenerateAllURIs("yandex", "e6q-new-secret-id")
	mock := &mockSecretStore{
		createResult: &lockbox.CreateResult{
			ID:   "e6q-new-secret-id",
			URIs: mockURIs,
		},
	}
	withMockStoreFactory(t, mock)

	// Simulate: "n\n" (no secret) + "y\n" (accept creation)
	input := "n\ny\n"
	reader := strings.NewReader(input)

	uris, err := runInteractiveSecretFlowWithReader(reader, "my-app", "yandex", "ru-central1-a", "folder-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uris == nil {
		t.Fatal("expected non-nil URIs after creation, got nil")
	}

	for _, key := range lockbox.RequiredEntries() {
		uri, ok := uris[key]
		if !ok {
			t.Errorf("missing URI for key %q", key)
			continue
		}
		expected := fmt.Sprintf("yc-lockbox://e6q-new-secret-id/%s", key)
		if uri != expected {
			t.Errorf("key %q: got %q, want %q", key, uri, expected)
		}
	}
}

// TestInteractiveFlow_NoSecret_AcceptCreation_AWS tests creation flow for AWS provider.
// Requirements: 7.3, 7.4
func TestInteractiveFlow_NoSecret_AcceptCreation_AWS(t *testing.T) {
	mockURIs := lockbox.GenerateAllURIs("aws", "polar-gosling/my-app")
	mock := &mockSecretStore{
		createResult: &lockbox.CreateResult{
			ID:   "polar-gosling/my-app",
			URIs: mockURIs,
		},
	}
	withMockStoreFactory(t, mock)

	input := "n\nyes\n"
	reader := strings.NewReader(input)

	uris, err := runInteractiveSecretFlowWithReader(reader, "my-app", "aws", "us-east-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uris == nil {
		t.Fatal("expected non-nil URIs after creation, got nil")
	}

	for _, key := range lockbox.RequiredEntries() {
		uri, ok := uris[key]
		if !ok {
			t.Errorf("missing URI for key %q", key)
			continue
		}
		expected := fmt.Sprintf("aws-sm://polar-gosling/my-app/%s", key)
		if uri != expected {
			t.Errorf("key %q: got %q, want %q", key, uri, expected)
		}
	}
}

// TestInteractiveFlow_CreateFails_YandexMissingFolderID tests that attempting to
// create a Yandex secret without --folder-id returns an error.
// Requirements: 7.3
func TestInteractiveFlow_CreateFails_YandexMissingFolderID(t *testing.T) {
	input := "n\ny\n"
	reader := strings.NewReader(input)

	_, err := runInteractiveSecretFlowWithReader(reader, "my-app", "yandex", "ru-central1-a", "")
	if err == nil {
		t.Fatal("expected error for missing folder-id, got nil")
	}
	if !strings.Contains(err.Error(), "folder-id") {
		t.Errorf("expected error about folder-id, got: %v", err)
	}
}

// TestInteractiveFlow_CreateFails_StoreError tests that a store creation error
// is propagated back to the caller.
// Requirements: 7.3
func TestInteractiveFlow_CreateFails_StoreError(t *testing.T) {
	mock := &mockSecretStore{
		createErr: fmt.Errorf("API error: permission denied"),
	}
	withMockStoreFactory(t, mock)

	input := "n\ny\n"
	reader := strings.NewReader(input)

	_, err := runInteractiveSecretFlowWithReader(reader, "my-app", "aws", "us-east-1", "")
	if err == nil {
		t.Fatal("expected error from store creation, got nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected 'permission denied' in error, got: %v", err)
	}
}

// TestNonInteractiveMode_NoStdinReads verifies that when --interactive is not set,
// the add egg command does not attempt to read from stdin.
// This is tested by verifying generateEggConfig works with nil secretURIs (placeholder mode)
// and that the non-interactive path produces valid config without any reader.
// Requirements: 7.5, 8.2
func TestNonInteractiveMode_PlaceholderConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		scheme   string
	}{
		{"yandex placeholder", "yandex", "yc-lockbox"},
		{"aws placeholder", "aws", "aws-sm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// nil secretURIs = non-interactive / placeholder mode
			config := generateEggConfig("my-app", "vm", tt.provider, "us-east-1", nil)

			// Should contain all three secret attributes
			for _, attr := range []string{"gitlab_token_secret", "gitlab_webhook_secret", "git_repo_url_secret"} {
				if !strings.Contains(config, attr) {
					t.Errorf("config missing attribute %q", attr)
				}
			}

			// Should contain TODO comments
			if !strings.Contains(config, "TODO") {
				t.Error("placeholder config should contain TODO comments")
			}

			// Should use correct scheme
			if !strings.Contains(config, tt.scheme+"://TODO-set-secret-id/") {
				t.Errorf("placeholder config should use scheme %q", tt.scheme)
			}
		})
	}
}

// TestNonInteractiveMode_RealURIsConfig verifies that when real URIs are provided
// (non-interactive with all flags), the config has no TODO comments for secrets.
// Requirements: 8.2
func TestNonInteractiveMode_RealURIsConfig(t *testing.T) {
	uris := lockbox.GenerateAllURIs("yandex", "e6q-real-id")
	config := generateEggConfig("my-app", "vm", "yandex", "ru-central1-a", uris)

	// Should contain all three secret attributes with real URIs
	for _, key := range lockbox.RequiredEntries() {
		expected := fmt.Sprintf("yc-lockbox://e6q-real-id/%s", key)
		if !strings.Contains(config, expected) {
			t.Errorf("config missing real URI for key %q", key)
		}
	}

	// Secret attribute lines should NOT have TODO comments above them
	lines := strings.Split(config, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, attr := range []string{"gitlab_token_secret", "gitlab_webhook_secret", "git_repo_url_secret"} {
			if strings.Contains(trimmed, attr) && strings.Contains(trimmed, "=") {
				if i > 0 {
					prevLine := strings.TrimSpace(lines[i-1])
					if strings.Contains(prevLine, "TODO") && strings.Contains(prevLine, "secret") {
						t.Errorf("found TODO comment above %q when real URIs provided", attr)
					}
				}
			}
		}
	}
}

// TestInteractiveFlow_URICount verifies that the interactive flow returns
// exactly len(RequiredEntries) URIs when a secret exists.
// Requirements: 7.2
func TestInteractiveFlow_URICount(t *testing.T) {
	input := "y\ntest-secret-id\n"
	reader := strings.NewReader(input)

	uris, err := runInteractiveSecretFlowWithReader(reader, "my-app", "yandex", "ru-central1-a", "folder-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := len(lockbox.RequiredEntries())
	if len(uris) != expected {
		t.Errorf("expected %d URIs, got %d", expected, len(uris))
	}
}
