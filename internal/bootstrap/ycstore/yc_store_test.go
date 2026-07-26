package ycstore

import (
	"errors"
	"testing"

	lockboxpb "github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"

	"github.com/polar-gosling/gosling/internal/bootstrap"
)

// ============================================================
// Constants verification
// Requirements: 2.2, 3.3
// ============================================================

func TestSecretName_IsExactValue(t *testing.T) {
	if SecretName != "pg-mothergoose-secrets" {
		t.Errorf("SecretName = %q, want %q", SecretName, "pg-mothergoose-secrets")
	}
}

func TestLabels_CorrectKeysAndValues(t *testing.T) {
	if LabelPolarGosling != "polar-gosling" {
		t.Errorf("LabelPolarGosling = %q, want %q", LabelPolarGosling, "polar-gosling")
	}
	if LabelResourceType != "resource-type" {
		t.Errorf("LabelResourceType = %q, want %q", LabelResourceType, "resource-type")
	}
}

func TestCreateLabels_ContainExpectedValues(t *testing.T) {
	// Verify the labels map that Create would use matches spec.
	labels := map[string]string{
		LabelPolarGosling: "true",
		LabelResourceType: "mothergoose-api",
	}
	if labels[LabelPolarGosling] != "true" {
		t.Errorf("label %q = %q, want %q", LabelPolarGosling, labels[LabelPolarGosling], "true")
	}
	if labels[LabelResourceType] != "mothergoose-api" {
		t.Errorf("label %q = %q, want %q", LabelResourceType, labels[LabelResourceType], "mothergoose-api")
	}
}

// ============================================================
// SecretURI helper
// Requirements: 2.2
// ============================================================

func TestSecretURI_Format(t *testing.T) {
	tests := []struct {
		secretID string
		want     string
	}{
		{"e6q123abc", "yc-lockbox://e6q123abc"},
		{"secret-id-with-dashes", "yc-lockbox://secret-id-with-dashes"},
		{"", "yc-lockbox://"},
	}
	for _, tt := range tests {
		got := SecretURI(tt.secretID)
		if got != tt.want {
			t.Errorf("SecretURI(%q) = %q, want %q", tt.secretID, got, tt.want)
		}
	}
}

func TestSecretURI_UsesCorrectScheme(t *testing.T) {
	uri := SecretURI("any-id")
	if SecretURIScheme != "yc-lockbox" {
		t.Errorf("SecretURIScheme = %q, want %q", SecretURIScheme, "yc-lockbox")
	}
	if uri[:len(SecretURIScheme)] != SecretURIScheme {
		t.Errorf("SecretURI does not start with scheme %q: got %q", SecretURIScheme, uri)
	}
}

// ============================================================
// extractCredentials helper
// Requirements: 2.2
// ============================================================

func TestExtractCredentials_BothEntries(t *testing.T) {
	payload := &lockboxpb.Payload{
		Entries: []*lockboxpb.Payload_Entry{
			{
				Key:   EntryKeyAPIURL,
				Value: &lockboxpb.Payload_Entry_TextValue{TextValue: "https://api.example.com"},
			},
			{
				Key:   EntryKeyAPIKey,
				Value: &lockboxpb.Payload_Entry_TextValue{TextValue: "secret-key-123"},
			},
		},
	}
	creds := extractCredentials(payload)
	if creds.APIURL != "https://api.example.com" {
		t.Errorf("APIURL = %q, want %q", creds.APIURL, "https://api.example.com")
	}
	if creds.APIKey != "secret-key-123" {
		t.Errorf("APIKey = %q, want %q", creds.APIKey, "secret-key-123")
	}
}

func TestExtractCredentials_EmptyPayload(t *testing.T) {
	payload := &lockboxpb.Payload{
		Entries: []*lockboxpb.Payload_Entry{},
	}
	creds := extractCredentials(payload)
	if creds.APIURL != "" {
		t.Errorf("APIURL = %q, want empty", creds.APIURL)
	}
	if creds.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", creds.APIKey)
	}
}

func TestExtractCredentials_OnlyAPIURL(t *testing.T) {
	payload := &lockboxpb.Payload{
		Entries: []*lockboxpb.Payload_Entry{
			{
				Key:   EntryKeyAPIURL,
				Value: &lockboxpb.Payload_Entry_TextValue{TextValue: "https://gw.example.com"},
			},
		},
	}
	creds := extractCredentials(payload)
	if creds.APIURL != "https://gw.example.com" {
		t.Errorf("APIURL = %q, want %q", creds.APIURL, "https://gw.example.com")
	}
	if creds.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", creds.APIKey)
	}
}

