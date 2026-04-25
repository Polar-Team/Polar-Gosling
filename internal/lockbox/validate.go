package lockbox

import "fmt"

// ValidateCreateInput validates all inputs before any cloud API call.
func ValidateCreateInput(params CreateParams) error {
	if params.Provider != "yandex" && params.Provider != "aws" {
		return fmt.Errorf("invalid provider %q: must be 'yandex' or 'aws'", params.Provider)
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
