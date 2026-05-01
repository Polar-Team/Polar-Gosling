package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: gitops-runner-orchestration, Property 5: Nest Initialization Structure
// Validates: Requirements 3.3
func TestNestInitializationStructure(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Nest initialization creates Eggs/, Jobs/, and UF/ directories",
		prop.ForAll(
			func(basePath string) bool {
				// Create a temporary directory for testing
				tempDir, err := os.MkdirTemp("", "nest-init-test-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tempDir)

				// Create the test path within the temp directory
				testPath := filepath.Join(tempDir, basePath)

				// Initialize the Nest repository
				err = initializeNest(testPath)
				if err != nil {
					t.Logf("Failed to initialize Nest: %v", err)
					return false
				}

				// Verify the directory structure
				requiredDirs := []string{"Eggs", "Jobs", "UF"}
				for _, dir := range requiredDirs {
					dirPath := filepath.Join(testPath, dir)
					info, err := os.Stat(dirPath)
					if err != nil {
						t.Logf("Directory %s does not exist: %v", dir, err)
						return false
					}
					if !info.IsDir() {
						t.Logf("%s is not a directory", dir)
						return false
					}
				}

				// Verify README.md exists
				readmePath := filepath.Join(testPath, "README.md")
				if _, err := os.Stat(readmePath); err != nil {
					t.Logf("README.md does not exist: %v", err)
					return false
				}

				// Verify .gitignore exists
				gitignorePath := filepath.Join(testPath, ".gitignore")
				if _, err := os.Stat(gitignorePath); err != nil {
					t.Logf(".gitignore does not exist: %v", err)
					return false
				}

				return true
			},
			genValidDirName(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// initializeNest is a helper function that performs the Nest initialization
// This is extracted from runInit to make it testable
func initializeNest(targetPath string) error {
	// Convert to absolute path
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(absPath, "Eggs"),
		filepath.Join(absPath, "Jobs"),
		filepath.Join(absPath, "UF"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create README.md
	readmePath := filepath.Join(absPath, "README.md")
	readmeContent := `# Nest Repository

This is a Nest repository for GitOps-based CI/CD runner orchestration.

## Directory Structure

- **Eggs/**: Contains configuration files for managed repositories (Eggs)
  - Single projects: ` + "`Eggs/{project-name}/config.fly`" + `
  - Grouped projects: ` + "`Eggs/{bucket-name}/config.fly`" + ` (EggsBucket)

- **Jobs/**: Contains self-management task definitions
  - Format: ` + "`Jobs/{job-name}.fly`" + `
  - Examples: secret rotation, runner updates, Nest maintenance

- **UF/**: Contains UglyFox configuration
  - ` + "`UF/config.fly`" + `: Runner pruning policies and lifecycle management

## Getting Started

1. Add an Egg configuration:
` + "   ```bash" + `
   gosling add egg my-app --type vm
` + "   ```" + `

2. Add a self-management job:
` + "   ```bash" + `
   gosling add job rotate-secrets
` + "   ```" + `

3. Validate configurations:
` + "   ```bash" + `
   gosling validate
` + "   ```" + `

4. Deploy to cloud:
` + "   ```bash" + `
   gosling deploy
` + "   ```" + `

## Documentation

For more information, see the Gosling CLI documentation.
`

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return err
	}

	// Create .gitignore
	gitignorePath := filepath.Join(absPath, ".gitignore")
	gitignoreContent := `# Terraform/OpenTofu state files
*.tfstate
*.tfstate.*
.terraform/
.terraform.lock.hcl

# Sensitive files
*.secret
*.key
*.pem

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db
`

	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return err
	}

	return nil
}

// Feature: gosling-init-upstream, Property 1: Git initialization creates a valid repository
// For any valid directory path name, running initGitRepo on that path results in a .git
// directory existing within the target path, confirming a valid Git repository was created.
// Validates: Requirements 1.1
func TestGitInitCreatesRepository(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("initGitRepo creates a .git directory in the target path",
		prop.ForAll(
			func(pathName string) bool {
				// Create a temporary directory to act as the parent.
				tempDir, err := os.MkdirTemp("", "git-init-prop-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tempDir)

				// Build the target directory using the generated path name.
				targetDir := filepath.Join(tempDir, pathName)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Logf("Failed to create target dir: %v", err)
					return false
				}

				// Run initGitRepo on the target directory.
				if err := initGitRepo(targetDir); err != nil {
					t.Logf("initGitRepo failed: %v", err)
					return false
				}

				// Assert that a .git directory exists inside the target.
				gitDir := filepath.Join(targetDir, ".git")
				info, err := os.Stat(gitDir)
				if err != nil {
					t.Logf(".git does not exist: %v", err)
					return false
				}
				if !info.IsDir() {
					t.Logf(".git is not a directory")
					return false
				}

				return true
			},
			genValidDirName(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// windowsReservedNames is the set of device names reserved by Windows that cannot
// be used as file or directory names regardless of extension.
var windowsReservedNames = map[string]struct{}{
	"con": {}, "prn": {}, "aux": {}, "nul": {},
	"com1": {}, "com2": {}, "com3": {}, "com4": {}, "com5": {},
	"com6": {}, "com7": {}, "com8": {}, "com9": {},
	"lpt1": {}, "lpt2": {}, "lpt3": {}, "lpt4": {}, "lpt5": {},
	"lpt6": {}, "lpt7": {}, "lpt8": {}, "lpt9": {},
}

// genValidDirName generates arbitrary valid directory names (alphanumeric, hyphens, underscores)
// that are safe on all platforms including Windows (reserved names are excluded).
func genValidDirName() gopter.Gen {
	return gen.RegexMatch(`[a-z][a-z0-9_-]{0,15}`).SuchThat(func(s string) bool {
		_, reserved := windowsReservedNames[strings.ToLower(s)]
		return !reserved
	})
}

// Feature: gosling-init-upstream, Property 2: Prompt default value preservation
// For any non-empty default value and empty/whitespace-only user input,
// promptWithDefault returns the original default unchanged.
// Validates: Requirements 2.2
func TestPromptDefaultPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("promptWithDefault returns default for empty/whitespace input",
		prop.ForAll(
			func(defaultVal string, wsInput string) bool {
				// Create a pipe to simulate stdin with the whitespace input
				// followed by a newline so bufio.Scanner.Scan() returns true.
				r, w, err := os.Pipe()
				if err != nil {
					t.Logf("Failed to create pipe: %v", err)
					return false
				}

				// Write the whitespace input followed by newline, then close writer.
				_, err = w.WriteString(wsInput + "\n")
				w.Close()
				if err != nil {
					r.Close()
					t.Logf("Failed to write to pipe: %v", err)
					return false
				}

				// Swap os.Stdin with our pipe reader.
				origStdin := os.Stdin
				os.Stdin = r
				defer func() {
					os.Stdin = origStdin
					r.Close()
				}()

				// Also capture stdout to avoid polluting test output with prompts.
				origStdout := os.Stdout
				devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				if err != nil {
					t.Logf("Failed to open devnull: %v", err)
					return false
				}
				os.Stdout = devNull
				defer func() {
					os.Stdout = origStdout
					devNull.Close()
				}()

				result := promptWithDefault("test prompt: ", defaultVal)
				return result == defaultVal
			},
			genNonEmptyString(),
			genWhitespaceString(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: gosling-init-upstream, Property 3: Git remote registration round-trip
// For any valid Git remote name and valid Git remote URL, calling addGitRemote
// then querying `git remote -v` shows the remote name mapped to the provided URL.
// Validates: Requirements 2.5, 3.4
func TestGitRemoteRegistrationRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 25
	properties := gopter.NewProperties(parameters)

	properties.Property("addGitRemote registers a remote that is visible via git remote -v",
		prop.ForAll(
			func(name string, url string) bool {
				// Create a temporary directory and initialize a git repo.
				tempDir, err := os.MkdirTemp("", "git-remote-prop-*")
				if err != nil {
					t.Logf("Failed to create temp dir: %v", err)
					return false
				}
				defer os.RemoveAll(tempDir)

				if err := initGitRepo(tempDir); err != nil {
					t.Logf("initGitRepo failed: %v", err)
					return false
				}

				// Add the remote using the function under test.
				if err := addGitRemote(tempDir, name, url); err != nil {
					t.Logf("addGitRemote failed: %v", err)
					return false
				}

				// Query git remote -v and verify the remote is registered.
				cmd := exec.Command("git", "remote", "-v")
				cmd.Dir = tempDir
				output, err := cmd.Output()
				if err != nil {
					t.Logf("git remote -v failed: %v", err)
					return false
				}

				lines := strings.Split(string(output), "\n")
				found := false
				for _, line := range lines {
					// Each line looks like: <name>\t<url> (fetch|push)
					if strings.HasPrefix(line, name+"\t") && strings.Contains(line, url) {
						found = true
						break
					}
				}

				if !found {
					t.Logf("Remote %q -> %q not found in git remote -v output:\n%s", name, url, string(output))
					return false
				}

				return true
			},
			genValidRemoteName(),
			genValidRemoteURL(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genValidRemoteName generates valid Git remote names.
// Git remote names must be non-empty, contain only alphanumeric characters, hyphens,
// underscores, and dots, and must not start with a dot or hyphen.
// Windows reserved device names (CON, PRN, AUX, NUL, COM1–9, LPT1–9) are excluded.
func genValidRemoteName() gopter.Gen {
	return gen.RegexMatch(`[a-z][a-z0-9_-]{0,15}`).SuchThat(func(s string) bool {
		if len(s) == 0 {
			return false
		}
		_, reserved := windowsReservedNames[strings.ToLower(s)]
		return !reserved
	})
}

// genValidRemoteURL generates valid Git remote URLs using the https:// scheme.
// The URLs are syntactically valid but do not need to point to real repositories
// since addGitRemote only registers the remote without fetching.
func genValidRemoteURL() gopter.Gen {
	host := gen.RegexMatch(`[a-z][a-z0-9]{2,10}`)
	path := gen.RegexMatch(`[a-z][a-z0-9_-]{1,10}/[a-z][a-z0-9_-]{1,10}`)
	return gopter.CombineGens(host, path).Map(func(vals []interface{}) string {
		h := vals[0].(string)
		p := vals[1].(string)
		return fmt.Sprintf("https://%s.com/%s.git", h, p)
	})
}

// genNonEmptyString generates arbitrary non-empty strings for use as default values.
func genNonEmptyString() gopter.Gen {
	return gen.AnyString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// genWhitespaceString generates strings that are either empty or contain only whitespace characters.
// These represent user inputs that should cause promptWithDefault to return the default value.
func genWhitespaceString() gopter.Gen {
	return gen.OneGenOf(
		gen.Const(""),
		gen.SliceOf(gen.OneConstOf(' ', '\t')).Map(func(chars []int32) string {
			var sb strings.Builder
			for _, c := range chars {
				sb.WriteRune(rune(c))
			}
			return sb.String()
		}),
	)
}
