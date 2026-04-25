package cli

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/polar-gosling/gosling/internal/lockbox"
)

// genValidEggNameForAdd generates random valid egg names: non-empty strings of [a-zA-Z0-9_-].
func genValidEggNameForAdd() gopter.Gen {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	return gen.IntRange(1, 64).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		return gen.SliceOfN(length, gen.IntRange(0, len(chars)-1)).Map(func(idxs []int) string {
			buf := make([]byte, len(idxs))
			for i, idx := range idxs {
				buf[i] = chars[idx]
			}
			return string(buf)
		})
	}, reflect.TypeOf(""))
}

// genIdentifier generates non-empty identifier strings suitable for secret URIs.
// Allows alphanumeric, hyphens, underscores, dots, and slashes (for AWS paths like "polar-gosling/my-app").
// Excludes "://" to avoid breaking URI parsing.
func genIdentifier() gopter.Gen {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_/."
	return gen.IntRange(1, 48).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		return gen.SliceOfN(length, gen.IntRange(0, len(chars)-1)).Map(func(idxs []int) string {
			buf := make([]byte, len(idxs))
			for i, idx := range idxs {
				buf[i] = chars[idx]
			}
			return string(buf)
		}).SuchThat(func(s string) bool {
			return len(s) > 0 && !strings.Contains(s, "://")
		})
	}, reflect.TypeOf(""))
}

// Feature: gosling-lockbox, Property 6: Config.fly generation includes all secret attributes
// Validates: Requirements 7.6, 10.1
func TestConfigFlyIncludesAllSecretAttributes(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genProvider := gen.OneConstOf("yandex", "aws")
	genRunnerType := gen.OneConstOf("serverless", "apex", "nadir")
	genRegion := gen.OneConstOf("us-east-1", "eu-west-1", "ru-central1-a")

	// Expected scheme per provider
	schemeFor := map[string]string{
		"yandex": "yc-lockbox",
		"aws":    "aws-sm",
	}

	// Mapping from RequiredEntries keys to config.fly attribute names
	attrFor := map[string]string{
		"runner-token":   "gitlab_token_secret",
		"webhook-secret": "gitlab_webhook_secret",
		"repo-url":       "git_repo_url_secret",
	}

	properties.Property(
		"generated config.fly contains all three secret attributes with correctly formatted URIs",
		prop.ForAll(
			func(provider, identifier, eggName, runnerType, region string) bool {
				// Generate real secret URIs using the lockbox package
				secretURIs := lockbox.GenerateAllURIs(provider, identifier)

				// Generate the config.fly content
				config := generateEggConfig(eggName, runnerType, provider, region, secretURIs)

				expectedScheme := schemeFor[provider]

				for _, key := range lockbox.RequiredEntries() {
					attrName := attrFor[key]

					// Check 1: attribute name appears in the config
					if !strings.Contains(config, attrName) {
						t.Logf("config missing attribute %q for key %q (provider=%s)", attrName, key, provider)
						return false
					}

					// Check 2: the full URI appears in the config
					expectedURI := fmt.Sprintf("%s://%s/%s", expectedScheme, identifier, key)
					if !strings.Contains(config, expectedURI) {
						t.Logf("config missing URI %q for attribute %q (provider=%s)", expectedURI, attrName, provider)
						return false
					}

					// Check 3: the attribute is assigned the correct URI (attribute = "uri" pattern)
					expectedAssignment := fmt.Sprintf(`%s   = "%s"`, attrName, expectedURI)
					// Also check with different spacing since fmt alignment may vary
					altAssignment := fmt.Sprintf(`%s = "%s"`, attrName, expectedURI)
					if !strings.Contains(config, expectedAssignment) && !strings.Contains(config, altAssignment) {
						t.Logf("config missing assignment %q or %q", expectedAssignment, altAssignment)
						return false
					}
				}

				return true
			},
			genProvider,
			genIdentifier(),
			genValidEggNameForAdd(),
			genRunnerType,
			genRegion,
		),
	)

	properties.Property(
		"generated config.fly with real URIs does not contain TODO comments for secrets",
		prop.ForAll(
			func(provider, identifier, eggName, runnerType, region string) bool {
				secretURIs := lockbox.GenerateAllURIs(provider, identifier)
				config := generateEggConfig(eggName, runnerType, provider, region, secretURIs)

				// When real URIs are provided, no TODO comments should appear near secret attributes
				lines := strings.Split(config, "\n")
				for i, line := range lines {
					trimmed := strings.TrimSpace(line)
					for _, attr := range []string{"gitlab_token_secret", "gitlab_webhook_secret", "git_repo_url_secret"} {
						if strings.Contains(trimmed, attr) && strings.Contains(trimmed, "=") {
							// Check the line above for a TODO comment about secret store
							if i > 0 {
								prevLine := strings.TrimSpace(lines[i-1])
								if strings.Contains(prevLine, "TODO") && strings.Contains(prevLine, "secret") {
									t.Logf("found TODO comment above %q when real URIs provided", attr)
									return false
								}
							}
						}
					}
				}

				return true
			},
			genProvider,
			genIdentifier(),
			genValidEggNameForAdd(),
			genRunnerType,
			genRegion,
		),
	)

	properties.Property(
		"placeholder config.fly contains TODO comments for each secret attribute",
		prop.ForAll(
			func(provider, eggName, runnerType, region string) bool {
				// Empty secretURIs map triggers placeholder mode
				config := generateEggConfig(eggName, runnerType, provider, region, map[string]string{})

				for _, attr := range []string{"gitlab_token_secret", "gitlab_webhook_secret", "git_repo_url_secret"} {
					if !strings.Contains(config, attr) {
						t.Logf("placeholder config missing attribute %q", attr)
						return false
					}
				}

				// Should contain TODO comments
				if !strings.Contains(config, "TODO") {
					t.Logf("placeholder config missing TODO comments")
					return false
				}

				// Should contain the correct scheme prefix for the provider
				expectedScheme := schemeFor[provider]
				if !strings.Contains(config, expectedScheme+"://") {
					t.Logf("placeholder config missing scheme %q", expectedScheme)
					return false
				}

				return true
			},
			genProvider,
			genValidEggNameForAdd(),
			genRunnerType,
			genRegion,
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