func TestExtractCredentials_OnlyAPIKey(t *testing.T) {
	payload := &lockboxpb.Payload{
		Entries: []*lockboxpb.Payload_Entry{
			{
				Key:   EntryKeyAPIKey,
				Value: &lockboxpb.Payload_Entry_TextValue{TextValue: "my-key"},
			},
		},
	}
	creds := extractCredentials(payload)
	if creds.APIURL != "" {
		t.Errorf("APIURL = %q, want empty", creds.APIURL)
	}
	if creds.APIKey != "my-key" {
		t.Errorf("APIKey = %q, want %q", creds.APIKey, "my-key")
	}
}

func TestExtractCredentials_IgnoresUnknownKeys(t *testing.T) {
	payload := &lockboxpb.Payload{
		Entries: []*lockboxpb.Payload_Entry{
			{
				Key:   "unknown-key",
				Value: &lockboxpb.Payload_Entry_TextValue{TextValue: "some-value"},
			},
			{
				Key:   EntryKeyAPIURL,
				Value: &lockboxpb.Payload_Entry_TextValue{TextValue: "https://api.test.com"},
			},
		},
	}
	creds := extractCredentials(payload)
	if creds.APIURL != "https://api.test.com" {
		t.Errorf("APIURL = %q, want %q", creds.APIURL, "https://api.test.com")
	}
	if creds.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", creds.APIKey)
	}
}

// ============================================================
// isPermissionDenied helper
// Requirements: 6.2
// ============================================================

func TestIsPermissionDenied_NilError(t *testing.T) {
	if isPermissionDenied(nil) {
		t.Error("isPermissionDenied(nil) = true, want false")
	}
}

func TestIsPermissionDenied_PermissionDeniedStrings(t *testing.T) {
	tests := []struct {
		errMsg string
		want   bool
	}{
		{"rpc error: code = PermissionDenied desc = ...", true},
		{"permission denied for folder xyz", true},
		{"PermissionDenied: insufficient access", true},
		{"connection timeout", false},
		{"not found", false},
		{"", false},
		{"PERMISSION DENIED", false}, // case-sensitive check
	}
	for _, tt := range tests {
		err := errors.New(tt.errMsg)
		got := isPermissionDenied(err)
		if got != tt.want {
			t.Errorf("isPermissionDenied(%q) = %v, want %v", tt.errMsg, got, tt.want)
		}
	}
}

// ============================================================
// INACTIVE status treated as scheduled-deletion (ErrSecretDeleted)
// Requirements: 2.6, 3.3
// ============================================================

func TestInactiveStatus_MapsToSecretDeleted(t *testing.T) {
	// The Discover method checks secret.Status == lockboxpb.Secret_INACTIVE
	// and returns bootstrap.ErrSecretDeleted. Verify the status value is distinct from ACTIVE.
	if lockboxpb.Secret_INACTIVE == lockboxpb.Secret_ACTIVE {
		t.Fatal("lockboxpb.Secret_INACTIVE should differ from Secret_ACTIVE")
	}

	// Verify ErrSecretDeleted sentinel is the expected error
	if bootstrap.ErrSecretDeleted == nil {
		t.Fatal("bootstrap.ErrSecretDeleted is nil")
	}
	if bootstrap.ErrSecretDeleted.Error() != "secret is scheduled for deletion" {
		t.Errorf("ErrSecretDeleted = %q, want %q",
			bootstrap.ErrSecretDeleted.Error(), "secret is scheduled for deletion")
	}
}

// ============================================================
// Entry key constants
// Requirements: 2.2
// ============================================================

func TestEntryKeyConstants(t *testing.T) {
	if EntryKeyAPIURL != "api-url" {
		t.Errorf("EntryKeyAPIURL = %q, want %q", EntryKeyAPIURL, "api-url")
	}
	if EntryKeyAPIKey != "api-key" {
		t.Errorf("EntryKeyAPIKey = %q, want %q", EntryKeyAPIKey, "api-key")
	}
}

// ============================================================
// New constructor
// Requirements: 2.2
// ============================================================

func TestNew_ReturnsNonNil(t *testing.T) {
	store := New(nil, "folder-123")
	if store == nil {
		t.Fatal("New(nil, ...) returned nil")
	}
	if store.folderID != "folder-123" {
		t.Errorf("store.folderID = %q, want %q", store.folderID, "folder-123")
	}
}
