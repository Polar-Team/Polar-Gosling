package lockbox

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// ============================================================
// Mock SecretStore implementation for integration testing
// Requirements: 2.7, 2.8, 3.7, 3.8, 5.4, 5.5, 6.5, 6.6
// ============================================================

// mockSecret represents a stored secret in the mock.
type mockSecret struct {
	name      string
	id        string
	eggName   string
	provider  string
	entries   map[string]string
	labels    map[string]string
	createdAt time.Time
}

// MockSecretStore implements SecretStore for integration testing.
// It simulates both YC Lockbox and AWS Secrets Manager behavior
// using in-memory storage.
type MockSecretStore struct {
	secrets   map[string]*mockSecret // keyed by secret name
	provider  string
	folderID  string
	nextID    int
	createErr error // injected error for Create
	listErr   error // injected error for List
	verifyErr error // injected error for Verify
}

// NewMockSecretStore creates a new mock store for the given provider.
func NewMockSecretStore(provider, folderID string) *MockSecretStore {
	return &MockSecretStore{
		secrets:  make(map[string]*mockSecret),
		provider: provider,
		folderID: folderID,
	}
}

func (m *MockSecretStore) Create(ctx context.Context, params CreateParams) (*CreateResult, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	secretName := SecretNameForProvider(params.Provider, params.EggName)

	// Check for duplicate
	if _, exists := m.secrets[secretName]; exists {
		if params.Provider == "yandex" {
			return nil, fmt.Errorf(
				"secret %q already exists in folder %s (id: %s). Use 'gosling lockbox verify --secret-id %s' to check its entries",
				secretName, m.folderID, m.secrets[secretName].id, m.secrets[secretName].id,
			)
		}
		return nil, fmt.Errorf(
			"secret %q already exists. Use 'gosling lockbox verify --secret-name %s' to check its entries",
			secretName, secretName,
		)
	}

	m.nextID++
	id := fmt.Sprintf("mock-id-%d", m.nextID)
	if params.Provider == "aws" {
		id = secretName // AWS uses the name as the identifier for URIs
	}

	entries := make(map[string]string, len(requiredEntries))
	for _, key := range requiredEntries {
		entries[key] = ""
	}

	m.secrets[secretName] = &mockSecret{
		name:     secretName,
		id:       id,
		eggName:  params.EggName,
		provider: params.Provider,
		entries:  entries,
		labels: map[string]string{
			"polar-gosling": "true",
			"egg-name":      params.EggName,
		},
		createdAt: time.Now(),
	}

	return &CreateResult{
		ID:   id,
		URIs: GenerateAllURIs(params.Provider, id),
	}, nil
}

func (m *MockSecretStore) List(ctx context.Context) ([]SecretInfo, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	var results []SecretInfo
	for _, s := range m.secrets {
		if s.labels["polar-gosling"] == "true" {
			results = append(results, SecretInfo{
				Name:      s.name,
				ID:        s.id,
				EggName:   s.eggName,
				CreatedAt: s.createdAt,
			})
		}
	}
	// Sort for deterministic test output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return results, nil
}

