package lockbox

import (
	"testing"
)

// ============================================================
// ParseSecretURI edge cases
// Requirements: 9.1, 9.2, 9.3
// ============================================================

func TestParseSecretURI_MissingSchemeSeparator(t *testing.T) {
	cases := []string{
		"no-scheme-separator",
		"yc-lockbox/secret-id/key",
		"aws-sm:secret-name/key",
		"just-a-string",
		"",
	}
	for _, uri := range cases {
		_, _, _, err := ParseSecretURI(uri)
		if err == nil {
			t.Errorf("ParseSecretURI(%q) expected error for missing ://, got nil", uri)
		}
	}
}

func TestParseSecretURI_MissingKey(t *testing.T) {
	cases := []string{
		"yc-lockbox://secret-id",      // no slash after identifier
		"aws-sm://secret-name",        // no slash after identifier
		"yc-lockbox://secret-id/",     // trailing slash, empty key
		"aws-sm://polar-gosling/app/", // trailing slash, empty key
	}
	for _, uri := range cases {
		_, _, _, err := ParseSecretURI(uri)
		if err == nil {
			t.Errorf("ParseSecretURI(%q) expected error for missing key, got nil", uri)
		}
	}
}

func TestParseSecretURI_EmptyIdentifier(t *testing.T) {
	// Identifier is empty when the first char after :// is /
	cases := []string{
		"yc-lockbox:///key",
		"aws-sm:///runner-token",
	}
	for _, uri := range cases {
		_, _, _, err := ParseSecretURI(uri)
		if err == nil {
			t.Errorf("ParseSecretURI(%q) expected error for empty identifier, got nil", uri)
		}
	}
}

func TestParseSecretURI_ValidURIs(t *testing.T) {
	tests := []struct {
		uri        string
		wantScheme string
		wantID     string
		wantKey    string
	}{
		{"yc-lockbox://e6q123/runner-token", "yc-lockbox", "e6q123", "runner-token"},
		{"aws-sm://polar-gosling/my-app/webhook-secret", "aws-sm", "polar-gosling/my-app", "webhook-secret"},
		{"custom://id/key", "custom", "id", "key"},
	}
	for _, tt := range tests {
		scheme, id, key, err := ParseSecretURI(tt.uri)
		if err != nil {
			t.Errorf("ParseSecretURI(%q) unexpected error: %v", tt.uri, err)
			continue
		}
		if scheme != tt.wantScheme {
			t.Errorf("ParseSecretURI(%q) scheme = %q, want %q", tt.uri, scheme, tt.wantScheme)
		}
		if id != tt.wantID {
			t.Errorf("ParseSecretURI(%q) identifier = %q, want %q", tt.uri, id, tt.wantID)
		}
		if key != tt.wantKey {
			t.Errorf("ParseSecretURI(%q) key = %q, want %q", tt.uri, key, tt.wantKey)
		}
	}
}

// ============================================================
// IsValidEggName edge cases
// Requirements: 4.1, 4.2, 4.3
// ============================================================

func TestIsValidEggName_EmptyString(t *testing.T) {
	if IsValidEggName("") {
		t.Error("IsValidEggName(\"\") should return false for empty string")
	}
}

func TestIsValidEggName_SpecialCharacters(t *testing.T) {
	invalid := []string{
		"has space",
		"has.dot",
		"has@at",
		"has!bang",
		"has/slash",
		"has:colon",
		"has#hash",
		"has$dollar",
		"emoji-🎉",
		"tab\there",
		"new\nline",
	}
	for _, name := range invalid {
		if IsValidEggName(name) {
			t.Errorf("IsValidEggName(%q) should return false for special characters", name)
		}
	}
}

func TestIsValidEggName_ValidNames(t *testing.T) {
	valid := []string{
		"a",
		"my-app",
		"my_app",
		"MyApp123",
		"ALLCAPS",
		"a-b-c-d",
		"under_score_name",
		"mix-of_both-123",
		"0-starts-with-digit",
	}
	for _, name := range valid {
		if !IsValidEggName(name) {
			t.Errorf("IsValidEggName(%q) should return true for valid name", name)
		}
	}
}

// ============================================================
// GenerateAllURIs
// Requirements: 9.1, 9.2, 9.3
// ============================================================

func TestGenerateAllURIs_ReturnsExactlyRequiredEntries(t *testing.T) {
	providers := []string{"yandex", "aws"}
	expected := RequiredEntries()
	for _, provider := range providers {
		uris := GenerateAllURIs(provider, "test-id")
		if len(uris) != len(expected) {
			t.Errorf("GenerateAllURIs(%q, ...) returned %d entries, want %d",
				provider, len(uris), len(expected))
		}
		for _, key := range expected {
			if _, ok := uris[key]; !ok {
				t.Errorf("GenerateAllURIs(%q, ...) missing key %q", provider, key)
			}
		}
	}
}

func TestGenerateAllURIs_URIsContainIdentifier(t *testing.T) {
	identifier := "my-secret-id"
	for _, provider := range []string{"yandex", "aws"} {
		uris := GenerateAllURIs(provider, identifier)
		for key, uri := range uris {
			scheme, parsedID, parsedKey, err := ParseSecretURI(uri)
			if err != nil {
				t.Errorf("GenerateAllURIs(%q, %q)[%q] produced unparseable URI %q: %v",
					provider, identifier, key, uri, err)
				continue
			}
			if parsedID != identifier {
				t.Errorf("URI identifier = %q, want %q", parsedID, identifier)
			}
			if parsedKey != key {
				t.Errorf("URI key = %q, want %q", parsedKey, key)
			}
			if provider == "yandex" && scheme != "yc-lockbox" {
				t.Errorf("Yandex URI scheme = %q, want yc-lockbox", scheme)
			}
			if provider == "aws" && scheme != "aws-sm" {
				t.Errorf("AWS URI scheme = %q, want aws-sm", scheme)
			}
		}
	}
}

func TestGenerateAllURIs_UnsupportedProvider(t *testing.T) {
	uris := GenerateAllURIs("gcp", "some-id")
	for key, uri := range uris {
		if uri != "" {
			t.Errorf("GenerateAllURIs(\"gcp\", ...)[%q] = %q, want empty string", key, uri)
		}
	}
}
