package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configPath string
	version    = "1.0.0"
)

var rootCmd = &cobra.Command{
	Use:   "awsmetalprices",
	Short: "AWS EC2 Metal Instance Pricing CLI",
	Long: `A CLI tool to retrieve and display AWS EC2 pricing for metal instances.
Supports both On-Demand and Reserved Instance pricing for Linux and Windows.`,
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "Path to configuration file")

	// Add subcommands
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(printCmd)
	rootCmd.AddCommand(diffCmd)
}

// Execute runs the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}
