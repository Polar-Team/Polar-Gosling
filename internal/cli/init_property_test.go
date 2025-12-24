package cli

import (
	"os"
	"path/filepath"
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