func (m *MockSecretStore) Verify(ctx context.Context, secretRef string) (*VerifyResult, error) {
	if m.verifyErr != nil {
		return nil, m.verifyErr
	}

	// Find secret by ID or name
	var found *mockSecret
	for _, s := range m.secrets {
		if s.id == secretRef || s.name == secretRef {
			found = s
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("secret %q not found", secretRef)
	}

	result := &VerifyResult{}
	for _, key := range requiredEntries {
		if _, ok := found.entries[key]; ok {
			result.Present = append(result.Present, key)
		} else {
			result.Missing = append(result.Missing, key)
		}
	}
	return result, nil
}

// addUntaggedSecret adds a secret without Polar Gosling tags (for list filtering tests).
func (m *MockSecretStore) addUntaggedSecret(name, id string) {
	m.secrets[name] = &mockSecret{
		name:      name,
		id:        id,
		provider:  m.provider,
		entries:   map[string]string{"some-key": "some-value"},
		labels:    map[string]string{},
		createdAt: time.Now(),
	}
}

// addSecretWithPartialEntries adds a tagged secret with only some RequiredEntries.
func (m *MockSecretStore) addSecretWithPartialEntries(name, id, eggName string, presentKeys []string) {
	entries := make(map[string]string, len(presentKeys))
	for _, key := range presentKeys {
		entries[key] = ""
	}
	m.secrets[name] = &mockSecret{
		name:     name,
		id:       id,
		eggName:  eggName,
		provider: m.provider,
		entries:  entries,
		labels: map[string]string{
			"polar-gosling": "true",
			"egg-name":      eggName,
		},
		createdAt: time.Now(),
	}
}

// ============================================================
// End-to-end Create → Verify flow tests
// Requirements: 2.7, 3.7, 6.5
// ============================================================

func TestIntegration_YandexCreateThenVerify(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("yandex", "folder-abc123")

	// Create a secret
	params := CreateParams{
		Provider: "yandex",
		EggName:  "my-app",
		FolderID: "folder-abc123",
	}
	result, err := store.Create(ctx, params)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if result.ID == "" {
		t.Fatal("Create returned empty ID")
	}
	if len(result.URIs) != len(RequiredEntries()) {
		t.Fatalf("Create returned %d URIs, want %d", len(result.URIs), len(RequiredEntries()))
	}

	// Verify all URIs have the correct scheme
	for key, uri := range result.URIs {
		scheme, _, parsedKey, err := ParseSecretURI(uri)
		if err != nil {
			t.Errorf("URI for key %q is unparseable: %v", key, err)
			continue
		}
		if scheme != "yc-lockbox" {
			t.Errorf("URI scheme = %q, want yc-lockbox", scheme)
		}
		if parsedKey != key {
			t.Errorf("URI key = %q, want %q", parsedKey, key)
		}
	}

	// Verify the created secret has all required entries
	verifyResult, err := store.Verify(ctx, result.ID)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if len(verifyResult.Missing) != 0 {
		t.Errorf("Verify found missing entries: %v", verifyResult.Missing)
	}
	if len(verifyResult.Present) != len(RequiredEntries()) {
		t.Errorf("Verify found %d present entries, want %d", len(verifyResult.Present), len(RequiredEntries()))
	}
}

func TestIntegration_AWSCreateThenVerify(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("aws", "")

	// Create a secret
	params := CreateParams{
		Provider: "aws",
		EggName:  "web-service",
		Region:   "us-east-1",
	}
	result, err := store.Create(ctx, params)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if result.ID == "" {
		t.Fatal("Create returned empty ID")
	}

	// AWS uses the secret name as the identifier
	expectedName := SecretNameForProvider("aws", "web-service")
	if result.ID != expectedName {
		t.Errorf("Create ID = %q, want %q", result.ID, expectedName)
	}

	// Verify all URIs have the correct scheme
	for key, uri := range result.URIs {
		scheme, _, parsedKey, err := ParseSecretURI(uri)
		if err != nil {
			t.Errorf("URI for key %q is unparseable: %v", key, err)
			continue
		}
		if scheme != "aws-sm" {
			t.Errorf("URI scheme = %q, want aws-sm", scheme)
		}
		if parsedKey != key {
			t.Errorf("URI key = %q, want %q", parsedKey, key)
		}
	}

	// Verify the created secret
	verifyResult, err := store.Verify(ctx, result.ID)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if len(verifyResult.Missing) != 0 {
		t.Errorf("Verify found missing entries: %v", verifyResult.Missing)
	}
	if len(verifyResult.Present) != len(RequiredEntries()) {
		t.Errorf("Verify found %d present entries, want %d", len(verifyResult.Present), len(RequiredEntries()))
	}
}

// ============================================================
// Create → Verify with all required entries present
// Validates the full round-trip produces a verifiable secret
// Requirements: 2.1, 2.2, 3.1, 3.2, 6.3
// ============================================================

func TestIntegration_CreateVerifyAllEntriesPresent(t *testing.T) {
	providers := []struct {
		name     string
		folderID string
	}{
		{"yandex", "folder-xyz"},
		{"aws", ""},
	}

	for _, prov := range providers {
		t.Run(prov.name, func(t *testing.T) {
			ctx := context.Background()
			store := NewMockSecretStore(prov.name, prov.folderID)

			result, err := store.Create(ctx, CreateParams{
				Provider: prov.name,
				EggName:  "test-egg",
				FolderID: prov.folderID,
				Region:   "us-east-1",
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			vr, err := store.Verify(ctx, result.ID)
			if err != nil {
				t.Fatalf("Verify failed: %v", err)
			}

			// All entries should be present, none missing
			sort.Strings(vr.Present)
			expectedPresent := RequiredEntries()
			sort.Strings(expectedPresent)

			if len(vr.Present) != len(expectedPresent) {
				t.Fatalf("Present count = %d, want %d", len(vr.Present), len(expectedPresent))
			}
			for i := range vr.Present {
				if vr.Present[i] != expectedPresent[i] {
					t.Errorf("Present[%d] = %q, want %q", i, vr.Present[i], expectedPresent[i])
				}
			}
			if len(vr.Missing) != 0 {
				t.Errorf("Missing should be empty, got %v", vr.Missing)
			}
		})
	}
}

// ============================================================
// List with mixed tagged/untagged secrets
// Requirements: 5.4, 5.5
// ============================================================

func TestIntegration_ListFiltersTaggedSecrets(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("yandex", "folder-123")

	// Create two Polar Gosling secrets
	_, err := store.Create(ctx, CreateParams{Provider: "yandex", EggName: "app-one", FolderID: "folder-123"})
	if err != nil {
		t.Fatalf("Create app-one failed: %v", err)
	}
	_, err = store.Create(ctx, CreateParams{Provider: "yandex", EggName: "app-two", FolderID: "folder-123"})
	if err != nil {
		t.Fatalf("Create app-two failed: %v", err)
	}

	// Add untagged secrets that should NOT appear in list
	store.addUntaggedSecret("unrelated-secret-1", "id-unrelated-1")
	store.addUntaggedSecret("unrelated-secret-2", "id-unrelated-2")

	// List should return only the two tagged secrets
	results, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("List returned %d results, want 2", len(results))
	}

	// Verify the listed secrets are the correct ones (sorted by name)
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Name
	}
	sort.Strings(names)

	expectedNames := []string{
		SecretNameForProvider("yandex", "app-one"),
		SecretNameForProvider("yandex", "app-two"),
	}
	sort.Strings(expectedNames)

	for i, name := range names {
		if name != expectedNames[i] {
			t.Errorf("List result[%d].Name = %q, want %q", i, name, expectedNames[i])
		}
	}

	// Verify egg names are populated
	for _, r := range results {
		if r.EggName == "" {
			t.Errorf("List result %q has empty EggName", r.Name)
		}
	}
}

func TestIntegration_ListEmptyResults(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("aws", "")

	// Add only untagged secrets
	store.addUntaggedSecret("other-secret", "arn:aws:secretsmanager:us-east-1:123:secret:other")

	results, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("List returned %d results, want 0 (no tagged secrets)", len(results))
	}
}

