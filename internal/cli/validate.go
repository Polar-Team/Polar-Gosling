package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/polar-gosling/gosling/internal/parser"
	"github.com/spf13/cobra"
)

var (
	validatePath string
	validateAll  bool
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate .fly configuration files",
	Long: `Validate .fly configuration files for syntax and semantic errors.

Without arguments, validates all .fly files in the Nest repository.
With a file argument, validates only that specific file.

Example:
  gosling validate
  gosling validate Eggs/my-app/config.fly
  gosling validate --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVarP(&validatePath, "path", "p", "", "Path to Nest repository (default: current directory)")
	validateCmd.Flags().BoolVarP(&validateAll, "all", "a", false, "Validate all .fly files in the repository")
}

func runValidate(cmd *cobra.Command, args []string) error {
	var filesToValidate []string

	if len(args) > 0 {
		// Validate specific file
		filePath := args[0]
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to resolve file path: %w", err)
		}
		filesToValidate = append(filesToValidate, absPath)
	} else {
		// Find Nest root
		nestRoot := validatePath
		if nestRoot == "" {
			var err error
			nestRoot, err = findNestRoot()
			if err != nil {
				return fmt.Errorf("not in a Nest repository: %w\nRun 'gosling init' to create a new Nest repository", err)
			}
		}

		// Find all .fly files
		var err error
		filesToValidate, err = findFlyFiles(nestRoot)
		if err != nil {
			return fmt.Errorf("failed to find .fly files: %w", err)
		}

		if len(filesToValidate) == 0 {
			fmt.Println("⚠️  No .fly files found in the repository")
			return nil
		}
	}

	fmt.Printf("Validating %d file(s)...\n\n", len(filesToValidate))

	// Validate each file
	p := parser.NewParser()
	hasErrors := false
	validCount := 0
	errorCount := 0

	for _, filePath := range filesToValidate {
		relPath, _ := filepath.Rel(validatePath, filePath)
		if relPath == "" {
			relPath = filePath
		}

		fmt.Printf("📄 %s\n", relPath)

		config, err := p.ParseFile(filePath)
		if err != nil {
			fmt.Printf("   ❌ Parse error: %v\n\n", err)
			hasErrors = true
			errorCount++
			continue
		}

		// Perform semantic validation
		if err := validateConfig(config, filePath); err != nil {
			fmt.Printf("   ❌ Validation error: %v\n\n", err)
			hasErrors = true
			errorCount++
			continue
		}

		fmt.Printf("   ✅ Valid\n\n")
		validCount++
	}

	// Print summary
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Summary: %d valid, %d errors\n", validCount, errorCount)

	if hasErrors {
		return fmt.Errorf("validation failed with %d error(s)", errorCount)
	}

	fmt.Println("✅ All files validated successfully!")
	return nil
}

func findFlyFiles(root string) ([]string, error) {
	var files []string

	// Search in Eggs directory
	eggsDir := filepath.Join(root, "Eggs")
	if info, err := os.Stat(eggsDir); err == nil && info.IsDir() {
		err := filepath.Walk(eggsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".fly") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Search in Jobs directory
	jobsDir := filepath.Join(root, "Jobs")
	if info, err := os.Stat(jobsDir); err == nil && info.IsDir() {
		err := filepath.Walk(jobsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".fly") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Search in UF directory
	ufDir := filepath.Join(root, "UF")
	if info, err := os.Stat(ufDir); err == nil && info.IsDir() {
		err := filepath.Walk(ufDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".fly") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func validateConfig(config *parser.Config, filePath string) error {
	if len(config.Blocks) == 0 {
		return fmt.Errorf("configuration file is empty")
	}

	// Use the parser's comprehensive validator
	validator := parser.NewValidator(config)
	result := validator.Validate()

	if !result.IsValid() {
		return fmt.Errorf("%s", result.Error())
	}

	// Additional file-location-based validation
	fileName := filepath.Base(filePath)
	dirName := filepath.Base(filepath.Dir(filePath))
	parentDir := filepath.Base(filepath.Dir(filepath.Dir(filePath)))

	// Determine expected block type
	var expectedBlockType string
	if parentDir == "Eggs" && fileName == "config.fly" {
		// Could be egg or eggsbucket
		expectedBlockType = "" // Will check both
	} else if parentDir == "Jobs" {
		expectedBlockType = "job"
	} else if dirName == "UF" && fileName == "config.fly" {
		expectedBlockType = "uglyfox"
	}

	// Validate blocks are in correct location
	if expectedBlockType != "" {
		for _, block := range config.Blocks {
			if block.Type != expectedBlockType {
				return fmt.Errorf("unexpected block type %q (expected %q)", block.Type, expectedBlockType)
			}
		}
	}

	return nil
}
