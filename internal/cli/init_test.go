package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// saveAndRestoreRemoteFlags saves the current package-level remote flag values
// and the isTerminal function, then returns a cleanup function that restores them.
// This prevents test pollution across test cases that modify shared state.
func saveAndRestoreRemoteFlags(t *testing.T) {
	t.Helper()
	origRemoteName := remoteName
	origRemoteURL := remoteURL
	origBranchName := branchName
	origIsTerminal := isTerminal
	t.Cleanup(func() {
		remoteName = origRemoteName
		remoteURL = origRemoteURL
		branchName = origBranchName
		isTerminal = origIsTerminal
	})
}

// extractBlock extracts the content of a named top-level block from a .fly template
// string. For example, extractBlock(tpl, "fastapi_app") returns the text between
// "fastapi_app {" and its matching closing brace. Returns empty string if not found.
func extractBlock(template, blockName string) string {
	// Find the block opening — handles both "block {" and "block \"label\" {"
	idx := strings.Index(template, blockName)
	if idx == -1 {
		return ""
	}
	// Find the opening brace after the block name
	braceStart := strings.Index(template[idx:], "{")
	if braceStart == -1 {
		return ""
	}
	braceStart += idx

	// Walk forward counting braces to find the matching close
	depth := 0
	for i := braceStart; i < len(template); i++ {
		switch template[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return template[braceStart+1 : i]
			}
		}
	}
	return ""
}

// initGitRepoForTest creates a temp directory with an initialized git repo and
// returns the path. The directory is cleaned up when the test finishes.
func initGitRepoForTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	return dir
}

// captureStdout redirects os.Stdout to a buffer for the duration of fn,
// then restores it and returns whatever was written. Defers are used so
// stdout is restored and the pipe is closed even if fn panics.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = origStdout
	}()

	fn()

	// Close writer before reading so the reader sees EOF.
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read from pipe: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("failed to close pipe reader: %v", err)
	}
	return buf.String()
}

// TestFlagSkipsPrompts verifies that when --remote-url is provided via flag,
// configureUpstreamRemote uses the flag values directly and does not read from
// stdin (i.e., interactive prompts are bypassed).
// Requirements: 3.4, 3.5
func TestFlagSkipsPrompts(t *testing.T) {
	saveAndRestoreRemoteFlags(t)
	dir := initGitRepoForTest(t)

	// Set flag values as if the user passed --remote-url, --remote-name, --branch.
	remoteName = "upstream"
	remoteURL = "https://example.com/repo.git"
	branchName = "develop"

	// Replace stdin with a closed pipe so any attempt to read would fail/return EOF.
	// If the function tried to prompt, it would get no input — but it shouldn't prompt at all.
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		if err := r.Close(); err != nil {
			t.Fatalf("failed to close pipe reader: %v", err)
		}
	})

	output := captureStdout(t, func() {
		if _, err := configureUpstreamRemote(dir); err != nil {
			t.Fatalf("configureUpstreamRemote returned error: %v", err)
		}
	})

	// Verify the remote was actually added with the flag values.
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = dir
	remoteOut, err := cmd.Output()
	if err != nil {
		t.Fatalf("git remote -v failed: %v", err)
	}

	remoteStr := string(remoteOut)
	if !strings.Contains(remoteStr, "upstream") {
		t.Errorf("expected remote name 'upstream' in git remote -v output, got:\n%s", remoteStr)
	}
	if !strings.Contains(remoteStr, "https://example.com/repo.git") {
		t.Errorf("expected remote URL in git remote -v output, got:\n%s", remoteStr)
	}

	// Verify the confirmation message includes the flag values.
	if !strings.Contains(output, "upstream") {
		t.Errorf("expected 'upstream' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "https://example.com/repo.git") {
		t.Errorf("expected remote URL in output, got:\n%s", output)
	}
	if !strings.Contains(output, "develop") {
		t.Errorf("expected branch 'develop' in output, got:\n%s", output)
	}
}

