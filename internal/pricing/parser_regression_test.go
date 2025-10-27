package pricing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseRealAWSResponse_NoZeroDollarValues is a regression test to ensure
// the $0 Reserved Instance pricing bug doesn't reoccur
func TestParseRealAWSResponse_NoZeroDollarValues(t *testing.T) {
	// This is actual AWS Pricing API response structure for m5.metal
	realAWSResponse := `{
		"product": {
			"attributes": {
				"vcpu": "96",
				"memory": "384 GiB"
			}
		},
		"terms": {
			"OnDemand": {
				"YNZBC6A26TXAHW57.JRTCKXETXF": {
					"priceDimensions": {
						"YNZBC6A26TXAHW57.JRTCKXETXF.6YS6EN2CT7": {
							"unit": "Hrs",
							"description": "$5.76 per On Demand Linux m5.metal Instance Hour",
							"pricePerUnit": {"USD": "5.7600000000"}
						}
					}
				}
			},
			"Reserved": {
				"YNZBC6A26TXAHW57.6QCMYABX3D": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"YNZBC6A26TXAHW57.6QCMYABX3D.2TG2D8R56U": {
							"unit": "Quantity",
							"description": "Upfront Fee",
							"pricePerUnit": {"USD": "29669"}
						},
						"YNZBC6A26TXAHW57.6QCMYABX3D.6YS6EN2CT7": {
							"unit": "Hrs",
							"description": "USD 0.0 per Linux/UNIX (Amazon VPC), m5.metal reserved instance applied",
							"pricePerUnit": {"USD": "0.0000000000"}
						}
					}
				},
				"YNZBC6A26TXAHW57.HU7G6KETJZ": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "Partial Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"YNZBC6A26TXAHW57.HU7G6KETJZ.2TG2D8R56U": {
							"unit": "Quantity",
							"description": "Upfront Fee",
							"pricePerUnit": {"USD": "15137"}
						},
						"YNZBC6A26TXAHW57.HU7G6KETJZ.6YS6EN2CT7": {
							"unit": "Hrs",
							"description": "Hourly charge",
							"pricePerUnit": {"USD": "1.7280000000"}
						}
					}
				},
				"YNZBC6A26TXAHW57.4NA7Y494T4": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "No Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"YNZBC6A26TXAHW57.4NA7Y494T4.6YS6EN2CT7": {
							"unit": "Hrs",
							"description": "Hourly charge",
							"pricePerUnit": {"USD": "3.6290000000"}
						}
					}
				},
				"YNZBC6A26TXAHW57.VJWZNREJX2": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront",
						"OfferingClass": "convertible"
					},
					"priceDimensions": {
						"YNZBC6A26TXAHW57.VJWZNREJX2.2TG2D8R56U": {
							"unit": "Quantity",
							"description": "Upfront Fee",
							"pricePerUnit": {"USD": "37531"}
						},
						"YNZBC6A26TXAHW57.VJWZNREJX2.6YS6EN2CT7": {
							"unit": "Hrs",
							"pricePerUnit": {"USD": "0.0000000000"}
						}
					}
				}
			}
		}
	}`

	parsed, err := ParsePriceList(realAWSResponse)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	// Verify basic attributes
	assert.Equal(t, 96, parsed.VCpu)
	assert.Equal(t, 384.0, parsed.MemoryGiB)
	assert.Equal(t, 5.76, parsed.OnDemandRate)

	// CRITICAL: Verify we only got standard class RIs (convertible should be filtered)
	assert.Equal(t, 3, len(parsed.ReservedRates), "Should only have 3 standard RI options (convertible filtered out)")

	// CRITICAL REGRESSION TEST: Verify NO raw upfront prices in output
	// The bug was that upfront prices (29669, 15137) were returned as-is
	for termKey, rate := range parsed.ReservedRates {
		t.Run(termKey, func(t *testing.T) {
			// Upfront prices should be stored correctly (not normalized yet)
			if termKey == "1yr_AU" {
				assert.Equal(t, 29669.0, rate.UpfrontPrice, "Upfront price should be extracted")
				assert.Equal(t, 0.0, rate.HourlyPrice, "All Upfront has $0 hourly")
			}

			if termKey == "1yr_PU" {
				assert.Equal(t, 15137.0, rate.UpfrontPrice, "Partial Upfront should have upfront fee")
				assert.Equal(t, 1.728, rate.HourlyPrice, "Partial Upfront should have hourly rate")
			}

			if termKey == "1yr_NU" {
				assert.Equal(t, 0.0, rate.UpfrontPrice, "No Upfront should have $0 upfront")
				assert.Equal(t, 3.629, rate.HourlyPrice, "No Upfront should have hourly rate")
			}
		})
	}

	// Verify convertible was filtered out
	_, hasConvertible := parsed.ReservedRates["1yr_AU_convertible"]
	assert.False(t, hasConvertible, "Convertible offerings should be filtered out")
}

