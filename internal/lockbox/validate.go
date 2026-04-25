package lockbox

import "fmt"

// ValidateProvider checks that provider is a supported value ("yandex" or "aws").
func ValidateProvider(provider string) error {
	if provider != "yandex" && provider != "aws" {
		return fmt.Errorf("invalid provider %q: must be 'yandex' or 'aws'", provider)
	}
	return nil
}

// ValidateProviderFlags checks provider value and provider-specific required fields
// (folder-id for Yandex). Suitable for list/verify commands that don't need egg-name.
func ValidateProviderFlags(provider, folderID string) error {
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	if provider == "yandex" && folderID == "" {
		return fmt.Errorf("folder-id is required for Yandex Cloud provider")
	}
	return nil
}

// ValidateCreateInput validates all inputs before any cloud API call.
func ValidateCreateInput(params CreateParams) error {
	if err := ValidateProvider(params.Provider); err != nil {
		return err
	}
	if params.EggName == "" {
		return fmt.Errorf("egg-name is required")
	}
	if !IsValidEggName(params.EggName) {
		return fmt.Errorf("invalid egg-name %q: must contain only alphanumeric characters, hyphens, or underscores", params.EggName)
	}
	if params.Provider == "yandex" && params.FolderID == "" {
		return fmt.Errorf("folder-id is required for Yandex Cloud provider")
	}
	// AWS region: if empty, SDK uses default from config (Requirement 4.5)
	return nil
}

// IsValidEggName checks that a name contains only [a-zA-Z0-9_-].
func IsValidEggName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}