// TestEmptyURLSkipsRemoteConfig verifies that when the user is in interactive mode
// and provides an empty URL, configureUpstreamRemote skips remote configuration
// and prints a message indicating no remote was configured.
// Requirements: 2.7
func TestEmptyURLSkipsRemoteConfig(t *testing.T) {
	saveAndRestoreRemoteFlags(t)
	dir := initGitRepoForTest(t)

	// No flag-based URL — force the interactive path.
	remoteURL = ""

	// Override isTerminal to return true so configureUpstreamRemote enters the
	// interactive branch. (Restored by saveAndRestoreRemoteFlags cleanup.)
	isTerminal = func() bool { return true }

	// Provide stdin with two empty lines: one for remote name (accept default)
	// and one for URL (empty → triggers skip).
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	if _, err := w.WriteString("\n\n"); err != nil {
		t.Fatalf("failed to write to pipe: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		if err := r.Close(); err != nil {
			t.Fatalf("failed to close pipe reader: %v", err)
		}
	})

	output := captureStdout(t, func() {
		if _, err := configureUpstreamRemote(dir); err != nil {
			t.Fatalf("configureUpstreamRemote returned error: %v", err)
		}
	})

	// Verify the skip message was printed.
	if !strings.Contains(output, "No remote URL provided") {
		t.Errorf("expected skip message about no remote URL, got:\n%s", output)
	}
	if !strings.Contains(output, "skipping upstream remote configuration") {
		t.Errorf("expected 'skipping upstream remote configuration' in output, got:\n%s", output)
	}

	// Verify no remote was added to the repo.
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = dir
	remoteOut, err := cmd.Output()
	if err != nil {
		t.Fatalf("git remote -v failed: %v", err)
	}
	if strings.TrimSpace(string(remoteOut)) != "" {
		t.Errorf("expected no remotes, but got:\n%s", string(remoteOut))
	}
}

// TestGitRemoteAddFailure verifies that when git remote add fails (e.g., duplicate
// remote name), the error is properly wrapped with the "failed to add upstream remote"
// prefix and includes the underlying git error.
// Requirements: 2.8
func TestGitRemoteAddFailure(t *testing.T) {
	dir := initGitRepoForTest(t)

	// Add a remote first so the second add with the same name fails.
	cmd := exec.Command("git", "remote", "add", "origin", "https://example.com/first.git")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("initial git remote add failed: %v\n%s", err, out)
	}

	// Now try to add a remote with the same name — this should fail.
	err := addGitRemote(dir, "origin", "https://example.com/second.git")
	if err == nil {
		t.Fatal("expected error when adding duplicate remote, got nil")
	}

	// Verify the error message wraps the failure correctly.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to add upstream remote") {
		t.Errorf("expected error to contain 'failed to add upstream remote', got: %s", errMsg)
	}

	// Also test addGitRemote on a directory that is not a git repo at all.
	nonGitDir := t.TempDir()
	err = addGitRemote(nonGitDir, "origin", "https://example.com/repo.git")
	if err == nil {
		t.Fatal("expected error when adding remote to non-git directory, got nil")
	}
	if !strings.Contains(err.Error(), "failed to add upstream remote") {
		t.Errorf("expected wrapped error for non-git dir, got: %s", err.Error())
	}
}

// TestNonTerminalSkipsPrompts verifies that when stdin is not a terminal and
// --remote-url is not provided, configureUpstreamRemote skips remote configuration
// silently without error and without adding any remote.
// Requirements: 3.6
func TestNonTerminalSkipsPrompts(t *testing.T) {
	saveAndRestoreRemoteFlags(t)
	dir := initGitRepoForTest(t)

	// No flag-based URL.
	remoteURL = ""

	// Replace stdin with a pipe (not a terminal device), so isTerminal() returns false.
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	// close writer — pipe is not a terminal
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}

	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		if err := r.Close(); err != nil {
			t.Fatalf("failed to close pipe reader: %v", err)
		}
	})

	output := captureStdout(t, func() {
		if _, err := configureUpstreamRemote(dir); err != nil {
			t.Fatalf("configureUpstreamRemote returned error: %v", err)
		}
	})

	// In non-terminal mode with no flags, the function should return nil silently.
	// No skip message, no confirmation — just nothing.
	if strings.Contains(output, "Configured upstream remote") || strings.Contains(output, "No remote URL provided") {
		t.Errorf("expected no remote-configuration output in non-terminal mode, got:\n%s", output)
	}

	// Verify no remote was added.
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = dir
	remoteOut, err := cmd.Output()
	if err != nil {
		t.Fatalf("git remote -v failed: %v", err)
	}
	if strings.TrimSpace(string(remoteOut)) != "" {
		t.Errorf("expected no remotes in non-terminal mode, but got:\n%s", string(remoteOut))
	}
}

