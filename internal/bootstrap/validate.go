package bootstrap

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateDeployFlags checks the --api-url and --api-key flag combination.
// Returns (useBootstrap bool, err error).
//   - Both non-empty AND apiURL is valid HTTP(S) URL: useBootstrap=false, err=nil
//   - Both empty: useBootstrap=true, err=nil
//   - One empty, one non-empty: err with descriptive message
//   - apiURL is non-empty but not a valid HTTP(S) URL: err with format guidance
func ValidateDeployFlags(apiURL, apiKey string) (bool, error) {
	apiURL = strings.TrimSpace(apiURL)
	apiKey = strings.TrimSpace(apiKey)

	urlEmpty := apiURL == ""
	keyEmpty := apiKey == ""

	// Both empty → bootstrap mode
	if urlEmpty && keyEmpty {
		return true, nil
	}

	// One provided, one missing → error
	if urlEmpty != keyEmpty {
		return false, fmt.Errorf("both --api-url and --api-key must be provided together, or both omitted for automatic bootstrap")
	}

	// Both provided → validate URL format
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return false, fmt.Errorf("--api-url must be a valid HTTP or HTTPS URL (e.g., https://api-gw.example.com)")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false, fmt.Errorf("--api-url must be a valid HTTP or HTTPS URL (e.g., https://api-gw.example.com)")
	}

	if parsed.Host == "" {
		return false, fmt.Errorf("--api-url must be a valid HTTP or HTTPS URL (e.g., https://api-gw.example.com)")
	}

	return false, nil
}
