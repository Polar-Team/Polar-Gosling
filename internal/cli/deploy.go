package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/polar-gosling/gosling/internal/deployer"
	"github.com/polar-gosling/gosling/internal/mothergoose"
	"github.com/polar-gosling/gosling/internal/parser"
	"github.com/spf13/cobra"
)

var (
	deployDryRun bool
	deployCloud  string
	deployRegion string
	deployAPIURL string
	deployAPIKey string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy resources from Nest repository",
	Long:  "Deploy resources from Nest repository to cloud providers.",
	RunE:  runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "Preview changes")
	deployCmd.Flags().StringVar(&deployCloud, "cloud", "", "Cloud provider")
	deployCmd.Flags().StringVar(&deployRegion, "region", "", "Cloud region")
	deployCmd.Flags().StringVar(&deployAPIURL, "api-url", "", "MotherGoose API URL")
	deployCmd.Flags().StringVar(&deployAPIKey, "api-key", "", "MotherGoose API key")
	deployCmd.MarkFlagRequired("api-url")
	deployCmd.MarkFlagRequired("api-key")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	if deployCloud == "" {
		return fmt.Errorf("--cloud flag is required")
	}
	if deployRegion == "" {
		return fmt.Errorf("--region flag is required")
	}
	var cloudProvider deployer.CloudProvider
	switch deployCloud {
	case "yandex":
		cloudProvider = deployer.CloudProviderYandex
	case "aws":
		cloudProvider = deployer.CloudProviderAWS
	default:
		return fmt.Errorf("unsupported cloud provider: %s", deployCloud)
	}
	nestRoot, err := findNestRoot()
	if err != nil {
		return fmt.Errorf("failed to find Nest repository: %w", err)
	}
	fmt.Printf("Found Nest repository at: %s\n", nestRoot)
	eggsDir := filepath.Join(nestRoot, "Eggs")
	eggs, err := parseEggConfigs(eggsDir)
	if err != nil {
		return fmt.Errorf("failed to parse Egg configurations: %w", err)
	}
	if len(eggs) == 0 {
		return fmt.Errorf("no Egg configurations found")
	}
	fmt.Printf("Found %d Egg configuration(s)\n", len(eggs))

	client := mothergoose.NewClient(deployAPIURL, deployAPIKey)

	for _, egg := range eggs {
		fmt.Printf("\n=== Deploying Egg: %s ===\n", egg.Name)
		if err := deployEgg(ctx, egg, cloudProvider, deployRegion, client); err != nil {
			return fmt.Errorf("failed to deploy egg %s: %w", egg.Name, err)
		}
	}
	if deployDryRun {
		fmt.Println("\nDry-run completed successfully.")
	} else {
		fmt.Println("\nDeployment completed successfully.")
	}
	return nil
}

func parseEggConfigs(eggsDir string) ([]*deployer.EggConfig, error) {
	var eggs []*deployer.EggConfig
	entries, err := os.ReadDir(eggsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read Eggs directory: %w", err)
	}
	p := parser.NewParser()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		configPath := filepath.Join(eggsDir, entry.Name(), "config.fly")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}
		config, err := p.ParseFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
		}
		egg, err := convertToEggConfig(config, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to convert config: %w", err)
		}
		eggs = append(eggs, egg)
	}
	return eggs, nil
}