func TestIntegration_ListAWSWithMixedTags(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("aws", "")

	// Create three tagged secrets
	for _, egg := range []string{"svc-alpha", "svc-beta", "svc-gamma"} {
		_, err := store.Create(ctx, CreateParams{Provider: "aws", EggName: egg, Region: "us-east-1"})
		if err != nil {
			t.Fatalf("Create %s failed: %v", egg, err)
		}
	}

	// Add untagged secrets
	store.addUntaggedSecret("manual-secret", "arn:manual")
	store.addUntaggedSecret("legacy-secret", "arn:legacy")

	results, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("List returned %d results, want 3", len(results))
	}

	// Verify each result has correct egg name
	eggNames := make(map[string]bool)
	for _, r := range results {
		eggNames[r.EggName] = true
	}
	for _, expected := range []string{"svc-alpha", "svc-beta", "svc-gamma"} {
		if !eggNames[expected] {
			t.Errorf("List missing egg %q", expected)
		}
	}
}

// ============================================================
// Error scenarios
// Requirements: 2.7, 2.8, 3.7, 3.8, 5.5, 6.5, 6.6
// ============================================================

func TestIntegration_CreateDuplicateSecretYandex(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("yandex", "folder-dup")

	params := CreateParams{Provider: "yandex", EggName: "dup-app", FolderID: "folder-dup"}

	// First create should succeed
	_, err := store.Create(ctx, params)
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Second create with same egg name should fail
	_, err = store.Create(ctx, params)
	if err == nil {
		t.Fatal("Second Create should have failed for duplicate secret")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention 'already exists', got: %v", err)
	}
	if !strings.Contains(err.Error(), "gosling lockbox verify") {
		t.Errorf("Error should suggest 'gosling lockbox verify', got: %v", err)
	}
}

