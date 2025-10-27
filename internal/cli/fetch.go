package cli

import (
	"context"
	"fmt"

	"github.com/sjramblings/awsmetalprices/internal/cache"
	"github.com/sjramblings/awsmetalprices/internal/config"
	"github.com/sjramblings/awsmetalprices/internal/pricing"
	"github.com/sjramblings/awsmetalprices/internal/render"
	"github.com/spf13/cobra"
)

var (
	outputPath string
	noCache    bool
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch EC2 pricing data from AWS",
	Long:  `Fetches pricing data for configured metal instances and exports to Markdown and JSON formats.`,
	RunE:  runFetch,
}

func init() {
	fetchCmd.Flags().StringVar(&outputPath, "out", "./pricing", "Output directory for pricing data")
	fetchCmd.Flags().BoolVar(&noCache, "no-cache", false, "Bypass cache and fetch fresh data")
}

func runFetch(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Fetching pricing data for %d instances across %d regions...\n",
		len(cfg.GetAllInstances()), len(cfg.Regions))

	// Initialize cache manager
	var cacheManager *cache.Manager
	if !noCache {
		cacheManager = cache.NewManager(cfg.Cache.Dir, cfg.Cache.TTLDays)
	}

	// Initialize pricing client
	client, err := pricing.NewClient(cfg, cacheManager, noCache)
	if err != nil {
		return fmt.Errorf("failed to create pricing client: %w", err)
	}

	// Fetch pricing data
	ctx := context.Background()
	catalog, err := client.FetchPricing(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch pricing: %w", err)
	}

	fmt.Printf("Retrieved pricing for %d instance/region/OS combinations\n", len(catalog.Items))

	// Render JSON
	fmt.Println("Rendering JSON output...")
	if err := render.RenderJSON(catalog, outputPath, cfg.Render.RoundDecimals); err != nil {
		return fmt.Errorf("failed to render JSON: %w", err)
	}

	// Render Markdown
	fmt.Println("Rendering Markdown output...")
	if err := render.RenderMarkdown(catalog, outputPath, cfg.Render.RoundDecimals, cfg.Render.SplitByInstance, cfg.Render.ShowNotes); err != nil {
		return fmt.Errorf("failed to render Markdown: %w", err)
	}

	fmt.Printf("\nPricing data exported to: %s\n", outputPath)
	fmt.Println("Files created:")
	fmt.Println("  - catalog.json")
	if cfg.Render.SplitByInstance {
		fmt.Println("  - <instance-type>.md (one per instance)")
	} else {
		fmt.Println("  - pricing.md")
	}

	return nil
}
