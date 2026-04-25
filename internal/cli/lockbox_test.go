package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// executeCommand runs rootCmd with the given args and captures stdout/stderr.
// Returns stdout, stderr, and any error from execution.
func executeCommand(args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(args)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

// resetCreateFlags resets the package-level flag variables and Cobra flag state
// for lockbox create to avoid state leaking between tests.
func resetCreateFlags() {
	createProvider = ""
	createEggName = ""
	createFolderID = ""
	createRegion = ""
	lockboxCreateCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		f.Value.Set(f.DefValue)
	})
}

// resetListFlags resets the package-level flag variables and Cobra flag state for lockbox list.
func resetListFlags() {
	listProvider = ""
	listFolderID = ""
	listRegion = ""
	lockboxListCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		f.Value.Set(f.DefValue)
	})
}

// resetVerifyFlags resets the package-level flag variables and Cobra flag state for lockbox verify.
func resetVerifyFlags() {
	verifyProvider = ""
	verifySecretID = ""
	verifySecretName = ""
	verifyFolderID = ""
	verifyRegion = ""
	lockboxVerifyCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		f.Value.Set(f.DefValue)
	})
}

// TestLockboxCommandRegistration verifies that the lockbox command is registered
// as a subcommand of rootCmd with create, list, and verify children.
// Requirements: 1.1, 1.2, 1.3
func TestLockboxCommandRegistration(t *testing.T) {
	// lockbox should be a direct child of rootCmd
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "lockbox" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("lockbox command not registered as subcommand of root")
	}

	// Verify lockbox has the expected short description
	if lockboxCmd.Short != "Manage cloud secret stores for Egg configurations" {
		t.Errorf("unexpected lockbox short description: %q", lockboxCmd.Short)
	}

	// Verify Use field
	if lockboxCmd.Use != "lockbox" {
		t.Errorf("unexpected lockbox Use: %q", lockboxCmd.Use)
	}
}

// TestLockboxSubcommandRegistration verifies create, list, verify are children of lockbox.
// Requirements: 1.1, 1.3
func TestLockboxSubcommandRegistration(t *testing.T) {
	expected := map[string]bool{
		"create": false,
		"list":   false,
		"verify": false,
	}

	for _, cmd := range lockboxCmd.Commands() {
		if _, ok := expected[cmd.Use]; ok {
			expected[cmd.Use] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("subcommand %q not registered under lockbox", name)
		}
	}
}

// TestLockboxHelpOutput verifies that running `gosling lockbox --help`
// shows usage help listing all available subcommands.
// Requirements: 1.2, 1.3
func TestLockboxHelpOutput(t *testing.T) {
	stdout, _, _ := executeCommand("lockbox", "--help")

	// Should mention all three subcommands
	for _, sub := range []string{"create", "list", "verify"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("lockbox help output missing subcommand %q\nOutput:\n%s", sub, stdout)
		}
	}

	// Should contain the short description
	if !strings.Contains(stdout, "Manage cloud secret stores for Egg configurations") {
		t.Errorf("lockbox help output missing short description\nOutput:\n%s", stdout)
	}
}

// TestLockboxCreateHelpOutput verifies that `gosling lockbox create --help` shows flag info.
// Requirements: 8.1
func TestLockboxCreateHelpOutput(t *testing.T) {
	stdout, _, _ := executeCommand("lockbox", "create", "--help")

	for _, flag := range []string{"--provider", "--egg-name", "--folder-id", "--region"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("lockbox create help missing flag %q\nOutput:\n%s", flag, stdout)
		}
	}
}

// TestLockboxCreateMissingProviderFlag verifies that omitting --provider produces an error.
// Requirements: 8.1, 8.2
func TestLockboxCreateMissingProviderFlag(t *testing.T) {
	resetCreateFlags()
	_, stderr, err := executeCommand("lockbox", "create", "--egg-name", "my-app")
	if err == nil {
		t.Fatal("expected error for missing --provider flag, got nil")
	}

	combined := strings.ToLower(err.Error() + " " + stderr)
	if !strings.Contains(combined, "provider") {
		t.Errorf("expected error to mention 'provider', got err=%q stderr=%q", err.Error(), stderr)
	}
}