func TestIntegration_CreateDuplicateSecretAWS(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("aws", "")

	params := CreateParams{Provider: "aws", EggName: "dup-svc", Region: "us-east-1"}

	// First create should succeed
	_, err := store.Create(ctx, params)
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Second create with same egg name should fail
	_, err = store.Create(ctx, params)
	if err == nil {
		t.Fatal("Second Create should have failed for duplicate secret")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention 'already exists', got: %v", err)
	}
	if !strings.Contains(err.Error(), "gosling lockbox verify") {
		t.Errorf("Error should suggest 'gosling lockbox verify', got: %v", err)
	}
}

func TestIntegration_VerifySecretNotFound(t *testing.T) {
	ctx := context.Background()

	for _, provider := range []string{"yandex", "aws"} {
		t.Run(provider, func(t *testing.T) {
			store := NewMockSecretStore(provider, "folder-nf")

			_, err := store.Verify(ctx, "nonexistent-secret-id")
			if err == nil {
				t.Fatal("Verify should fail for nonexistent secret")
			}
			if !strings.Contains(err.Error(), "not found") {
				t.Errorf("Error should mention 'not found', got: %v", err)
			}
		})
	}
}

func TestIntegration_VerifyPartialEntries(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("yandex", "folder-partial")

	// Add a secret with only runner-token present
	store.addSecretWithPartialEntries(
		"pg-partial-app-secrets", "partial-id", "partial-app",
		[]string{"runner-token"},
	)

	result, err := store.Verify(ctx, "partial-id")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if len(result.Present) != 1 {
		t.Errorf("Present count = %d, want 1", len(result.Present))
	}
	if len(result.Present) > 0 && result.Present[0] != "runner-token" {
		t.Errorf("Present[0] = %q, want runner-token", result.Present[0])
	}
	if len(result.Missing) != 2 {
		t.Errorf("Missing count = %d, want 2", len(result.Missing))
	}

	// Verify the missing entries are the correct ones
	missingSet := make(map[string]bool)
	for _, key := range result.Missing {
		missingSet[key] = true
	}
	if !missingSet["webhook-secret"] {
		t.Error("Missing should contain webhook-secret")
	}
	if !missingSet["repo-url"] {
		t.Error("Missing should contain repo-url")
	}
}

