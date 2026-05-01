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
		if err := configureUpstreamRemote(dir); err != nil {
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
		if err := configureUpstreamRemote(dir); err != nil {
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
		if err := configureUpstreamRemote(dir); err != nil {
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
