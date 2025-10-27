// +build integration

package pricing

import (
	"testing"

	"github.com/sjramblings/awsmetalprices/internal/config"
	"github.com/sjramblings/awsmetalprices/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndPricing_NoZeroValues tests the complete pipeline from AWS response
// through parsing, normalization, and final PricePoint creation
func TestEndToEndPricing_NoZeroValues(t *testing.T) {
	// Real AWS response for m5.metal
	awsResponse := `{
		"product": {
			"attributes": {
				"vcpu": "96",
				"memory": "384 GiB"
			}
		},
		"terms": {
			"OnDemand": {
				"offer1": {
					"priceDimensions": {
						"dim1": {
							"unit": "Hrs",
							"pricePerUnit": {"USD": "5.76"}
						}
					}
				}
			},
			"Reserved": {
				"offer_1yr_AU": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"upfront": {
							"unit": "Quantity",
							"pricePerUnit": {"USD": "29669"}
						},
						"hourly": {
							"unit": "Hrs",
							"pricePerUnit": {"USD": "0"}
						}
					}
				},
				"offer_1yr_PU": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "Partial Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"upfront": {
							"unit": "Quantity",
							"pricePerUnit": {"USD": "15137"}
						},
						"hourly": {
							"unit": "Hrs",
							"pricePerUnit": {"USD": "1.728"}
						}
					}
				},
				"offer_1yr_NU": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "No Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"hourly": {
							"unit": "Hrs",
							"pricePerUnit": {"USD": "3.629"}
						}
					}
				}
			}
		}
	}`

	// Parse the response
	parsed, err := ParsePriceList(awsResponse)
	require.NoError(t, err)

	// Simulate what the client does: normalize and create PricePoint
	pricePoint := &models.PricePoint{
		Region:          "us-east-1",
		InstanceType:    "m5.metal",
		VCpu:            parsed.VCpu,
		MemoryGiB:       parsed.MemoryGiB,
		OperatingSystem: "Linux",
		OsLabel:         "Linux",
		OnDemandUSDH:    parsed.OnDemandRate,
		ReservedUSDH:    make(map[string]float64),
	}

	// Normalize reserved pricing (this is what client.go does)
	for termKey, rate := range parsed.ReservedRates {
		years := GetTermYears(termKey)
		if years > 0 {
			effectiveRate := NormalizeReservedRate(rate.UpfrontPrice, rate.HourlyPrice, years)
			pricePoint.ReservedUSDH[termKey] = effectiveRate
		}
	}

	// CRITICAL ASSERTIONS: Verify the final output has no $0 values
	t.Run("NoZeroValues", func(t *testing.T) {
		for termKey, rate := range pricePoint.ReservedUSDH {
			assert.NotEqual(t, 0.0, rate,
				"Term %s should NOT have $0 rate - this was the bug!", termKey)
			assert.Greater(t, rate, 0.0,
				"Term %s rate should be positive", termKey)
		}
	})

	t.Run("NoRawUpfrontValues", func(t *testing.T) {
		for termKey, rate := range pricePoint.ReservedUSDH {
			// Raw upfront values are in thousands (29669, 15137)
			// Normalized hourly rates should be single digits
			assert.Less(t, rate, 100.0,
				"Term %s rate %f is too high - normalization may have been skipped!", termKey, rate)
		}
	})

	t.Run("ReasonableRates", func(t *testing.T) {
		// 1yr All Upfront: 29669 / 8760 = 3.3869
		assert.InDelta(t, 3.3869, pricePoint.ReservedUSDH["1yr_AU"], 0.001)

		// 1yr Partial Upfront: (15137 / 8760) + 1.728 = 3.456
		assert.InDelta(t, 3.456, pricePoint.ReservedUSDH["1yr_PU"], 0.001)

		// 1yr No Upfront: just the hourly rate
		assert.InDelta(t, 3.629, pricePoint.ReservedUSDH["1yr_NU"], 0.001)
	})

	t.Run("AllExpectedTermsPresent", func(t *testing.T) {
		assert.Equal(t, 3, len(pricePoint.ReservedUSDH),
			"Should have all 3 RI terms")
		assert.Contains(t, pricePoint.ReservedUSDH, "1yr_AU")
		assert.Contains(t, pricePoint.ReservedUSDH, "1yr_PU")
		assert.Contains(t, pricePoint.ReservedUSDH, "1yr_NU")
	})
}

// TestConfigReservedTermValidation ensures the config specifies expected RI terms
func TestConfigReservedTermValidation(t *testing.T) {
	// This test ensures our config expects the right number of RI options
	// If this changes, our validation logic needs to be updated
	cfg := &config.Config{
		Reserved: config.ReservedConfig{
			Classes:   []string{"standard"},
			Terms:     []string{"1yr", "3yr"},
			Purchases: []string{"All Upfront", "Partial Upfront", "No Upfront"},
		},
	}

	expectedTerms := len(cfg.Reserved.Terms) * len(cfg.Reserved.Purchases)
	assert.Equal(t, 6, expectedTerms,
		"Config expects 6 RI options (2 terms × 3 purchases)")
}