// TestInitGitRepoFailure verifies that initGitRepo returns a properly wrapped error
// when called on an invalid path.
// Requirements: 1.3
func TestInitGitRepoFailure(t *testing.T) {
	// Use a path that doesn't exist — git init should fail.
	nonExistentDir := filepath.Join(t.TempDir(), "does", "not", "exist")

	err := initGitRepo(nonExistentDir)
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
	if !strings.Contains(err.Error(), "failed to initialize git repository") {
		t.Errorf("expected wrapped error message, got: %s", err.Error())
	}
}

// --- UglyFox template smoke tests ---

// TestUglyFoxTemplateHasBlockLabel verifies that the UglyFox template includes
// a labeled uglyfox block as required by the .fly language specification.
// Requirements: 4.1
func TestUglyFoxTemplateHasBlockLabel(t *testing.T) {
	output := defaultUglyFoxConfig()
	if !strings.Contains(output, `uglyfox "default" {`) {
		t.Errorf("expected UglyFox template to contain 'uglyfox \"default\" {', got:\n%s", output)
	}
}

// TestUglyFoxTemplateHasMothergooseAttr verifies that the UglyFox template includes
// a mothergoose attribute referencing a MotherGoose instance name.
// Requirements: 4.2
func TestUglyFoxTemplateHasMothergooseAttr(t *testing.T) {
	output := defaultUglyFoxConfig()
	if !strings.Contains(output, `mothergoose = "default"`) {
		t.Errorf("expected UglyFox template to contain 'mothergoose = \"default\"', got:\n%s", output)
	}
}

// TestUglyFoxTemplateHasThresholds verifies that the UglyFox template includes
// cpu_threshold and memory_threshold attributes in the apex block.
// Requirements: 4.3
func TestUglyFoxTemplateHasThresholds(t *testing.T) {
	output := defaultUglyFoxConfig()
	apexBlock := extractBlock(output, "apex")
	if apexBlock == "" {
		t.Fatalf("expected UglyFox template to contain an 'apex' block, got:\n%s", output)
	}
	if !strings.Contains(apexBlock, "cpu_threshold") {
		t.Errorf("expected 'cpu_threshold' inside apex block, got:\n%s", apexBlock)
	}
	if !strings.Contains(apexBlock, "memory_threshold") {
		t.Errorf("expected 'memory_threshold' inside apex block, got:\n%s", apexBlock)
	}
}

// TestUglyFoxTemplateHasPolicies verifies that the UglyFox template includes
// a policies block with at least one rule sub-block.
// Requirements: 4.4
func TestUglyFoxTemplateHasPolicies(t *testing.T) {
	output := defaultUglyFoxConfig()
	if !strings.Contains(output, "policies {") {
		t.Errorf("expected UglyFox template to contain 'policies {', got:\n%s", output)
	}
	if !strings.Contains(output, "rule ") {
		t.Errorf("expected UglyFox template to contain a 'rule' sub-block, got:\n%s", output)
	}
}

// --- MotherGoose template smoke tests ---

// TestMotherGooseTemplateHasBlockLabel verifies that the MotherGoose template
// includes a labeled mothergoose block as required by the .fly language specification.
// Requirements: 5.1
func TestMotherGooseTemplateHasBlockLabel(t *testing.T) {
	output := defaultMotherGooseConfig()
	if !strings.Contains(output, `mothergoose "default" {`) {
		t.Errorf("expected MotherGoose template to contain 'mothergoose \"default\" {', got:\n%s", output)
	}
}

