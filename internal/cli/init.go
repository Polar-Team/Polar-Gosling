package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	initPath   string
	remoteName string
	remoteURL  string
	branchName string
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
	initCmd.Flags().StringVar(&remoteName, "remote-name", "origin", "Name for the upstream remote")
	initCmd.Flags().StringVar(&remoteURL, "remote-url", "", "URL for the upstream remote repository")
	initCmd.Flags().StringVar(&branchName, "branch", "main", "Default branch name")
}

// TODO: Placeholder for new functions
// isTerminal reports whether stdin is connected to a terminal.
// func isTerminal() bool {
// 	fi, err := os.Stdin.Stat()
// 	if err != nil {
// 		return false
// 	}
// 	return fi.Mode()&os.ModeCharDevice != 0
// }

// promptWithDefault prints a prompt to stdout and reads a line from stdin.
// If the user input is empty or whitespace-only, it returns defaultVal.
func promptWithDefault(prompt, defaultVal string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			return input
		}
	}
	return defaultVal
}

// initGitRepo initializes a new Git repository in the given directory.
func initGitRepo(dir string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("failed to initialize git repository: %v: %s", err, msg)
		}
		return fmt.Errorf("failed to initialize git repository: %v", err)
	}
	return nil
}

// TODO: Placeholder for new functions
// addGitRemote adds a named remote to the git repository at dir.
// func addGitRemote(dir, name, url string) error {
// 	cmd := exec.Command("git", "remote", "add", name, url)
// 	cmd.Dir = dir
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("failed to add upstream remote: %w", err)
// 	}
// 	return nil
// }

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
		filepath.Join(absPath, "MG"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  ✓ Created %s/\n", filepath.Base(dir))
	}

	// Create UF/config.fly with default UglyFox configuration
	ufConfigPath := filepath.Join(absPath, "UF", "config.fly")
	if err := os.WriteFile(ufConfigPath, []byte(defaultUglyFoxConfig()), 0644); err != nil {
		return fmt.Errorf("failed to create UF/config.fly: %w", err)
	}
	fmt.Println("  ✓ Created UF/config.fly (defaultUglyfox)")

	// Create MG/config.fly with default MotherGoose configuration
	mgConfigPath := filepath.Join(absPath, "MG", "config.fly")
	if err := os.WriteFile(mgConfigPath, []byte(defaultMotherGooseConfig()), 0644); err != nil {
		return fmt.Errorf("failed to create MG/config.fly: %w", err)
	}
	fmt.Println("  ✓ Created MG/config.fly (defaultMothergoose)")

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

	// Initialize Git repository (skip if already initialized)
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := initGitRepo(absPath); err != nil {
			return err
		}
		fmt.Println("  ✓ Initialized Git repository")
	} else {
		fmt.Println("  ✓ Git repository already exists, skipping init")
	}
	fmt.Println("\n✅ Nest repository initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add an Egg configuration: gosling add egg <name>")
	fmt.Println("  2. Customize UglyFox policies: edit UF/config.fly")
	fmt.Println("  3. Customize MotherGoose infra: edit MG/config.fly")
	fmt.Println("  4. Validate your configuration: gosling validate")

	return nil
}

func defaultUglyFoxConfig() string {
	return `# UglyFox Configuration: defaultUglyfox
# Runner lifecycle management: pruning policies and Apex/Nadir pool rules

uglyfox {
  pruning {
    failed_threshold = 3
    check_interval   = "5m"
    max_age          = "24h"
  }

  runners_condition "default" {
    # TODO: list the egg names this condition applies to
    eggs_entities = ["my-app"]

    apex {
      max_count = 5
      min_count = 1
    }

    nadir {
      max_count    = 3
      min_count    = 0
      idle_timeout = "30m"
    }
  }
}
`
}

func defaultMotherGooseConfig() string {
	return `# MotherGoose Infrastructure Configuration: defaultMothergoose
# API Gateway, serverless containers, message queues, cloud triggers, database, storage

mothergoose {
  api_gateway {
    name         = "polar-gosling-api"
    openapi_spec = "openapi.yaml"
  }

  fastapi_app {
    name    = "mothergoose-api"
    runtime = "python312"
    memory  = 512
    timeout = 30
  }

  celery_workers {
    name    = "mothergoose-celery"
    runtime = "python312"
    memory  = 1024
    timeout = 300
  }

  uglyfox_workers {
    name    = "uglyfox-celery"
    runtime = "python312"
    memory  = 512
    timeout = 180
  }

  message_queues {
    webhook_queue {
      name               = "mothergoose-webhooks"
      visibility_timeout = 300
    }

    uglyfox_queue {
      name               = "uglyfox-tasks"
      visibility_timeout = 180
    }
  }

  triggers {
    git_sync {
      name     = "git-sync-trigger"
      schedule = "*/5 * * * *"
      endpoint = "/internal/sync-git"
    }

    health_check {
      name     = "uglyfox-health-trigger"
      schedule = "*/10 * * * *"
      endpoint = "/internal/uglyfox/health-check"
    }
  }

  database {
    type = "ydb"
    name = "polar-gosling-db"
    mode = "serverless"
  }

  storage {
    bucket_name = "polar-gosling-storage"
    region      = "ru-central1"
  }

  service_accounts {
    mothergoose {
      name  = "mothergoose-sa"
      roles = ["lockbox.payloadViewer", "ydb.editor"]
    }

    uglyfox {
      name  = "uglyfox-sa"
      roles = ["lockbox.payloadViewer", "ydb.viewer"]
    }
  }
}
`
}