// TestLockboxCreateMissingEggNameFlag verifies that omitting --egg-name produces an error.
// Requirements: 8.1, 8.2
func TestLockboxCreateMissingEggNameFlag(t *testing.T) {
	resetCreateFlags()
	// Use an invalid provider to ensure we hit validation before cloud SDK
	_, stderr, err := executeCommand("lockbox", "create", "--provider", "gcp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	combined := strings.ToLower(err.Error() + " " + stderr)
	// Should mention either egg-name (Cobra required flag) or invalid provider (validation)
	if !strings.Contains(combined, "egg-name") && !strings.Contains(combined, "invalid provider") {
		t.Errorf("expected error to mention 'egg-name' or 'invalid provider', got err=%q stderr=%q", err.Error(), stderr)
	}
}

// TestLockboxCreateNoFlags verifies that omitting all required flags produces an error.
// Requirements: 8.1, 8.2
func TestLockboxCreateNoFlags(t *testing.T) {
	resetCreateFlags()
	_, stderr, err := executeCommand("lockbox", "create")
	if err == nil {
		t.Fatal("expected error for missing required flags, got nil")
	}

	combined := strings.ToLower(err.Error() + " " + stderr)
	// Should mention at least one of the required flags
	if !strings.Contains(combined, "required") && !strings.Contains(combined, "provider") {
		t.Errorf("expected error to mention 'required' or 'provider', got err=%q stderr=%q", err.Error(), stderr)
	}
}

// TestLockboxListMissingProviderFlag verifies that omitting --provider produces an error.
// Requirements: 8.1
func TestLockboxListMissingProviderFlag(t *testing.T) {
	resetListFlags()
	_, stderr, err := executeCommand("lockbox", "list")
	if err == nil {
		t.Fatal("expected error for missing --provider flag, got nil")
	}

	combined := strings.ToLower(err.Error() + " " + stderr)
	if !strings.Contains(combined, "provider") {
		t.Errorf("expected error to mention 'provider', got err=%q stderr=%q", err.Error(), stderr)
	}
}

// TestLockboxVerifyMissingProviderFlag verifies that omitting --provider produces an error.
// Requirements: 8.1
func TestLockboxVerifyMissingProviderFlag(t *testing.T) {
	resetVerifyFlags()
	_, stderr, err := executeCommand("lockbox", "verify")
	if err == nil {
		t.Fatal("expected error for missing --provider flag, got nil")
	}

	combined := strings.ToLower(err.Error() + " " + stderr)
	if !strings.Contains(combined, "provider") {
		t.Errorf("expected error to mention 'provider', got err=%q stderr=%q", err.Error(), stderr)
	}
}

// TestLockboxCreateInvalidProvider verifies that an invalid provider value produces an error.
// This tests the validation layer (ValidateCreateInput) before any cloud API call.
// Requirements: 8.1, 8.2
func TestLockboxCreateInvalidProvider(t *testing.T) {
	resetCreateFlags()
	_, _, err := executeCommand("lockbox", "create", "--provider", "gcp", "--egg-name", "my-app")
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}

	if !strings.Contains(err.Error(), "invalid provider") {
		t.Errorf("expected 'invalid provider' in error, got: %s", err.Error())
	}
}

// TestLockboxListInvalidProvider verifies that an invalid provider value produces an error.
// Requirements: 8.1
func TestLockboxListInvalidProvider(t *testing.T) {
	resetListFlags()
	_, _, err := executeCommand("lockbox", "list", "--provider", "gcp")
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}

	if !strings.Contains(err.Error(), "invalid provider") {
		t.Errorf("expected 'invalid provider' in error, got: %s", err.Error())
	}
}

// TestLockboxVerifyInvalidProvider verifies that an invalid provider value produces an error.
// Requirements: 8.1
func TestLockboxVerifyInvalidProvider(t *testing.T) {
	resetVerifyFlags()
	_, _, err := executeCommand("lockbox", "verify", "--provider", "azure")
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}

	if !strings.Contains(err.Error(), "invalid provider") {
		t.Errorf("expected 'invalid provider' in error, got: %s", err.Error())
	}
}

// TestLockboxListMissingFolderIDForYandex verifies that yandex provider requires folder-id.
// Requirements: 8.1
func TestLockboxListMissingFolderIDForYandex(t *testing.T) {
	resetListFlags()
	_, _, err := executeCommand("lockbox", "list", "--provider", "yandex")
	if err == nil {
		t.Fatal("expected error for missing folder-id, got nil")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "folder-id") {
		t.Errorf("expected error to mention 'folder-id', got: %s", err.Error())
	}
}

