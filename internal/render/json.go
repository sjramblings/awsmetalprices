package render

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/sjramblings/awsmetalprices/internal/models"
)

// RenderJSON exports the catalog to a JSON file
func RenderJSON(catalog *models.Catalog, outputPath string, roundDecimals int) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Round prices
	rounded := roundCatalog(catalog, roundDecimals)

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(rounded, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	// Write to file
	filename := filepath.Join(outputPath, "catalog.json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// roundCatalog creates a copy of the catalog with rounded prices
func roundCatalog(catalog *models.Catalog, decimals int) *models.Catalog {
	rounded := &models.Catalog{
		GeneratedAt: catalog.GeneratedAt,
		Currency:    catalog.Currency,
		Items:       make([]models.PricePoint, len(catalog.Items)),
	}

	for i, item := range catalog.Items {
		rounded.Items[i] = models.PricePoint{
			Region:          item.Region,
			InstanceType:    item.InstanceType,
			VCpu:            item.VCpu,
			MemoryGiB:       item.MemoryGiB,
			OperatingSystem: item.OperatingSystem,
			PreInstalledSw:  item.PreInstalledSw,
			LicenseModel:    item.LicenseModel,
			OsLabel:         item.OsLabel,
			OnDemandUSDH:    roundFloat(item.OnDemandUSDH, decimals),
			ReservedUSDH:    make(map[string]float64),
			Notes:           item.Notes,
		}

		for key, value := range item.ReservedUSDH {
			rounded.Items[i].ReservedUSDH[key] = roundFloat(value, decimals)
		}
	}

	return rounded
}

// roundFloat rounds a float to the specified number of decimal places
func roundFloat(val float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(val*multiplier) / multiplier
}