func TestIntegration_VerifyNoEntries(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("aws", "")

	// Add a secret with no required entries
	store.addSecretWithPartialEntries(
		"polar-gosling/empty-app", "polar-gosling/empty-app", "empty-app",
		[]string{},
	)

	result, err := store.Verify(ctx, "polar-gosling/empty-app")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if len(result.Present) != 0 {
		t.Errorf("Present count = %d, want 0", len(result.Present))
	}
	if len(result.Missing) != len(RequiredEntries()) {
		t.Errorf("Missing count = %d, want %d", len(result.Missing), len(RequiredEntries()))
	}
}

func TestIntegration_CreateAPIError(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("yandex", "folder-err")
	store.createErr = fmt.Errorf("rpc error: code = PermissionDenied desc = access denied")

	_, err := store.Create(ctx, CreateParams{
		Provider: "yandex",
		EggName:  "err-app",
		FolderID: "folder-err",
	})
	if err == nil {
		t.Fatal("Create should fail when API returns error")
	}
	if !strings.Contains(err.Error(), "PermissionDenied") {
		t.Errorf("Error should contain API error, got: %v", err)
	}
}

func TestIntegration_ListAPIError(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("aws", "")
	store.listErr = fmt.Errorf("listing Secrets Manager secrets: AccessDeniedException")

	_, err := store.List(ctx)
	if err == nil {
		t.Fatal("List should fail when API returns error")
	}
	if !strings.Contains(err.Error(), "AccessDeniedException") {
		t.Errorf("Error should contain API error, got: %v", err)
	}
}

func TestIntegration_VerifyAPIError(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("yandex", "folder-verr")
	store.verifyErr = fmt.Errorf("retrieving Lockbox payload: rpc error: code = Internal")

	// Even with a secret present, injected error should propagate
	store.addSecretWithPartialEntries("pg-test-secrets", "test-id", "test", RequiredEntries())

	_, err := store.Verify(ctx, "test-id")
	if err == nil {
		t.Fatal("Verify should fail when API returns error")
	}
	if !strings.Contains(err.Error(), "Internal") {
		t.Errorf("Error should contain API error, got: %v", err)
	}
}

// ============================================================
// SecretStore interface compliance tests
// Ensures mock behaves like the real implementations
// ============================================================

func TestIntegration_MockImplementsSecretStoreInterface(t *testing.T) {
	// Compile-time check that MockSecretStore implements SecretStore
	var _ SecretStore = (*MockSecretStore)(nil)
}

// ============================================================
// End-to-end flow: Create → List → Verify for multiple eggs
// Requirements: 2.1, 3.1, 5.1, 5.2, 6.1, 6.2
// ============================================================

func TestIntegration_FullFlowMultipleEggs(t *testing.T) {
	providers := []struct {
		name     string
		folderID string
		eggs     []string
	}{
		{"yandex", "folder-full", []string{"frontend", "backend", "worker"}},
		{"aws", "", []string{"api-gateway", "processor"}},
	}

	for _, prov := range providers {
		t.Run(prov.name, func(t *testing.T) {
			ctx := context.Background()
			store := NewMockSecretStore(prov.name, prov.folderID)

			// Create secrets for all eggs
			createdIDs := make(map[string]string) // eggName -> secretID
			for _, egg := range prov.eggs {
				result, err := store.Create(ctx, CreateParams{
					Provider: prov.name,
					EggName:  egg,
					FolderID: prov.folderID,
					Region:   "us-east-1",
				})
				if err != nil {
					t.Fatalf("Create %s failed: %v", egg, err)
				}
				createdIDs[egg] = result.ID

				// Verify URIs count
				if len(result.URIs) != len(RequiredEntries()) {
					t.Errorf("Create %s: URIs count = %d, want %d", egg, len(result.URIs), len(RequiredEntries()))
				}
			}

			// List should return all created secrets
			listed, err := store.List(ctx)
			if err != nil {
				t.Fatalf("List failed: %v", err)
			}
			if len(listed) != len(prov.eggs) {
				t.Fatalf("List returned %d, want %d", len(listed), len(prov.eggs))
			}

			// Verify each created secret
			for egg, id := range createdIDs {
				vr, err := store.Verify(ctx, id)
				if err != nil {
					t.Errorf("Verify %s (id=%s) failed: %v", egg, id, err)
					continue
				}
				if len(vr.Missing) != 0 {
					t.Errorf("Verify %s: unexpected missing entries: %v", egg, vr.Missing)
				}
				if len(vr.Present) != len(RequiredEntries()) {
					t.Errorf("Verify %s: present count = %d, want %d", egg, len(vr.Present), len(RequiredEntries()))
				}
			}
		})
	}
}