// TestNormalizedPricing_NoZeroValues tests the end-to-end normalization
// to ensure effective hourly rates are never $0
func TestNormalizedPricing_NoZeroValues(t *testing.T) {
	testCases := []struct {
		name         string
		upfront      float64
		hourly       float64
		years        int
		expectZero   bool
		expectedRate float64
	}{
		{
			name:         "1yr All Upfront should NOT be zero",
			upfront:      29669.0,
			hourly:       0.0,
			years:        1,
			expectZero:   false,
			expectedRate: 3.3868607, // 29669 / 8760 hours
		},
		{
			name:         "1yr Partial Upfront should NOT be zero",
			upfront:      15137.0,
			hourly:       1.728,
			years:        1,
			expectZero:   false,
			expectedRate: 3.456, // (15137 / 8760) + 1.728
		},
		{
			name:         "1yr No Upfront should NOT be zero",
			upfront:      0.0,
			hourly:       3.629,
			years:        1,
			expectZero:   false,
			expectedRate: 3.629,
		},
		{
			name:         "3yr All Upfront should NOT be zero",
			upfront:      56916.0,
			hourly:       0.0,
			years:        3,
			expectZero:   false,
			expectedRate: 2.1657534, // 56916 / 26280 hours
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalizedRate := NormalizeReservedRate(tc.upfront, tc.hourly, tc.years)

			// CRITICAL: Effective hourly rate should NEVER be zero for valid pricing
			if !tc.expectZero {
				assert.NotEqual(t, 0.0, normalizedRate,
					"Normalized rate should NOT be $0 - this was the original bug!")
			}

			// Verify it matches expected calculation
			assert.InDelta(t, tc.expectedRate, normalizedRate, 0.0001,
				"Normalized rate should match expected value")

			// CRITICAL: Should be a reasonable hourly rate (between $0.50 and $50/hour for metal instances)
			assert.Greater(t, normalizedRate, 0.5,
				"Normalized rate seems too low - possible calculation error")
			assert.Less(t, normalizedRate, 50.0,
				"Normalized rate seems too high - possible missing division")
		})
	}
}

// TestUnitFieldDetection ensures we correctly identify Quantity vs Hrs units
func TestUnitFieldDetection(t *testing.T) {
	testJSON := `{
		"product": {
			"attributes": {
				"vcpu": "96",
				"memory": "384 GiB"
			}
		},
		"terms": {
			"OnDemand": {},
			"Reserved": {
				"offer1": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"cryptic.key.2TG2D8R56U": {
							"unit": "Quantity",
							"description": "Upfront Fee",
							"pricePerUnit": {"USD": "12345"}
						},
						"cryptic.key.6YS6EN2CT7": {
							"unit": "Hrs",
							"description": "Hourly charge",
							"pricePerUnit": {"USD": "0"}
						}
					}
				}
			}
		}
	}`

	parsed, err := ParsePriceList(testJSON)
	require.NoError(t, err)

	rate := parsed.ReservedRates["1yr_AU"]
	assert.Equal(t, 12345.0, rate.UpfrontPrice,
		"Should identify Quantity unit as upfront price")
	assert.Equal(t, 0.0, rate.HourlyPrice,
		"Should identify Hrs unit as hourly price")
}

// TestOfferingClassFiltering ensures convertible RIs are filtered out
func TestOfferingClassFiltering(t *testing.T) {
	testJSON := `{
		"product": {
			"attributes": {
				"vcpu": "96",
				"memory": "384 GiB"
			}
		},
		"terms": {
			"OnDemand": {},
			"Reserved": {
				"standard_offer": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"dim1": {
							"unit": "Quantity",
							"pricePerUnit": {"USD": "1000"}
						}
					}
				},
				"convertible_offer": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront",
						"OfferingClass": "convertible"
					},
					"priceDimensions": {
						"dim2": {
							"unit": "Quantity",
							"pricePerUnit": {"USD": "2000"}
						}
					}
				}
			}
		}
	}`

	parsed, err := ParsePriceList(testJSON)
	require.NoError(t, err)

	// Should only have 1 RI (standard), not 2
	assert.Equal(t, 1, len(parsed.ReservedRates),
		"Should filter out convertible offerings")

	// Verify the standard offering is present
	rate, exists := parsed.ReservedRates["1yr_AU"]
	assert.True(t, exists, "Standard offering should be present")
	assert.Equal(t, "standard", rate.OfferingClass)
	assert.Equal(t, 1000.0, rate.UpfrontPrice)
}

// TestRegressionForLargeRawValues ensures we detect if normalization is skipped
func TestRegressionForLargeRawValues(t *testing.T) {
	// Simulate what happens when normalization is accidentally skipped
	// Raw upfront values like 29669, 56916 would appear as-is
	testRates := map[string]ReservedRate{
		"1yr_AU": {
			UpfrontPrice: 29669.0,
			HourlyPrice:  0.0,
		},
		"3yr_AU": {
			UpfrontPrice: 56916.0,
			HourlyPrice:  0.0,
		},
	}

	for termKey, rate := range testRates {
		t.Run(termKey, func(t *testing.T) {
			years := GetTermYears(termKey)
			normalizedRate := NormalizeReservedRate(rate.UpfrontPrice, rate.HourlyPrice, years)

			// CRITICAL: Normalized rate should be < $10/hour, not thousands
			assert.Less(t, normalizedRate, 10.0,
				"If rate is > $10/hour, normalization was likely skipped (bug detected!)")
			assert.Greater(t, normalizedRate, 0.0,
				"Normalized rate should not be zero (bug detected!)")
		})
	}
}
