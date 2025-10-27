package models

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCatalogValidation_NoZeroDollarValues is a validation test that can be
// run against real catalog.json output to detect the $0 pricing bug
func TestCatalogValidation_NoZeroDollarValues(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{
				InstanceType: "m5.metal",
				Region:       "us-east-1",
				OsLabel:      "Linux",
				OnDemandUSDH: 5.76,
				ReservedUSDH: map[string]float64{
					"1yr_AU": 3.3869,
					"1yr_PU": 3.456,
					"1yr_NU": 3.629,
					"3yr_AU": 2.1658,
					"3yr_PU": 2.304,
					"3yr_NU": 2.488,
				},
			},
		},
	}

	errors := ValidateCatalogPricing(catalog)
	assert.Empty(t, errors, "Test catalog should have no validation errors")
}

// TestCatalogValidation_DetectsZeroValues ensures our validation catches $0 bugs
func TestCatalogValidation_DetectsZeroValues(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{
				InstanceType: "m5.metal",
				Region:       "us-east-1",
				OsLabel:      "Linux",
				OnDemandUSDH: 5.76,
				ReservedUSDH: map[string]float64{
					"1yr_AU": 0.0, // BUG: This should NOT be zero!
					"1yr_PU": 3.456,
					"1yr_NU": 3.629,
				},
			},
		},
	}

	errors := ValidateCatalogPricing(catalog)
	assert.NotEmpty(t, errors, "Should detect zero dollar value")
	assert.Contains(t, errors[0], "m5.metal")
	assert.Contains(t, errors[0], "1yr_AU")
	assert.Contains(t, errors[0], "$0")
}

// TestCatalogValidation_DetectsUnnormalizedValues ensures we catch raw upfront prices
func TestCatalogValidation_DetectsUnnormalizedValues(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{
				InstanceType: "m5.metal",
				Region:       "us-east-1",
				OsLabel:      "Linux",
				OnDemandUSDH: 5.76,
				ReservedUSDH: map[string]float64{
					"1yr_AU": 29669.0, // BUG: This is the raw upfront price, not normalized!
					"1yr_PU": 3.456,
					"1yr_NU": 3.629,
				},
			},
		},
	}

	errors := ValidateCatalogPricing(catalog)
	assert.NotEmpty(t, errors, "Should detect unnormalized upfront price")
	assert.Contains(t, errors[0], "m5.metal")
	assert.Contains(t, errors[0], "1yr_AU")
	assert.Contains(t, errors[0], "suspiciously high")
}

// TestCatalogValidation_FromFile can load and validate an actual catalog.json file
func TestCatalogValidation_FromFile(t *testing.T) {
	// This test is skipped by default but can be run to validate real output
	catalogPath := "../../pricing/catalog.json"
	if _, err := os.Stat(catalogPath); os.IsNotExist(err) {
		t.Skip("No catalog.json found, skipping file validation")
	}

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Skip("Could not read catalog.json")
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatalf("Failed to parse catalog.json: %v", err)
	}

	errors := ValidateCatalogPricing(&catalog)
	if len(errors) > 0 {
		t.Errorf("Catalog validation failed with %d errors:", len(errors))
		for _, err := range errors {
			t.Errorf("  - %s", err)
		}
	}
}

// ValidateCatalogPricing checks a catalog for common pricing issues
func ValidateCatalogPricing(catalog *Catalog) []string {
	var errors []string

	for _, item := range catalog.Items {
		// Check On-Demand pricing
		if item.OnDemandUSDH <= 0 {
			errors = append(errors,
				formatError(item, "on-demand", "$0 or negative"))
		}

		// Check Reserved Instance pricing
		for termKey, rate := range item.ReservedUSDH {
			// CRITICAL: Detect $0 values (the original bug)
			if rate == 0.0 {
				errors = append(errors,
					formatError(item, termKey, "$0 value detected - this was the bug!"))
			}

			// CRITICAL: Detect unnormalized upfront prices
			// Metal instance RI rates should typically be $1-15/hour, not thousands
			if rate > 100.0 {
				errors = append(errors,
					formatError(item, termKey, "suspiciously high rate (>$100/hr) - may be unnormalized upfront price"))
			}

			// Detect suspiciously low rates (except for spot/free tier which don't apply to metal)
			if rate < 0.5 && rate > 0 {
				errors = append(errors,
					formatError(item, termKey, "suspiciously low rate (<$0.50/hr) - possible calculation error"))
			}

			// Sanity check: RI should be cheaper than On-Demand
			// Allow some tolerance for rounding and unusual cases
			if rate > item.OnDemandUSDH*1.5 {
				errors = append(errors,
					formatError(item, termKey, "RI rate higher than On-Demand - suspicious"))
			}
		}

		// Check for missing RI data
		// Standard config expects 6 RI options (2 terms × 3 purchases)
		if len(item.ReservedUSDH) < 6 && len(item.ReservedUSDH) > 0 {
			// This is a warning, not an error (some instances may not have all options)
			// But in the bug case, we might have partial data
			if len(item.ReservedUSDH) < 3 {
				errors = append(errors,
					formatError(item, "reserved", "very few RI options available"))
			}
		}
	}

	return errors
}

func formatError(item PricePoint, priceType, issue string) string {
	return sprintf("%s/%s/%s [%s]: %s",
		item.Region, item.InstanceType, item.OsLabel, priceType, issue)
}

// Helper function to format strings (simple implementation)
func sprintf(format string, args ...interface{}) string {
	// Simple sprintf implementation for the test
	result := format
	for _, arg := range args {
		// This is a simplified version - in real code use fmt.Sprintf
		switch v := arg.(type) {
		case string:
			result = replaceFirst(result, "%s", v)
		case float64:
			result = replaceFirst(result, "%f", floatToString(v))
		}
	}
	return result
}

func replaceFirst(s, old, new string) string {
	// Simple implementation - in production use strings.Replace
	for i := 0; i < len(s)-len(old)+1; i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func floatToString(f float64) string {
	// Simple float to string - in production use strconv
	return "float"
}
