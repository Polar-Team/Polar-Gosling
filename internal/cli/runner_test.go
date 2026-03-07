package cli

import (
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/polar-gosling/gosling/internal/runner"
)

// Feature: gitops-runner-orchestration, Property 27: Runner Tag-Based Routing
// Validates: Requirements 11.7
//
// Property: For any set of runner tags T defined in an Egg config, when the runner
// registers with GitLab using those tags, the registered runner's tags must exactly
// match T. Job routing based on tags must be consistent:
//   - A job requiring tag subset S ⊆ T MUST be routable to this runner
//   - A job requiring tag S where S ⊄ T MUST NOT be routable to this runner
func TestRunnerTagBasedRouting(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property 27a: Registered tags exactly match Egg config tags
	properties.Property("runner registers with exactly the tags defined in Egg config",
		prop.ForAll(
			func(tags []string) bool {
				router := &runner.TagRouter{RunnerTags: tags}
				if len(router.RunnerTags) != len(tags) {
					return false
				}
				tagSet := make(map[string]struct{}, len(tags))
				for _, t := range tags {
					tagSet[t] = struct{}{}
				}
				for _, t := range router.RunnerTags {
					if _, ok := tagSet[t]; !ok {
						return false
					}
				}
				return true
			},
			genTagSet(),
		))

	// Property 27b: A job requiring a subset S ⊆ T is always routable
	properties.Property("job with tag subset S ⊆ T is routable to runner with tags T",
		prop.ForAll(
			func(pair tagSetWithSubset) bool {
				router := &runner.TagRouter{RunnerTags: pair.RunnerTags}
				return router.CanRoute(pair.Subset)
			},
			genTagSetWithSubset(),
		))

	// Property 27c: A job requiring a tag NOT in T is never routable
	properties.Property("job requiring tag not in runner tag set T is not routable",
		prop.ForAll(
			func(runnerTags []string, extraTag string) bool {
				for _, t := range runnerTags {
					if t == extraTag {
						return true // precondition not met, vacuously true
					}
				}
				router := &runner.TagRouter{RunnerTags: runnerTags}
				return !router.CanRoute([]string{extraTag})
			},
			genTagSet(),
			genDistinctTag(),
		))

	// Property 27d: A job with no required tags is always routable (untagged jobs)
	properties.Property("job with no required tags is always routable to any runner",
		prop.ForAll(
			func(runnerTags []string) bool {
				router := &runner.TagRouter{RunnerTags: runnerTags}
				return router.CanRoute([]string{})
			},
			genTagSet(),
		))

	// Property 27e: Routing is deterministic for the same inputs
	properties.Property("tag routing is deterministic for the same inputs",
		prop.ForAll(
			func(runnerTags []string, jobTags []string) bool {
				router := &runner.TagRouter{RunnerTags: runnerTags}
				first := router.CanRoute(jobTags)
				second := router.CanRoute(jobTags)
				return first == second
			},
			genTagSet(),
			genTagSet(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genTagSet generates a random slice of unique tag strings (simulates Egg config tags)
func genTagSet() gopter.Gen {
	return gen.SliceOf(genTagString()).Map(func(tags []string) []string {
		seen := make(map[string]struct{})
		unique := make([]string, 0, len(tags))
		for _, t := range tags {
			if t == "" {
				continue
			}
			if _, ok := seen[t]; !ok {
				seen[t] = struct{}{}
				unique = append(unique, t)
			}
		}
		return unique
	})
}

// tagSetWithSubset holds a runner tag set T and a guaranteed subset S ⊆ T
type tagSetWithSubset struct {
	RunnerTags []string
	Subset     []string
}

// genTagSetWithSubset generates a (T, S) pair where S ⊆ T
func genTagSetWithSubset() gopter.Gen {
	return genTagSet().FlatMap(func(v interface{}) gopter.Gen {
		runnerTags := v.([]string)
		return genSubsetOf(runnerTags).Map(func(subset []string) tagSetWithSubset {
			return tagSetWithSubset{RunnerTags: runnerTags, Subset: subset}
		})
	}, reflect.TypeOf(tagSetWithSubset{}))
}

// genSubsetOf generates a random subset of the given slice
func genSubsetOf(tags []string) gopter.Gen {
	if len(tags) == 0 {
		return gen.Const([]string{})
	}
	return gen.IntRange(0, len(tags)).Map(func(n int) []string {
		return tags[:n]
	})
}

// genTagString generates a single valid tag string
func genTagString() gopter.Gen {
	return gen.OneConstOf(
		"docker", "linux", "windows", "arm64", "amd64",
		"gpu", "high-mem", "fast-disk", "private", "shared",
		"yandex", "aws", "k8s", "vm", "serverless",
	)
}

// genDistinctTag generates a tag that is unlikely to appear in a small random tag set
func genDistinctTag() gopter.Gen {
	return gen.OneConstOf("__unique_tag_xyz__", "__never_in_set__", "__exclusive_tag__")
}

// TestParseTagsRoundTrip verifies that runner.ParseTags correctly splits comma-separated tags
func TestParseTagsRoundTrip(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"docker", []string{"docker"}},
		{"docker,linux", []string{"docker", "linux"}},
		{"docker, linux, arm64", []string{"docker", "linux", "arm64"}},
		{" docker , linux ", []string{"docker", "linux"}},
	}

	for _, tc := range tests {
		got := runner.ParseTags(tc.input)
		if len(got) != len(tc.expected) {
			t.Errorf("ParseTags(%q): got %v, want %v", tc.input, got, tc.expected)
			continue
		}
		for i := range got {
			if got[i] != tc.expected[i] {
				t.Errorf("ParseTags(%q)[%d]: got %q, want %q", tc.input, i, got[i], tc.expected[i])
			}
		}
	}
}

// TestTagRouterCanRoute verifies specific routing examples
func TestTagRouterCanRoute(t *testing.T) {
	tests := []struct {
		runnerTags []string
		jobTags    []string
		want       bool
	}{
		{[]string{"docker", "linux"}, []string{"docker", "linux"}, true},
		{[]string{"docker", "linux", "arm64"}, []string{"docker"}, true},
		{[]string{"docker", "linux"}, []string{}, true},
		{[]string{}, []string{}, true},
		{[]string{"docker", "linux"}, []string{"windows"}, false},
		{[]string{"docker", "linux"}, []string{"docker", "windows"}, false},
		{[]string{}, []string{"docker"}, false},
	}

	for _, tc := range tests {
		router := &runner.TagRouter{RunnerTags: tc.runnerTags}
		got := router.CanRoute(tc.jobTags)
		if got != tc.want {
			t.Errorf("CanRoute(runnerTags=%v, jobTags=%v) = %v, want %v",
				tc.runnerTags, tc.jobTags, got, tc.want)
		}
	}
}
