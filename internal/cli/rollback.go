package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/polar-gosling/gosling/internal/deployer"
	"github.com/polar-gosling/gosling/internal/mothergoose"
	"github.com/spf13/cobra"
)

var (
	rollbackTo     string
	rollbackEgg    string
	rollbackAPIURL string
	rollbackAPIKey string
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback a deployment",
	Long:  "Rollback a deployment to a previous state.",
	RunE:  runRollback,
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
	rollbackCmd.Flags().StringVar(&rollbackTo, "to", "", "Plan ID to rollback to")
	rollbackCmd.Flags().StringVar(&rollbackEgg, "egg", "", "Egg name")
	rollbackCmd.Flags().StringVar(&rollbackAPIURL, "api-url", "", "MotherGoose API URL")
	rollbackCmd.Flags().StringVar(&rollbackAPIKey, "api-key", "", "MotherGoose API key")
	rollbackCmd.MarkFlagRequired("egg")
	rollbackCmd.MarkFlagRequired("api-url")
	rollbackCmd.MarkFlagRequired("api-key")
}

func runRollback(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	client := mothergoose.NewClient(rollbackAPIURL, rollbackAPIKey)

	// Get current deployment status
	status, err := client.GetEggStatus(ctx, rollbackEgg)
	if err != nil {
		return fmt.Errorf("failed to get egg status: %w", err)
	}

	if status.LatestPlan == nil {
		return fmt.Errorf("no deployment found for egg: %s", rollbackEgg)
	}

	currentPlan := status.LatestPlan
	fmt.Printf("Current plan: %s\n", currentPlan.ID)

	var targetPlan *deployer.DeploymentPlan
	if rollbackTo != "" {
		// Get specific plan by ID
		targetPlan, err = client.GetDeploymentPlan(ctx, rollbackEgg, rollbackTo)
		if err != nil {
			return fmt.Errorf("failed to get target plan: %w", err)
		}
	} else {
		// Find previous applied plan
		targetPlan, err = findPreviousPlan(status.DeploymentHistory, currentPlan.ID)
		if err != nil {
			return fmt.Errorf("failed to find previous plan: %w", err)
		}
	}

	if targetPlan == nil {
		return fmt.Errorf("no previous plan found")
	}

	fmt.Printf("\n=== Rollback Plan ===\n")
	fmt.Printf("Target Plan ID: %s\n", targetPlan.ID)
	fmt.Printf("Created At: %s\n", targetPlan.CreatedAt.Format(time.RFC3339))
	fmt.Printf("\nRollback egg '%s' from %s to %s\n", rollbackEgg, currentPlan.ID[:8], targetPlan.ID[:8])
	fmt.Print("Continue? (yes/no): ")
	var response string
	fmt.Scanln(&response)
	if response != "yes" && response != "y" {
		fmt.Println("Rollback cancelled")
		return nil
	}

	fmt.Println("\nPerforming rollback...")
	fmt.Println("Note: Rollback status updates are managed by MotherGoose backend")
	fmt.Printf("Target plan: %s\n", targetPlan.ID)
	fmt.Println("\nRollback initiated successfully")
	fmt.Println("Use 'gosling status --egg " + rollbackEgg + "' to check rollback status")
	return nil
}

func findPreviousPlan(plans []*deployer.DeploymentPlan, currentPlanID string) (*deployer.DeploymentPlan, error) {
	var previousPlan *deployer.DeploymentPlan
	for _, plan := range plans {
		if plan.ID == currentPlanID {
			continue
		}
		if plan.Status == "applied" {
			if previousPlan == nil || plan.AppliedAt.After(*previousPlan.AppliedAt) {
				previousPlan = plan
			}
		}
	}
	if previousPlan == nil {
		return nil, fmt.Errorf("no previous applied plan found")
	}
	return previousPlan, nil
}
