package lockbox

import (
	"fmt"
	"strings"
)

// GenerateSecretURI builds a secret URI for the given provider, identifier, and key.
func GenerateSecretURI(provider, identifier, key string) string {
	switch provider {
	case "yandex":
		return fmt.Sprintf("yc-lockbox://%s/%s", identifier, key)
	case "aws":
		return fmt.Sprintf("aws-sm://%s/%s", identifier, key)
	default:
		return ""
	}
}

// GenerateAllURIs returns a map of key -> URI for all RequiredEntries.
func GenerateAllURIs(provider, identifier string) map[string]string {
	entries := RequiredEntries()
	uris := make(map[string]string, len(entries))
	for _, key := range entries {
		uris[key] = GenerateSecretURI(provider, identifier, key)
	}
	return uris
}

// ParseSecretURI splits a secret URI into (scheme, identifier, key).
// Returns an error if the URI does not have exactly 3 components.
func ParseSecretURI(uri string) (scheme, identifier, key string, err error) {
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid secret URI %q: missing ://", uri)
	}
	scheme = parts[0]
	rest := parts[1]
	lastSlash := strings.LastIndex(rest, "/")
	if lastSlash < 0 || lastSlash == 0 || lastSlash == len(rest)-1 {
		return "", "", "", fmt.Errorf("invalid secret URI %q: must have format scheme://identifier/key", uri)
	}
	identifier = rest[:lastSlash]
	key = rest[lastSlash+1:]
	return scheme, identifier, key, nil
}

// SecretNameForProvider returns the conventional secret name for a given egg.
func SecretNameForProvider(provider, eggName string) string {
	switch provider {
	case "yandex":
		return fmt.Sprintf("pg-%s-secrets", eggName)
	case "aws":
		return fmt.Sprintf("polar-gosling/%s", eggName)
	default:
		return ""
	}
}
