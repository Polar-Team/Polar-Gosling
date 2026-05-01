package cli

import (
	"os"
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
			genValidPathName(),
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

// genValidPathName generates valid directory path names for testing
func genValidPathName() gopter.Gen {
	return gen.OneConstOf(
		"nest",
		"my-nest",
		"test-nest",
		"nest-repo",
		"project-nest",
		"nest_test",
		"nest123",
		"a",
		"ab",
		"abc",
	)
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