func convertToEggConfig(config *parser.Config, name string) (*deployer.EggConfig, error) {
	var eggBlock *parser.Block
	for i := range config.Blocks {
		if config.Blocks[i].Type == "egg" {
			eggBlock = &config.Blocks[i]
			break
		}
	}
	if eggBlock == nil {
		return nil, fmt.Errorf("no egg block found")
	}
	egg := &deployer.EggConfig{
		Name:        name,
		Environment: make(map[string]string),
	}
	if typeAttr, ok := eggBlock.GetAttribute("type"); ok {
		if typeStr, err := typeAttr.AsString(); err == nil {
			egg.Type = deployer.RunnerType(typeStr)
		}
	}
	for i := range eggBlock.Blocks {
		childBlock := &eggBlock.Blocks[i]
		switch childBlock.Type {
		case "cloud":
			if provider, ok := childBlock.GetAttribute("provider"); ok {
				if providerStr, err := provider.AsString(); err == nil {
					switch providerStr {
					case "yandex":
						egg.Cloud.Provider = deployer.CloudProviderYandex
					case "aws":
						egg.Cloud.Provider = deployer.CloudProviderAWS
					}
				}
			}
			if region, ok := childBlock.GetAttribute("region"); ok {
				if regionStr, err := region.AsString(); err == nil {
					egg.Cloud.Region = regionStr
				}
			}
		case "resources":
			if cpu, ok := childBlock.GetAttribute("cpu"); ok {
				if cpuInt, err := cpu.AsInt(); err == nil {
					egg.Resources.CPU = cpuInt
				}
			}
			if memory, ok := childBlock.GetAttribute("memory"); ok {
				if memInt, err := memory.AsInt(); err == nil {
					egg.Resources.Memory = memInt
				}
			}
			if disk, ok := childBlock.GetAttribute("disk"); ok {
				if diskInt, err := disk.AsInt(); err == nil {
					egg.Resources.Disk = diskInt
				}
			}
		case "runner":
			if tags, ok := childBlock.GetAttribute("tags"); ok {
				if tagList, err := tags.AsList(); err == nil {
					var tagStrings []string
					for _, tag := range tagList {
						if tagStr, err := tag.AsString(); err == nil {
							tagStrings = append(tagStrings, tagStr)
						}
					}
					egg.Runner.Tags = tagStrings
				}
			}
			if concurrent, ok := childBlock.GetAttribute("concurrent"); ok {
				if concInt, err := concurrent.AsInt(); err == nil {
					egg.Runner.Concurrent = concInt
				}
			}
			if idleTimeout, ok := childBlock.GetAttribute("idle_timeout"); ok {
				if timeoutStr, err := idleTimeout.AsString(); err == nil {
					if duration, err := time.ParseDuration(timeoutStr); err == nil {
						egg.Runner.IdleTimeout = duration
					}
				}
			}
		case "gitlab":
			if projectID, ok := childBlock.GetAttribute("project_id"); ok {
				if projInt, err := projectID.AsInt(); err == nil {
					egg.GitLab.ProjectID = projInt
				}
			}
			if tokenSecret, ok := childBlock.GetAttribute("token_secret"); ok {
				if tokenStr, err := tokenSecret.AsString(); err == nil {
					egg.GitLab.TokenSecret = tokenStr
				}
			}
		case "environment":
			for key, attr := range childBlock.Attributes {
				if valStr, err := attr.AsString(); err == nil {
					egg.Environment[key] = valStr
				}
			}
		}
	}
	return egg, nil
}

func deployEgg(ctx context.Context, egg *deployer.EggConfig, provider deployer.CloudProvider, region string, client mothergoose.MotherGooseClient) error {
	configHash, err := generateConfigHash(egg)
	if err != nil {
		return fmt.Errorf("failed to generate hash: %w", err)
	}
	fmt.Printf("Config hash: %s\n", configHash)

	// Check if configuration has changed
	status, err := client.GetEggStatus(ctx, egg.Name)
	if err == nil && status.LatestPlan != nil && status.LatestPlan.ConfigHash == configHash {
		fmt.Println("No changes detected")
		return nil
	}

	plan := &deployer.DeploymentPlan{
		ID:         uuid.New().String(),
		EggName:    egg.Name,
		PlanType:   "runner",
		ConfigHash: configHash,
		CreatedAt:  time.Now(),
		Status:     "pending",
		Metadata: map[string]interface{}{
			"runner_type": string(egg.Type),
			"cloud":       string(provider),
			"region":      region,
		},
	}

	planBinary, err := generatePlanBinary(egg)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}
	plan.PlanBinary = planBinary

	if deployDryRun {
		fmt.Println("\n--- Deployment Plan (Dry Run) ---")
		fmt.Printf("Plan ID: %s\n", plan.ID)
		fmt.Printf("Egg Name: %s\n", plan.EggName)
		fmt.Printf("Runner Type: %s\n", egg.Type)
		fmt.Printf("Cloud: %s\n", provider)
		fmt.Printf("Region: %s\n", region)
		fmt.Printf("Resources: CPU=%d, Memory=%dMB, Disk=%dGB\n", egg.Resources.CPU, egg.Resources.Memory, egg.Resources.Disk)
		fmt.Println("\nNo resources will be created")
		return nil
	}

	// Store Egg configuration via MotherGoose API
	if err := client.CreateOrUpdateEgg(ctx, egg); err != nil {
		return fmt.Errorf("failed to store egg configuration: %w", err)
	}
	fmt.Printf("Egg configuration stored successfully\n")

	fmt.Println("Deployment applied successfully")
	return nil
}

func generateConfigHash(egg *deployer.EggConfig) (string, error) {
	configJSON, err := json.Marshal(egg)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(configJSON)
	return hex.EncodeToString(hash[:]), nil
}

func generatePlanBinary(egg *deployer.EggConfig) ([]byte, error) {
	planData := map[string]interface{}{
		"egg_name":    egg.Name,
		"runner_type": egg.Type,
		"cloud":       egg.Cloud,
		"resources":   egg.Resources,
		"timestamp":   time.Now().Unix(),
	}
	return json.Marshal(planData)
}