// TestMotherGooseTemplateHasCloudBlock verifies that the MotherGoose template
// includes a cloud block with provider, yc_folder_id, and yc_cloud_id attributes.
// Requirements: 5.2
func TestMotherGooseTemplateHasCloudBlock(t *testing.T) {
	output := defaultMotherGooseConfig()
	cloudBlock := extractBlock(output, "cloud")
	if cloudBlock == "" {
		t.Fatalf("expected MotherGoose template to contain a 'cloud' block, got:\n%s", output)
	}
	for _, attr := range []string{"provider", "yc_folder_id", "yc_cloud_id"} {
		if !strings.Contains(cloudBlock, attr) {
			t.Errorf("expected '%s' inside cloud block, got:\n%s", attr, cloudBlock)
		}
	}
}

// TestMotherGooseTemplateHasFastapiAttrs verifies that the MotherGoose template
// includes image, cores, execution_timeout, concurrency, and service_account
// attributes in the fastapi_app block.
// Requirements: 5.3
func TestMotherGooseTemplateHasFastapiAttrs(t *testing.T) {
	output := defaultMotherGooseConfig()
	fastapiBlock := extractBlock(output, "fastapi_app")
	if fastapiBlock == "" {
		t.Fatalf("expected MotherGoose template to contain a 'fastapi_app' block, got:\n%s", output)
	}
	for _, attr := range []string{"image", "cores", "execution_timeout", "concurrency", "service_account"} {
		if !strings.Contains(fastapiBlock, attr) {
			t.Errorf("expected '%s' inside fastapi_app block, got:\n%s", attr, fastapiBlock)
		}
	}
}

// TestMotherGooseTemplateHasCeleryAttrs verifies that the MotherGoose template
// includes cores, execution_timeout, and concurrency attributes in the
// celery_workers block.
// Requirements: 5.4
func TestMotherGooseTemplateHasCeleryAttrs(t *testing.T) {
	output := defaultMotherGooseConfig()
	celeryBlock := extractBlock(output, "celery_workers")
	if celeryBlock == "" {
		t.Fatalf("expected MotherGoose template to contain a 'celery_workers' block, got:\n%s", output)
	}
	for _, attr := range []string{"cores", "execution_timeout", "concurrency"} {
		if !strings.Contains(celeryBlock, attr) {
			t.Errorf("expected '%s' inside celery_workers block, got:\n%s", attr, celeryBlock)
		}
	}
}

// TestMotherGooseTemplateUsesGitSyncTrigger verifies that the MotherGoose template
// uses git_sync_trigger as the trigger block name instead of the old triggers block.
// Requirements: 5.5
func TestMotherGooseTemplateUsesGitSyncTrigger(t *testing.T) {
	output := defaultMotherGooseConfig()
	if !strings.Contains(output, "git_sync_trigger {") {
		t.Errorf("expected MotherGoose template to contain 'git_sync_trigger {', got:\n%s", output)
	}
	if strings.Contains(output, "triggers {") {
		t.Errorf("expected MotherGoose template NOT to contain old 'triggers {' block, got:\n%s", output)
	}
}

// TestMotherGooseTemplateUsesMothergooseQueues verifies that the MotherGoose template
// uses mothergoose_queues as the queue block name with task_queue and dlq sub-blocks
// instead of the old message_queues block.
// Requirements: 5.6
func TestMotherGooseTemplateUsesMothergooseQueues(t *testing.T) {
	output := defaultMotherGooseConfig()
	if !strings.Contains(output, "mothergoose_queues {") {
		t.Errorf("expected MotherGoose template to contain 'mothergoose_queues {', got:\n%s", output)
	}
	if !strings.Contains(output, "task_queue {") {
		t.Errorf("expected MotherGoose template to contain 'task_queue {' sub-block, got:\n%s", output)
	}
	if !strings.Contains(output, "dlq {") {
		t.Errorf("expected MotherGoose template to contain 'dlq {' sub-block, got:\n%s", output)
	}
	if strings.Contains(output, "message_queues {") {
		t.Errorf("expected MotherGoose template NOT to contain old 'message_queues {' block, got:\n%s", output)
	}
}