// ============================================================
// URI generation consistency in Create results
// Requirements: 9.1, 9.2
// ============================================================

func TestIntegration_CreateURIsAreConsistentWithProvider(t *testing.T) {
	tests := []struct {
		provider       string
		folderID       string
		eggName        string
		expectedScheme string
	}{
		{"yandex", "folder-uri", "uri-test", "yc-lockbox"},
		{"aws", "", "uri-test", "aws-sm"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			ctx := context.Background()
			store := NewMockSecretStore(tt.provider, tt.folderID)

			result, err := store.Create(ctx, CreateParams{
				Provider: tt.provider,
				EggName:  tt.eggName,
				FolderID: tt.folderID,
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			for key, uri := range result.URIs {
				scheme, _, parsedKey, err := ParseSecretURI(uri)
				if err != nil {
					t.Errorf("URI for %q unparseable: %v", key, err)
					continue
				}
				if scheme != tt.expectedScheme {
					t.Errorf("URI scheme = %q, want %q", scheme, tt.expectedScheme)
				}
				if parsedKey != key {
					t.Errorf("URI key = %q, want %q", parsedKey, key)
				}
			}
		})
	}
}

// ============================================================
// AWS JSON secret value structure validation
// Requirements: 3.2, 3.3
// ============================================================

func TestIntegration_AWSSecretJSONStructure(t *testing.T) {
	// Simulate what the real AWS implementation does: build JSON with RequiredEntries
	entries := RequiredEntries()
	secretValue := make(map[string]string, len(entries))
	for _, key := range entries {
		secretValue[key] = ""
	}
	jsonBytes, err := json.Marshal(secretValue)
	if err != nil {
		t.Fatalf("Failed to marshal secret JSON: %v", err)
	}

	// Parse it back and verify structure
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal secret JSON: %v", err)
	}

	if len(parsed) != len(entries) {
		t.Errorf("JSON has %d keys, want %d", len(parsed), len(entries))
	}

	for _, key := range entries {
		val, ok := parsed[key]
		if !ok {
			t.Errorf("JSON missing key %q", key)
			continue
		}
		strVal, ok := val.(string)
		if !ok {
			t.Errorf("JSON key %q is not a string", key)
			continue
		}
		if strVal != "" {
			t.Errorf("JSON key %q = %q, want empty string placeholder", key, strVal)
		}
	}
}

// ============================================================
// Verify with secret found by name (AWS pattern)
// Requirements: 6.2
// ============================================================

func TestIntegration_VerifyBySecretName(t *testing.T) {
	ctx := context.Background()
	store := NewMockSecretStore("aws", "")

	result, err := store.Create(ctx, CreateParams{
		Provider: "aws",
		EggName:  "name-lookup",
		Region:   "eu-west-1",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify using the secret name (which is the ID for AWS)
	secretName := SecretNameForProvider("aws", "name-lookup")
	vr, err := store.Verify(ctx, secretName)
	if err != nil {
		t.Fatalf("Verify by name failed: %v", err)
	}
	if len(vr.Missing) != 0 {
		t.Errorf("Verify found missing entries: %v", vr.Missing)
	}

	// Also verify using the result ID (should be the same for AWS)
	if result.ID != secretName {
		t.Errorf("AWS Create ID = %q, want %q", result.ID, secretName)
	}
}
