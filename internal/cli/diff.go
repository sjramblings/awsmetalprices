package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sjramblings/awsmetalprices/internal/models"
	"github.com/spf13/cobra"
)

var (
	oldCatalogPath string
	newCatalogPath string
	diffFormat     string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare two pricing catalogs",
	Long:  `Compares pricing data between two catalog files and reports differences.`,
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&oldCatalogPath, "old", "", "Path to old catalog JSON file (required)")
	diffCmd.Flags().StringVar(&newCatalogPath, "new", "", "Path to new catalog JSON file (required)")
	diffCmd.Flags().StringVar(&diffFormat, "format", "table", "Output format (table or json)")

	diffCmd.MarkFlagRequired("old")
	diffCmd.MarkFlagRequired("new")
}

type priceDiff struct {
	Region       string
	InstanceType string
	OsLabel      string
	OldPrice     float64
	NewPrice     float64
	Delta        float64
	Percentage   float64
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Load old catalog
	oldData, err := os.ReadFile(oldCatalogPath)
	if err != nil {
		return fmt.Errorf("failed to read old catalog: %w", err)
	}

	var oldCatalog models.Catalog
	if err := json.Unmarshal(oldData, &oldCatalog); err != nil {
		return fmt.Errorf("failed to parse old catalog JSON: %w", err)
	}

	// Load new catalog
	newData, err := os.ReadFile(newCatalogPath)
	if err != nil {
		return fmt.Errorf("failed to read new catalog: %w", err)
	}

	var newCatalog models.Catalog
	if err := json.Unmarshal(newData, &newCatalog); err != nil {
		return fmt.Errorf("failed to parse new catalog JSON: %w", err)
	}

	fmt.Printf("Comparing catalogs:\n")
	fmt.Printf("  Old: %s (%d items)\n", oldCatalogPath, len(oldCatalog.Items))
	fmt.Printf("  New: %s (%d items)\n\n", newCatalogPath, len(newCatalog.Items))

	// Build maps for comparison
	oldMap := buildPriceMap(oldCatalog.Items)
	newMap := buildPriceMap(newCatalog.Items)

	// Find added items
	added := []string{}
	for key := range newMap {
		if _, exists := oldMap[key]; !exists {
			added = append(added, key)
		}
	}

	// Find removed items
	removed := []string{}
	for key := range oldMap {
		if _, exists := newMap[key]; !exists {
			removed = append(removed, key)
		}
	}

	// Find changed items
	changes := []priceDiff{}
	for key, newItem := range newMap {
		if oldItem, exists := oldMap[key]; exists {
			if oldItem.OnDemandUSDH != newItem.OnDemandUSDH {
				delta := newItem.OnDemandUSDH - oldItem.OnDemandUSDH
				percentage := (delta / oldItem.OnDemandUSDH) * 100

				changes = append(changes, priceDiff{
					Region:       newItem.Region,
					InstanceType: newItem.InstanceType,
					OsLabel:      newItem.OsLabel,
					OldPrice:     oldItem.OnDemandUSDH,
					NewPrice:     newItem.OnDemandUSDH,
					Delta:        delta,
					Percentage:   percentage,
				})
			}
		}
	}

	// Report results
	if len(added) > 0 {
		fmt.Printf("Added (%d items):\n", len(added))
		for _, key := range added {
			fmt.Printf("  + %s\n", key)
		}
		fmt.Println()
	}

	if len(removed) > 0 {
		fmt.Printf("Removed (%d items):\n", len(removed))
		for _, key := range removed {
			fmt.Printf("  - %s\n", key)
		}
		fmt.Println()
	}

	if len(changes) > 0 {
		fmt.Printf("Price Changes (%d items):\n\n", len(changes))

		if diffFormat == "json" {
			output, err := json.MarshalIndent(changes, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal output: %w", err)
			}
			fmt.Println(string(output))
		} else {
			fmt.Println("Region       | Instance Type     | OS      | Old Price | New Price | Delta    | Change %")
			fmt.Println("-------------|-------------------|---------|-----------|-----------|----------|----------")

			for _, change := range changes {
				changeSymbol := "↑"
				if change.Delta < 0 {
					changeSymbol = "↓"
				}

				fmt.Printf("%-12s | %-17s | %-7s | %9.4f | %9.4f | %s%8.4f | %+7.2f%%\n",
					change.Region,
					change.InstanceType,
					change.OsLabel,
					change.OldPrice,
					change.NewPrice,
					changeSymbol,
					abs(change.Delta),
					change.Percentage,
				)
			}
		}
		fmt.Println()
	}

	if len(added) == 0 && len(removed) == 0 && len(changes) == 0 {
		fmt.Println("No differences found")
	}

	return nil
}

func buildPriceMap(items []models.PricePoint) map[string]models.PricePoint {
	m := make(map[string]models.PricePoint)
	for _, item := range items {
		key := fmt.Sprintf("%s/%s/%s", item.Region, item.InstanceType, item.OsLabel)
		m[key] = item
	}
	return m
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