// TestMotherGooseTemplateUsesLabeledServiceAccounts verifies that the MotherGoose
// template uses labeled service_account blocks (e.g., service_account "mg-sa" {})
// instead of a service_accounts wrapper block.
// Requirements: 5.7
func TestMotherGooseTemplateUsesLabeledServiceAccounts(t *testing.T) {
	output := defaultMotherGooseConfig()
	if !strings.Contains(output, `service_account "mg-sa"`) {
		t.Errorf("expected MotherGoose template to contain 'service_account \"mg-sa\"', got:\n%s", output)
	}
	if strings.Contains(output, "service_accounts {") {
		t.Errorf("expected MotherGoose template NOT to contain old 'service_accounts {' wrapper, got:\n%s", output)
	}
}

// TestMotherGooseTemplateUsesServerlessMode verifies that the MotherGoose template
// uses serverless_mode = true in the database block instead of the old mode = "serverless".
// Requirements: 5.8
func TestMotherGooseTemplateUsesServerlessMode(t *testing.T) {
	output := defaultMotherGooseConfig()
	if !strings.Contains(output, "serverless_mode = true") {
		t.Errorf("expected MotherGoose template to contain 'serverless_mode = true', got:\n%s", output)
	}
	if strings.Contains(output, `mode = "serverless"`) {
		t.Errorf("expected MotherGoose template NOT to contain old 'mode = \"serverless\"', got:\n%s", output)
	}
}

// --- Success output tests ---

// TestSuccessOutputWithRemote verifies that when runInit completes with an upstream
// remote configured (via flags), the success output includes the remote name and URL,
// and suggests `git push -u <remote_name> <branch_name>` as a next step.
// Requirements: 6.1, 6.2
func TestSuccessOutputWithRemote(t *testing.T) {
	saveAndRestoreRemoteFlags(t)

	// Save and restore initPath since runInit reads it.
	origInitPath := initPath
	t.Cleanup(func() { initPath = origInitPath })

	dir := t.TempDir()
	initPath = dir

	// Set flag values to configure a remote without interactive prompts.
	remoteName = "upstream"
	remoteURL = "https://example.com/nest.git"
	branchName = "develop"

	output := captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit returned error: %v", err)
		}
	})

	// Requirement 6.1: output includes remote name and URL.
	if !strings.Contains(output, "upstream") {
		t.Errorf("expected output to contain remote name 'upstream', got:\n%s", output)
	}
	if !strings.Contains(output, "https://example.com/nest.git") {
		t.Errorf("expected output to contain remote URL 'https://example.com/nest.git', got:\n%s", output)
	}

	// Requirement 6.2: output suggests git push -u <remote_name> <branch_name>.
	expectedPush := "git push -u upstream develop"
	if !strings.Contains(output, expectedPush) {
		t.Errorf("expected output to contain '%s', got:\n%s", expectedPush, output)
	}
}

// TestSuccessOutputWithoutRemote verifies that when runInit completes without an
// upstream remote configured (non-terminal, no --remote-url flag), the success output
// suggests manually adding a remote as a next step.
// Requirements: 6.3
func TestSuccessOutputWithoutRemote(t *testing.T) {
	saveAndRestoreRemoteFlags(t)

	// Save and restore initPath since runInit reads it.
	origInitPath := initPath
	t.Cleanup(func() { initPath = origInitPath })

	dir := t.TempDir()
	initPath = dir

	// No remote URL flag — non-terminal stdin will cause remote config to be skipped.
	remoteURL = ""

	// Replace stdin with a pipe (not a terminal) so isTerminal() returns false
	// and configureUpstreamRemote skips silently.
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		if err := r.Close(); err != nil {
			t.Fatalf("failed to close pipe reader: %v", err)
		}
	})

	output := captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit returned error: %v", err)
		}
	})

	// Requirement 6.3: output suggests manually adding a remote.
	if !strings.Contains(output, "git remote add") {
		t.Errorf("expected output to suggest 'git remote add' when no remote configured, got:\n%s", output)
	}

	// Should NOT contain a push command since no remote was configured.
	if strings.Contains(output, "git push -u") {
		t.Errorf("expected output NOT to contain 'git push -u' when no remote configured, got:\n%s", output)
	}
}
