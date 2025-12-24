package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initPath string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a new Nest repository",
	Long: `Initialize a new Nest repository with the standard directory structure.

The Nest repository will contain:
  - Eggs/     : Managed repository configurations
  - Jobs/     : Self-management task definitions
  - UF/       : UglyFox configuration for runner lifecycle management

Example:
  gosling init
  gosling init /path/to/nest
  gosling init --path /path/to/nest`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initPath, "path", "p", "", "Path to initialize Nest repository (default: current directory)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine the target path
	targetPath := initPath
	if len(args) > 0 {
		targetPath = args[0]
	}
	if targetPath == "" {
		var err error
		targetPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	fmt.Printf("Initializing Nest repository at: %s\n", absPath)

	// Create directory structure
	dirs := []string{
		filepath.Join(absPath, "Eggs"),
		filepath.Join(absPath, "Jobs"),
		filepath.Join(absPath, "UF"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  ✓ Created %s/\n", filepath.Base(dir))
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
		return fmt.Errorf("failed to create README.md: %w", err)
	}
	fmt.Println("  ✓ Created README.md")

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
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	fmt.Println("  ✓ Created .gitignore")

	fmt.Println("\n✅ Nest repository initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add an Egg configuration: gosling add egg <name>")
	fmt.Println("  2. Configure UglyFox policies: edit UF/config.fly")
	fmt.Println("  3. Validate your configuration: gosling validate")

	return nil
}
