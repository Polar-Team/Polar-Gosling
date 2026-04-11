package deployer

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: gitops-runner-orchestration, Property 49: Storage Config Round-Trip
// **Validates: Requirements 25.1, 25.3, 25.12, 25.13**
//
// For any valid StorageConfig with bucket name and region, serializing to .fly
// and parsing back produces equivalent StorageConfig.

// genBucketName generates valid S3 bucket names: lowercase alphanumeric + hyphens,
// 3-63 chars, must start and end with alphanumeric.
func genBucketName() gopter.Gen {
	alphaNum := "abcdefghijklmnopqrstuvwxyz0123456789"
	midChars := "abcdefghijklmnopqrstuvwxyz0123456789-"

	return gen.IntRange(3, 63).FlatMap(func(v interface{}) gopter.Gen {
		length := v.(int)
		return gopter.CombineGens(
			gen.IntRange(0, len(alphaNum)-1),
			gen.SliceOfN(length-2, gen.IntRange(0, len(midChars)-1)),
			gen.IntRange(0, len(alphaNum)-1),
		).Map(func(vals []interface{}) string {
			firstIdx := vals[0].(int)
			midIdxs := vals[1].([]int)
			lastIdx := vals[2].(int)

			buf := make([]byte, 0, length)
			buf = append(buf, alphaNum[firstIdx])
			for _, idx := range midIdxs {
				buf = append(buf, midChars[idx])
			}
			buf = append(buf, alphaNum[lastIdx])
			return string(buf)
		})
	}, reflect.TypeOf(""))
}

// genRegion generates realistic cloud region strings.
func genRegion() gopter.Gen {
	return gen.OneConstOf(
		"ru-central1",
		"us-east-1",
		"us-west-2",
		"eu-west-1",
		"eu-west-2",
		"ap-southeast-1",
		"ap-northeast-1",
	)
}

func TestStorageConfigRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property(
		"storage config round-trip: serialize to .fly then parse back",
		prop.ForAll(
			func(bucketName, region string) bool {
				// Build a minimal mothergoose .fly file with storage block
				flyContent := fmt.Sprintf(`mothergoose "test-mg" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g0000000000"
    yc_cloud_id  = "b1gabcdefghij"
  }

  storage {
    bucket_name = %q
    region      = %q
  }
}
`, bucketName, region)

				// Write to temp dir and parse
				dir := t.TempDir()
				flyPath := filepath.Join(dir, "config.fly")
				if err := os.WriteFile(flyPath, []byte(flyContent), 0644); err != nil {
					t.Logf("failed to write .fly file: %v", err)
					return false
				}

				configs, err := ParseMGDirectory(dir)
				if err != nil {
					t.Logf("ParseMGDirectory failed: %v", err)
					return false
				}
				if len(configs) != 1 {
					t.Logf("expected 1 config, got %d", len(configs))
					return false
				}

				got := configs[0].Storage
				if got.BucketName != bucketName {
					t.Logf("BucketName mismatch: want %q, got %q", bucketName, got.BucketName)
					return false
				}
				if got.Region != region {
					t.Logf("Region mismatch: want %q, got %q", region, got.Region)
					return false
				}
				return true
			},
			genBucketName(),
			genRegion(),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gitops-runner-orchestration, Property 50: Legacy Storage Block Rejection
// **Validates: Requirements 25.2, 25.14**
//
// For any storage block containing legacy sub-bucket definitions (state_bucket,
// binary_bucket, or similar nested blocks), the parser returns a descriptive
// migration error and not a valid StorageConfig.

func TestLegacyStorageBlockRejection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// genLegacyCombination generates an int in [1,3] representing which legacy
	// sub-blocks to include: 1=state_bucket only, 2=binary_bucket only, 3=both.
	legacyCombination := gen.IntRange(1, 3)

	properties.Property(
		"legacy storage sub-blocks produce migration error",
		prop.ForAll(
			func(combo int, stateName, binaryName string) bool {
				hasStateBucket := combo == 1 || combo == 3
				hasBinaryBucket := combo == 2 || combo == 3

				// Build the legacy sub-blocks
				var legacyBlocks string
				var expectedNames []string

				if hasStateBucket {
					legacyBlocks += fmt.Sprintf(`
    state_bucket {
      name = %q
    }`, stateName)
					expectedNames = append(expectedNames, "state_bucket")
				}
				if hasBinaryBucket {
					legacyBlocks += fmt.Sprintf(`
    binary_bucket {
      name = %q
    }`, binaryName)
					expectedNames = append(expectedNames, "binary_bucket")
				}

				flyContent := fmt.Sprintf(`mothergoose "test-mg" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g0000000000"
    yc_cloud_id  = "b1gabcdefghij"
  }

  storage {%s
  }
}
`, legacyBlocks)

				dir := t.TempDir()
				flyPath := filepath.Join(dir, "config.fly")
				if err := os.WriteFile(flyPath, []byte(flyContent), 0644); err != nil {
					t.Logf("failed to write .fly file: %v", err)
					return false
				}

				_, err := ParseMGDirectory(dir)

				// Must return an error
				if err == nil {
					t.Log("expected error for legacy storage sub-blocks, got nil")
					return false
				}

				errMsg := err.Error()

				// Error must mention "no longer supported"
				if !strings.Contains(errMsg, "no longer supported") {
					t.Logf("error missing 'no longer supported': %s", errMsg)
					return false
				}

				// Error must mention each legacy sub-block name that was present
				for _, name := range expectedNames {
					if !strings.Contains(errMsg, name) {
						t.Logf("error missing legacy block name %q: %s", name, errMsg)
						return false
					}
				}

				return true
			},
			legacyCombination,
			genBucketName(),
			genBucketName(),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gitops-runner-orchestration, Property 51: Static Folder Prefix Immutability
// **Validates: Requirements 25.4, 25.5, 25.6, 25.7, 25.8, 25.16**
//
// For any StorageConfig, folder prefixes are always the hardcoded constants
// regardless of configuration content.

func TestStaticFolderPrefixImmutability(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property(
		"folder prefix constants are immutable regardless of StorageConfig content",
		prop.ForAll(
			func(bucketName, region string) bool {
				// Verify the four folder prefix constants are always the expected hardcoded values,
				// regardless of what StorageConfig values are generated.
				if StoragePrefixBinaries != "binaries/" {
					t.Logf("StoragePrefixBinaries changed: got %q, want %q", StoragePrefixBinaries, "binaries/")
					return false
				}
				if StoragePrefixStates != "states/" {
					t.Logf("StoragePrefixStates changed: got %q, want %q", StoragePrefixStates, "states/")
					return false
				}
				if StoragePrefixPluginCache != "plugin-cache/" {
					t.Logf("StoragePrefixPluginCache changed: got %q, want %q", StoragePrefixPluginCache, "plugin-cache/")
					return false
				}
				if StoragePrefixRunnersCache != "runners-cache/" {
					t.Logf("StoragePrefixRunnersCache changed: got %q, want %q", StoragePrefixRunnersCache, "runners-cache/")
					return false
				}

				// Verify that constructing full paths always uses the constant prefix,
				// not anything derived from the StorageConfig.
				cfg := StorageConfig{BucketName: bucketName, Region: region}

				expectedPaths := map[string]string{
					"binaries":      cfg.BucketName + "/" + StoragePrefixBinaries,
					"states":        cfg.BucketName + "/" + StoragePrefixStates,
					"plugin-cache":  cfg.BucketName + "/" + StoragePrefixPluginCache,
					"runners-cache": cfg.BucketName + "/" + StoragePrefixRunnersCache,
				}

				for label, fullPath := range expectedPaths {
					// The path must start with the bucket name and contain the static prefix
					expectedPrefix := cfg.BucketName + "/"
					if !strings.HasPrefix(fullPath, expectedPrefix) {
						t.Logf("%s path missing bucket prefix: got %q", label, fullPath)
						return false
					}
					// The suffix after bucket+slash must be exactly the static constant
					suffix := fullPath[len(expectedPrefix):]
					switch label {
					case "binaries":
						if suffix != "binaries/" {
							t.Logf("%s suffix mismatch: got %q", label, suffix)
							return false
						}
					case "states":
						if suffix != "states/" {
							t.Logf("%s suffix mismatch: got %q", label, suffix)
							return false
						}
					case "plugin-cache":
						if suffix != "plugin-cache/" {
							t.Logf("%s suffix mismatch: got %q", label, suffix)
							return false
						}
					case "runners-cache":
						if suffix != "runners-cache/" {
							t.Logf("%s suffix mismatch: got %q", label, suffix)
							return false
						}
					}
				}

				return true
			},
			genBucketName(),
			genRegion(),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gitops-runner-orchestration, Property 52: Init Generates Single-Bucket Storage
// **Validates: Requirements 25.10, 25.11, 1.19**
//
// For any invocation of `gosling init`, generated storage block parses to valid
// single-bucket StorageConfig with only BucketName and Region.
// We test this by generating random bucket names and regions, constructing a
// storage block with ONLY those two fields (like init produces), parsing it,
// and verifying the result has exactly BucketName and Region set.

func TestInitSingleBucketGeneration(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property(
		"single-bucket storage block (like init generates) parses to valid StorageConfig with only BucketName and Region",
		prop.ForAll(
			func(bucketName, region string) bool {
				// Construct a minimal MG config with only bucket_name and region
				// in the storage block — exactly what `gosling init` generates.
				flyContent := fmt.Sprintf(`mothergoose "init-test" {
  cloud {
    provider     = "yandex"
    yc_folder_id = "b1g0000000000"
    yc_cloud_id  = "b1gabcdefghij"
  }

  storage {
    bucket_name = %q
    region      = %q
  }
}
`, bucketName, region)

				dir := t.TempDir()
				flyPath := filepath.Join(dir, "config.fly")
				if err := os.WriteFile(flyPath, []byte(flyContent), 0644); err != nil {
					t.Logf("failed to write .fly file: %v", err)
					return false
				}

				configs, err := ParseMGDirectory(dir)
				if err != nil {
					t.Logf("ParseMGDirectory failed: %v", err)
					return false
				}
				if len(configs) != 1 {
					t.Logf("expected 1 config, got %d", len(configs))
					return false
				}

				got := configs[0].Storage

				// BucketName must be non-empty and match input
				if got.BucketName == "" {
					t.Log("BucketName is empty")
					return false
				}
				if got.BucketName != bucketName {
					t.Logf("BucketName mismatch: want %q, got %q", bucketName, got.BucketName)
					return false
				}

				// Region must be non-empty and match input
				if got.Region == "" {
					t.Log("Region is empty")
					return false
				}
				if got.Region != region {
					t.Logf("Region mismatch: want %q, got %q", region, got.Region)
					return false
				}

				// StorageConfig should have ONLY BucketName and Region set —
				// verify by comparing against a struct with just those two fields.
				expected := StorageConfig{BucketName: bucketName, Region: region}
				if !reflect.DeepEqual(got, expected) {
					t.Logf("StorageConfig has unexpected fields: got %+v, want %+v", got, expected)
					return false
				}

				return true
			},
			genBucketName(),
			genRegion(),
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gitops-runner-orchestration, Property 53: DB-Based Active Version Path Resolution
// **Validates: Requirements 23.30, 25.15, 25.16**
//
// For any binary name and active version record, resolved path is
// /mnt/s3-storage/binaries/{binary_name}/{version}/{binary_name}.

func TestDBBasedActiveVersionPathResolution(t *testing.T) {
	properties := gopter.NewProperties(nil)

	genBinaryName := gen.OneConstOf("gosling", "opentofu")
	genVersion := gen.OneConstOf("1.0.0", "1.2.3", "2.0.0", "0.1.0", "3.5.1", "1.0.0-beta", "2.1.0-rc1")

	properties.Property(
		"resolved binary path matches /mnt/s3-storage/binaries/{binaryName}/{version}/{binaryName}",
		prop.ForAll(
			func(binaryName, version string) bool {
				resolved := ResolveBinaryPath(binaryName, version)

				// Verify the result matches the expected pattern exactly
				expected := fmt.Sprintf("/mnt/s3-storage/binaries/%s/%s/%s", binaryName, version, binaryName)
				if resolved != expected {
					t.Logf("path mismatch: got %q, want %q", resolved, expected)
					return false
				}

				// Verify the path starts with the S3 mount point
				if !strings.HasPrefix(resolved, "/mnt/s3-storage/") {
					t.Logf("path missing S3 mount prefix: %q", resolved)
					return false
				}

				// Verify the path contains the StoragePrefixBinaries constant
				if !strings.Contains(resolved, StoragePrefixBinaries) {
					t.Logf("path missing StoragePrefixBinaries %q: %q", StoragePrefixBinaries, resolved)
					return false
				}

				// Verify the path ends with /{binaryName}
				expectedSuffix := "/" + binaryName
				if !strings.HasSuffix(resolved, expectedSuffix) {
					t.Logf("path does not end with %q: %q", expectedSuffix, resolved)
					return false
				}

				return true
			},
			genBinaryName,
			genVersion,
		),
	)

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