// TestLockboxVerifyMissingSecretRef verifies that provider-specific secret reference
// flags are validated (secret-id for yandex, secret-name for aws).
// Requirements: 8.1
func TestLockboxVerifyMissingSecretRef(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectInErr string
	}{
		{
			name:        "yandex missing secret-id",
			args:        []string{"lockbox", "verify", "--provider", "yandex", "--folder-id", "abc123"},
			expectInErr: "secret-id",
		},
		{
			name:        "aws missing secret-name",
			args:        []string{"lockbox", "verify", "--provider", "aws"},
			expectInErr: "secret-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetVerifyFlags()
			_, _, err := executeCommand(tt.args...)
			if err == nil {
				t.Fatal("expected error for missing secret reference, got nil")
			}

			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.expectInErr)) {
				t.Errorf("expected error to mention %q, got: %s", tt.expectInErr, err.Error())
			}
		})
	}
}

// TestLockboxCommandErrorsNotInStdout verifies that error output does not leak to stdout.
// Requirements: 1.3, 8.2
func TestLockboxCommandErrorsNotInStdout(t *testing.T) {
	resetCreateFlags()
	// Use an invalid provider to trigger a validation error (no cloud SDK call)
	stdout, _, err := executeCommand("lockbox", "create", "--provider", "gcp", "--egg-name", "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The validation error should not appear in stdout
	if strings.Contains(stdout, "invalid provider") {
		t.Errorf("validation error appeared in stdout: %s", stdout)
	}
}

// TestLockboxCreateFlagDefinitions verifies that all expected flags are defined on the create command.
// Requirements: 8.1
func TestLockboxCreateFlagDefinitions(t *testing.T) {
	expectedFlags := []string{"provider", "egg-name", "folder-id", "region"}
	for _, flag := range expectedFlags {
		if lockboxCreateCmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag %q not defined on lockbox create command", flag)
		}
	}
}

// TestLockboxListFlagDefinitions verifies that all expected flags are defined on the list command.
// Requirements: 8.1
func TestLockboxListFlagDefinitions(t *testing.T) {
	expectedFlags := []string{"provider", "folder-id", "region"}
	for _, flag := range expectedFlags {
		if lockboxListCmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag %q not defined on lockbox list command", flag)
		}
	}
}

// TestLockboxVerifyFlagDefinitions verifies that all expected flags are defined on the verify command.
// Requirements: 8.1
func TestLockboxVerifyFlagDefinitions(t *testing.T) {
	expectedFlags := []string{"provider", "secret-id", "secret-name", "folder-id", "region"}
	for _, flag := range expectedFlags {
		if lockboxVerifyCmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag %q not defined on lockbox verify command", flag)
		}
	}
}

// TestLockboxCreateRequiredFlagsMarked verifies that provider and egg-name are marked as required.
// Requirements: 8.1
func TestLockboxCreateRequiredFlagsMarked(t *testing.T) {
	for _, flag := range []string{"provider", "egg-name"} {
		f := lockboxCreateCmd.Flags().Lookup(flag)
		if f == nil {
			t.Fatalf("flag %q not found", flag)
		}
		ann := f.Annotations
		if ann == nil {
			t.Errorf("flag %q has no annotations (not marked required)", flag)
			continue
		}
		if _, ok := ann["cobra_annotation_bash_completion_one_required_flag"]; !ok {
			t.Errorf("flag %q not marked as required", flag)
		}
	}
}

// TestLockboxListRequiredFlagsMarked verifies that provider is marked as required.
// Requirements: 8.1
func TestLockboxListRequiredFlagsMarked(t *testing.T) {
	f := lockboxListCmd.Flags().Lookup("provider")
	if f == nil {
		t.Fatal("flag 'provider' not found")
	}
	ann := f.Annotations
	if ann == nil {
		t.Error("flag 'provider' has no annotations (not marked required)")
		return
	}
	if _, ok := ann["cobra_annotation_bash_completion_one_required_flag"]; !ok {
		t.Error("flag 'provider' not marked as required")
	}
}

// TestLockboxVerifyRequiredFlagsMarked verifies that provider is marked as required.
// Requirements: 8.1
func TestLockboxVerifyRequiredFlagsMarked(t *testing.T) {
	f := lockboxVerifyCmd.Flags().Lookup("provider")
	if f == nil {
		t.Fatal("flag 'provider' not found")
	}
	ann := f.Annotations
	if ann == nil {
		t.Error("flag 'provider' has no annotations (not marked required)")
		return
	}
	if _, ok := ann["cobra_annotation_bash_completion_one_required_flag"]; !ok {
		t.Error("flag 'provider' not marked as required")
	}
}
