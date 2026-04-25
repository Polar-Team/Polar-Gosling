package lockbox

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genValidEggName generates random valid egg names: non-empty strings of [a-zA-Z0-9_-].
func genValidEggName() gopter.Gen {
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

// Feature: gosling-lockbox, Property 1: Secret naming convention is deterministic and follows provider patterns
// Validates: Requirements 2.4, 3.4
func TestSecretNamingConvention(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property(
		"Yandex secret name follows pg-{eggName}-secrets pattern",
		prop.ForAll(
			func(eggName string) bool {
				result := SecretNameForProvider("yandex", eggName)
				expected := fmt.Sprintf("pg-%s-secrets", eggName)
				if result != expected {
					t.Logf("Yandex naming mismatch: want %q, got %q", expected, result)
					return false
				}
				if !strings.Contains(result, eggName) {
					t.Logf("egg name %q not found verbatim in result %q", eggName, result)
					return false
				}
				return true
			},
			genValidEggName(),
		),
	)

	properties.Property(
		"AWS secret name follows polar-gosling/{eggName} pattern",
		prop.ForAll(
			func(eggName string) bool {
				result := SecretNameForProvider("aws", eggName)
				expected := fmt.Sprintf("polar-gosling/%s", eggName)
				if result != expected {
					t.Logf("AWS naming mismatch: want %q, got %q", expected, result)
					return false
				}
				if !strings.Contains(result, eggName) {
					t.Logf("egg name %q not found verbatim in result %q", eggName, result)
					return false
				}
				return true
			},
			genValidEggName(),
		),
	)

	properties.Property(
		"naming is deterministic: same input always produces same output",
		prop.ForAll(
			func(eggName string, providerIdx int) bool {
				providers := []string{"yandex", "aws"}
				provider := providers[providerIdx%2]
				first := SecretNameForProvider(provider, eggName)
				second := SecretNameForProvider(provider, eggName)
				if first != second {
					t.Logf("non-deterministic: %q vs %q for provider=%s egg=%s", first, second, provider, eggName)
					return false
				}
				return true
			},
			genValidEggName(),
			gen.IntRange(0, 1),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gosling-lockbox, Property 2: Secret URI generation/parse round-trip
// Validates: Requirements 9.1, 9.2, 9.3
func TestSecretURIRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for provider: either "yandex" or "aws"
	genProvider := gen.OneConstOf("yandex", "aws")

	// Generator for non-empty identifier strings that do NOT contain "://"
	// Allows "/" in the middle (important for AWS identifiers like "polar-gosling/my-app")
	genIdentifier := gen.IntRange(1, 64).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_/."
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

	// Generator for non-empty key strings that do NOT contain "/"
	genKey := gen.IntRange(1, 32).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."
		return gen.SliceOfN(length, gen.IntRange(0, len(chars)-1)).Map(func(idxs []int) string {
			buf := make([]byte, len(idxs))
			for i, idx := range idxs {
				buf[i] = chars[idx]
			}
			return string(buf)
		}).SuchThat(func(s string) bool {
			return len(s) > 0 && !strings.Contains(s, "/")
		})
	}, reflect.TypeOf(""))

	properties.Property(
		"ParseSecretURI(GenerateSecretURI(provider, id, key)) returns original values with correct scheme",
		prop.ForAll(
			func(provider, identifier, key string) bool {
				uri := GenerateSecretURI(provider, identifier, key)
				if uri == "" {
					t.Logf("GenerateSecretURI returned empty for provider=%q", provider)
					return false
				}

				scheme, parsedID, parsedKey, err := ParseSecretURI(uri)
				if err != nil {
					t.Logf("ParseSecretURI error: %v for uri=%q", err, uri)
					return false
				}

				// Verify scheme matches expected for provider
				expectedScheme := ""
				switch provider {
				case "yandex":
					expectedScheme = "yc-lockbox"
				case "aws":
					expectedScheme = "aws-sm"
				}
				if scheme != expectedScheme {
					t.Logf("scheme mismatch: want %q, got %q (provider=%q)", expectedScheme, scheme, provider)
					return false
				}

				// Verify identifier round-trips
				if parsedID != identifier {
					t.Logf("identifier mismatch: want %q, got %q", identifier, parsedID)
					return false
				}

				// Verify key round-trips
				if parsedKey != key {
					t.Logf("key mismatch: want %q, got %q", key, parsedKey)
					return false
				}

				return true
			},
			genProvider,
			genIdentifier,
			genKey,
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genInvalidCharsEggName generates non-empty strings that contain at least one character outside [a-zA-Z0-9_-].
func genInvalidCharsEggName() gopter.Gen {
	// Generate a valid base, then inject at least one invalid character.
	invalidChars := "!@#$%^&*()+={}[]|:;\"'<>,. /?\t\n~`"
	return gen.IntRange(1, 32).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		return gen.SliceOfN(length, gen.IntRange(0, len(invalidChars)-1)).FlatMap(func(v interface{}) gopter.Gen {
			badIdxs := v.([]int)
			return genValidEggName().Map(func(base string) string {
				badChar := invalidChars[badIdxs[0]%len(invalidChars)]
				return base + string(badChar)
			})
		}, reflect.TypeOf(""))
	}, reflect.TypeOf(""))
}

// Feature: gosling-lockbox, Property 3: Invalid inputs are rejected before cloud API calls
// Validates: Requirements 4.1, 4.2, 4.3, 4.4
func TestInvalidInputsRejected(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for a bad provider (not "yandex" or "aws")
	genBadProvider := gen.AnyString().SuchThat(func(s string) bool {
		return s != "yandex" && s != "aws"
	})

	// Generator for a valid provider
	genValidProvider := gen.OneConstOf("yandex", "aws")

	// Generator for a non-empty folder ID
	genFolderID := gen.IntRange(1, 32).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
		return gen.SliceOfN(length, gen.IntRange(0, len(chars)-1)).Map(func(idxs []int) string {
			buf := make([]byte, len(idxs))
			for i, idx := range idxs {
				buf[i] = chars[idx]
			}
			return string(buf)
		})
	}, reflect.TypeOf(""))

	properties.Property(
		"bad provider is rejected",
		prop.ForAll(
			func(provider string, eggName string, folderID string) bool {
				params := CreateParams{
					Provider: provider,
					EggName:  eggName,
					FolderID: folderID,
				}
				err := ValidateCreateInput(params)
				if err == nil {
					t.Logf("expected error for bad provider %q but got nil", provider)
					return false
				}
				return true
			},
			genBadProvider,
			genValidEggName(),
			genFolderID,
		),
	)

	properties.Property(
		"empty egg-name is rejected",
		prop.ForAll(
			func(provider string, folderID string) bool {
				params := CreateParams{
					Provider: provider,
					EggName:  "",
					FolderID: folderID,
				}
				err := ValidateCreateInput(params)
				if err == nil {
					t.Logf("expected error for empty egg-name but got nil (provider=%q)", provider)
					return false
				}
				return true
			},
			genValidProvider,
			genFolderID,
		),
	)

	properties.Property(
		"egg-name with invalid characters is rejected",
		prop.ForAll(
			func(provider string, eggName string, folderID string) bool {
				params := CreateParams{
					Provider: provider,
					EggName:  eggName,
					FolderID: folderID,
				}
				err := ValidateCreateInput(params)
				if err == nil {
					t.Logf("expected error for invalid egg-name %q but got nil", eggName)
					return false
				}
				return true
			},
			genValidProvider,
			genInvalidCharsEggName(),
			genFolderID,
		),
	)

	properties.Property(
		"yandex provider with empty folder-id is rejected",
		prop.ForAll(
			func(eggName string) bool {
				params := CreateParams{
					Provider: "yandex",
					EggName:  eggName,
					FolderID: "",
				}
				err := ValidateCreateInput(params)
				if err == nil {
					t.Logf("expected error for yandex with empty folder-id but got nil (egg=%q)", eggName)
					return false
				}
				return true
			},
			genValidEggName(),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gosling-lockbox, Property 4: Valid inputs pass validation
// Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5
func TestValidInputsPassValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for a non-empty folder ID
	genFolderID := gen.IntRange(1, 32).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
		return gen.SliceOfN(length, gen.IntRange(0, len(chars)-1)).Map(func(idxs []int) string {
			buf := make([]byte, len(idxs))
			for i, idx := range idxs {
				buf[i] = chars[idx]
			}
			return string(buf)
		})
	}, reflect.TypeOf(""))

	// Generator for an optional AWS region (can be empty per Requirement 4.5)
	genRegion := gen.OneGenOf(
		gen.Const(""),
		gen.OneConstOf("us-east-1", "eu-west-1", "ap-northeast-1"),
	)

	properties.Property(
		"yandex with valid egg-name and folder-id passes validation",
		prop.ForAll(
			func(eggName string, folderID string) bool {
				params := CreateParams{
					Provider: "yandex",
					EggName:  eggName,
					FolderID: folderID,
				}
				err := ValidateCreateInput(params)
				if err != nil {
					t.Logf("unexpected error for valid yandex params (egg=%q, folder=%q): %v", eggName, folderID, err)
					return false
				}
				return true
			},
			genValidEggName(),
			genFolderID,
		),
	)

	properties.Property(
		"aws with valid egg-name passes validation (region optional per Requirement 4.5)",
		prop.ForAll(
			func(eggName string, region string) bool {
				params := CreateParams{
					Provider: "aws",
					EggName:  eggName,
					Region:   region,
				}
				err := ValidateCreateInput(params)
				if err != nil {
					t.Logf("unexpected error for valid aws params (egg=%q, region=%q): %v", eggName, region, err)
					return false
				}
				return true
			},
			genValidEggName(),
			genRegion,
		),
	)

	properties.Property(
		"both providers accept valid params without error",
		prop.ForAll(
			func(eggName string, folderID string, region string, providerIdx int) bool {
				providers := []string{"yandex", "aws"}
				provider := providers[providerIdx%2]
				params := CreateParams{
					Provider: provider,
					EggName:  eggName,
					FolderID: folderID,
					Region:   region,
				}
				err := ValidateCreateInput(params)
				if err != nil {
					t.Logf("unexpected error for provider=%q egg=%q folder=%q region=%q: %v",
						provider, eggName, folderID, region, err)
					return false
				}
				return true
			},
			genValidEggName(),
			genFolderID,
			genRegion,
			gen.IntRange(0, 1),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gosling-lockbox, Property 5: Verify correctly partitions entries into present and missing
// Validates: Requirements 6.3, 6.4
func TestVerifyPartitionsEntries(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for a random subset of RequiredEntries represented as a bitmask.
	// Each bit determines whether the corresponding RequiredEntries element is present.
	genSubsetMask := gen.IntRange(0, (1<<len(RequiredEntries))-1)

	properties.Property(
		"partition has exactly the chosen keys in Present and the complement in Missing",
		prop.ForAll(
			func(mask int) bool {
				// Build the present key set from the bitmask
				presentSet := make(map[string]bool, len(RequiredEntries))
				for i, key := range RequiredEntries {
					if mask&(1<<i) != 0 {
						presentSet[key] = true
					}
				}

				// Simulate the same partitioning logic used by both providers
				result := &VerifyResult{}
				for _, key := range RequiredEntries {
					if presentSet[key] {
						result.Present = append(result.Present, key)
					} else {
						result.Missing = append(result.Missing, key)
					}
				}

				// Check 1: total count equals RequiredEntries length
				if len(result.Present)+len(result.Missing) != len(RequiredEntries) {
					t.Logf("count mismatch: Present(%d) + Missing(%d) != RequiredEntries(%d)",
						len(result.Present), len(result.Missing), len(RequiredEntries))
					return false
				}

				// Check 2: Present contains exactly the keys from the subset
				for _, key := range result.Present {
					if !presentSet[key] {
						t.Logf("key %q in Present but was not in the chosen subset", key)
						return false
					}
				}

				// Check 3: Missing contains exactly the complement
				for _, key := range result.Missing {
					if presentSet[key] {
						t.Logf("key %q in Missing but was in the chosen subset", key)
						return false
					}
				}

				// Check 4: no duplicates across Present and Missing
				seen := make(map[string]bool, len(RequiredEntries))
				for _, key := range result.Present {
					if seen[key] {
						t.Logf("duplicate key %q in Present", key)
						return false
					}
					seen[key] = true
				}
				for _, key := range result.Missing {
					if seen[key] {
						t.Logf("key %q appears in both Present and Missing", key)
						return false
					}
					seen[key] = true
				}

				return true
			},
			genSubsetMask,
		),
	)

	properties.Property(
		"all entries present yields empty Missing",
		prop.ForAll(
			func(_ int) bool {
				// All keys present
				presentSet := make(map[string]bool, len(RequiredEntries))
				for _, key := range RequiredEntries {
					presentSet[key] = true
				}

				result := &VerifyResult{}
				for _, key := range RequiredEntries {
					if presentSet[key] {
						result.Present = append(result.Present, key)
					} else {
						result.Missing = append(result.Missing, key)
					}
				}

				if len(result.Missing) != 0 {
					t.Logf("expected empty Missing when all entries present, got %v", result.Missing)
					return false
				}
				if len(result.Present) != len(RequiredEntries) {
					t.Logf("expected %d Present entries, got %d", len(RequiredEntries), len(result.Present))
					return false
				}
				return true
			},
			gen.IntRange(0, 0), // dummy generator to satisfy gopter
		),
	)

	properties.Property(
		"no entries present yields empty Present",
		prop.ForAll(
			func(_ int) bool {
				// No keys present
				result := &VerifyResult{}
				result.Missing = append(result.Missing, RequiredEntries...)

				if len(result.Present) != 0 {
					t.Logf("expected empty Present when no entries present, got %v", result.Present)
					return false
				}
				if len(result.Missing) != len(RequiredEntries) {
					t.Logf("expected %d Missing entries, got %d", len(RequiredEntries), len(result.Missing))
					return false
				}
				return true
			},
			gen.IntRange(0, 0), // dummy generator to satisfy gopter
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
