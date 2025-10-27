package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sjramblings/awsmetalprices/internal/config"
	"github.com/sjramblings/awsmetalprices/internal/models"
	"github.com/spf13/cobra"
)

var catalogPath string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate pricing catalog completeness",
	Long:  `Validates that a pricing catalog contains all expected instances, regions, and OS configurations.`,
	RunE:  runValidate,
}

func init() {
	validateCmd.Flags().StringVar(&catalogPath, "catalog", "", "Path to catalog JSON file (required)")
	validateCmd.MarkFlagRequired("catalog")
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load catalog
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to read catalog: %w", err)
	}

	var catalog models.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("failed to parse catalog JSON: %w", err)
	}

	fmt.Printf("Validating catalog with %d items...\n\n", len(catalog.Items))

	// Build expected combinations
	instances := cfg.GetAllInstances()
	expectedCombos := len(instances) * len(cfg.Regions) * len(cfg.OsMatrix)

	fmt.Printf("Expected combinations: %d\n", expectedCombos)
	fmt.Printf("  - Instances: %d\n", len(instances))
	fmt.Printf("  - Regions: %d\n", len(cfg.Regions))
	fmt.Printf("  - OS configurations: %d\n\n", len(cfg.OsMatrix))

	// Validate instances
	instanceMap := make(map[string]bool)
	for _, item := range catalog.Items {
		instanceMap[item.InstanceType] = true
	}

	missingInstances := []string{}
	for _, inst := range instances {
		if !instanceMap[inst] {
			missingInstances = append(missingInstances, inst)
		}
	}

	fmt.Println("Instance coverage:")
	if len(missingInstances) == 0 {
		fmt.Println("  ✓ All configured instances present")
	} else {
		fmt.Printf("  ✗ Missing instances: %v\n", missingInstances)
	}

	// Validate regions
	regionMap := make(map[string]bool)
	for _, item := range catalog.Items {
		regionMap[item.Region] = true
	}

	missingRegions := []string{}
	for _, region := range cfg.Regions {
		if !regionMap[region] {
			missingRegions = append(missingRegions, region)
		}
	}

	fmt.Println("Region coverage:")
	if len(missingRegions) == 0 {
		fmt.Println("  ✓ All configured regions present")
	} else {
		fmt.Printf("  ✗ Missing regions: %v\n", missingRegions)
	}

	// Validate OS configurations
	osMap := make(map[string]bool)
	for _, item := range catalog.Items {
		osMap[item.OsLabel] = true
	}

	missingOS := []string{}
	for _, os := range cfg.OsMatrix {
		if !osMap[os.Label] {
			missingOS = append(missingOS, os.Label)
		}
	}

	fmt.Println("OS coverage:")
	if len(missingOS) == 0 {
		fmt.Println("  ✓ All configured OS types present")
	} else {
		fmt.Printf("  ✗ Missing OS types: %v\n", missingOS)
	}

	// Validate data completeness
	fmt.Println("\nData completeness:")
	incompleteItems := 0
	for _, item := range catalog.Items {
		if item.VCpu == 0 || item.MemoryGiB == 0 {
			incompleteItems++
			fmt.Printf("  ✗ Missing attributes: %s/%s/%s\n", item.Region, item.InstanceType, item.OsLabel)
		}
		if item.OnDemandUSDH == 0 {
			incompleteItems++
			fmt.Printf("  ✗ Missing On-Demand price: %s/%s/%s\n", item.Region, item.InstanceType, item.OsLabel)
		}
	}

	if incompleteItems == 0 {
		fmt.Println("  ✓ All items have complete data")
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 50))
	allValid := len(missingInstances) == 0 && len(missingRegions) == 0 && len(missingOS) == 0 && incompleteItems == 0
	if allValid {
		fmt.Println("✓ Validation PASSED")
		return nil
	} else {
		fmt.Println("✗ Validation FAILED")
		return fmt.Errorf("validation failed with errors")
	}
}
