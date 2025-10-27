package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sjramblings/awsmetalprices/internal/models"
	"github.com/spf13/cobra"
)

var (
	instanceFilter string
	regionFilter   string
	osFilter       string
	printFormat    string
)

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Print filtered pricing data",
	Long:  `Prints pricing data from a catalog, optionally filtered by instance type, region, and OS.`,
	RunE:  runPrint,
}

func init() {
	printCmd.Flags().StringVar(&catalogPath, "catalog", "", "Path to catalog JSON file (required)")
	printCmd.Flags().StringVar(&instanceFilter, "instance", "", "Filter by instance type")
	printCmd.Flags().StringVar(&regionFilter, "region", "", "Filter by region")
	printCmd.Flags().StringVar(&osFilter, "os", "", "Filter by OS label")
	printCmd.Flags().StringVar(&printFormat, "format", "table", "Output format (table or json)")

	printCmd.MarkFlagRequired("catalog")
}

func runPrint(cmd *cobra.Command, args []string) error {
	// Load catalog
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to read catalog: %w", err)
	}

	var catalog models.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("failed to parse catalog JSON: %w", err)
	}

	// Apply filters
	filtered := catalog.Filter(instanceFilter, regionFilter, osFilter)

	if len(filtered) == 0 {
		fmt.Println("No items match the specified filters")
		return nil
	}

	// Output based on format
	if printFormat == "json" {
		output, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}
		fmt.Println(string(output))
	} else {
		// Table format
		printTable(filtered)
	}

	return nil
}

func printTable(items []models.PricePoint) {
	fmt.Println("Region       | Instance Type     | OS      | vCPU | Mem(GiB) | On-Demand | 1yr RI AU | 3yr RI AU")
	fmt.Println("-------------|-------------------|---------|------|----------|-----------|-----------|----------")

	for _, item := range items {
		ri1yr := item.ReservedUSDH["1yr_AU"]
		ri3yr := item.ReservedUSDH["3yr_AU"]

		fmt.Printf("%-12s | %-17s | %-7s | %4d | %8.1f | %9.4f | %9.4f | %9.4f\n",
			item.Region,
			item.InstanceType,
			item.OsLabel,
			item.VCpu,
			item.MemoryGiB,
			item.OnDemandUSDH,
			ri1yr,
			ri3yr,
		)
	}

	fmt.Printf("\nTotal: %d items\n", len(items))
}
