package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sjramblings/awsmetalprices/internal/models"
)

// RenderMarkdown generates Markdown pricing tables
func RenderMarkdown(catalog *models.Catalog, outputPath string, roundDecimals int, splitByInstance bool, showNotes bool) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	rounded := roundCatalog(catalog, roundDecimals)

	if splitByInstance {
		return renderSplitMarkdown(rounded, outputPath, showNotes)
	}

	return renderCombinedMarkdown(rounded, outputPath, showNotes)
}

// renderCombinedMarkdown generates a single Markdown file with all instances
func renderCombinedMarkdown(catalog *models.Catalog, outputPath string, showNotes bool) error {
	var sb strings.Builder

	date := catalog.GeneratedAt.Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("# EC2 Metal Pricing — Linux & Windows — %s\n\n", date))

	// Group by instance type
	instanceGroups := groupByInstance(catalog.Items)

	// Sort instance types
	instanceTypes := make([]string, 0, len(instanceGroups))
	for instanceType := range instanceGroups {
		instanceTypes = append(instanceTypes, instanceType)
	}
	sort.Strings(instanceTypes)

	for _, instanceType := range instanceTypes {
		items := instanceGroups[instanceType]

		sb.WriteString(fmt.Sprintf("## %s\n\n", instanceType))

		// Group by OS
		osGroups := groupByOS(items)

		// Sort OS labels
		osLabels := make([]string, 0, len(osGroups))
		for osLabel := range osGroups {
			osLabels = append(osLabels, osLabel)
		}
		sort.Strings(osLabels)

		for _, osLabel := range osLabels {
			osItems := osGroups[osLabel]
			sb.WriteString(fmt.Sprintf("### %s\n\n", osLabel))
			sb.WriteString(renderTable(osItems))
			sb.WriteString("\n")
		}
	}

	if showNotes {
		sb.WriteString("_Notes_: Linux and Windows public list prices. No SQL Server editions. No mac*. No Spot.\n")
	}

	filename := filepath.Join(outputPath, "pricing.md")
	return os.WriteFile(filename, []byte(sb.String()), 0644)
}

// renderSplitMarkdown generates separate Markdown files per instance type
func renderSplitMarkdown(catalog *models.Catalog, outputPath string, showNotes bool) error {
	instanceGroups := groupByInstance(catalog.Items)

	for instanceType, items := range instanceGroups {
		var sb strings.Builder

		date := catalog.GeneratedAt.Format("2006-01-02")
		sb.WriteString(fmt.Sprintf("# %s — Pricing — %s\n\n", instanceType, date))

		// Group by OS
		osGroups := groupByOS(items)

		// Sort OS labels
		osLabels := make([]string, 0, len(osGroups))
		for osLabel := range osGroups {
			osLabels = append(osLabels, osLabel)
		}
		sort.Strings(osLabels)

		for _, osLabel := range osLabels {
			osItems := osGroups[osLabel]
			sb.WriteString(fmt.Sprintf("## %s\n\n", osLabel))
			sb.WriteString(renderTable(osItems))
			sb.WriteString("\n")
		}

		if showNotes {
			sb.WriteString("_Notes_: Linux and Windows public list prices. No SQL Server editions. No mac*. No Spot.\n")
		}

		// Sanitize filename
		safeFilename := strings.ReplaceAll(instanceType, ".", "_")
		filename := filepath.Join(outputPath, fmt.Sprintf("%s.md", safeFilename))
		if err := os.WriteFile(filename, []byte(sb.String()), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

// renderTable generates a Markdown table for a set of price points
func renderTable(items []models.PricePoint) string {
	var sb strings.Builder

	sb.WriteString("| Region | vCPU | Mem (GiB) | On-Demand | 1yr RI (A / P / N) | 3yr RI (A / P / N) |\n")
	sb.WriteString("|--------|------|-----------|------------|--------------------|--------------------||\n")

	// Sort by region
	sort.Slice(items, func(i, j int) bool {
		return items[i].Region < items[j].Region
	})

	for _, item := range items {
		ri1yr := formatRIColumn(item.ReservedUSDH, "1yr")
		ri3yr := formatRIColumn(item.ReservedUSDH, "3yr")

		sb.WriteString(fmt.Sprintf("| %s | %d | %.1f | %.4f | %s | %s |\n",
			item.Region,
			item.VCpu,
			item.MemoryGiB,
			item.OnDemandUSDH,
			ri1yr,
			ri3yr,
		))
	}

	return sb.String()
}

// formatRIColumn formats the RI pricing for a specific term (1yr or 3yr)
func formatRIColumn(prices map[string]float64, term string) string {
	au := prices[term+"_AU"]
	pu := prices[term+"_PU"]
	nu := prices[term+"_NU"]

	if au == 0 && pu == 0 && nu == 0 {
		return "N/A"
	}

	return fmt.Sprintf("%.4f / %.4f / %.4f", au, pu, nu)
}

// groupByInstance groups price points by instance type
func groupByInstance(items []models.PricePoint) map[string][]models.PricePoint {
	groups := make(map[string][]models.PricePoint)
	for _, item := range items {
		groups[item.InstanceType] = append(groups[item.InstanceType], item)
	}
	return groups
}

// groupByOS groups price points by OS label
func groupByOS(items []models.PricePoint) map[string][]models.PricePoint {
	groups := make(map[string][]models.PricePoint)
	for _, item := range items {
		groups[item.OsLabel] = append(groups[item.OsLabel], item)
	}
	return groups
}
