package pricing

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
	"github.com/sjramblings/awsmetalprices/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFilters(t *testing.T) {
	os := config.OsSpec{
		OperatingSystem: "Linux",
		PreInstalledSw:  "NA",
		LicenseModel:    "No License required",
		Label:           "Linux",
	}

	filters := BuildFilters("us-east-1", "m5.metal", os)

	assert.Equal(t, 8, len(filters))

	// Verify filter types and values
	filterMap := make(map[string]string)
	for _, filter := range filters {
		if filter.Field != nil && filter.Value != nil {
			filterMap[*filter.Field] = *filter.Value
		}
	}

	assert.Equal(t, "AmazonEC2", filterMap["ServiceCode"])
	assert.Equal(t, "m5.metal", filterMap["instanceType"])
	assert.Equal(t, "Linux", filterMap["operatingSystem"])
	assert.Equal(t, "NA", filterMap["preInstalledSw"])
	assert.Equal(t, "No License required", filterMap["licenseModel"])
	assert.Equal(t, "Shared", filterMap["tenancy"])
	assert.Equal(t, "Used", filterMap["capacitystatus"])
	assert.Equal(t, "US East (N. Virginia)", filterMap["location"])
}

func TestRegionToLocation(t *testing.T) {
	tests := []struct {
		region   string
		expected string
	}{
		{"us-east-1", "US East (N. Virginia)"},
		{"us-west-2", "US West (Oregon)"},
		{"ap-southeast-2", "Asia Pacific (Sydney)"},
		{"eu-central-1", "EU (Frankfurt)"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			location, exists := RegionToLocation[tt.region]
			assert.True(t, exists)
			assert.Equal(t, tt.expected, location)
		})
	}
}

func TestNormalizeReservedRate(t *testing.T) {
	tests := []struct {
		name     string
		upfront  float64
		hourly   float64
		years    int
		expected float64
	}{
		{
			name:     "1yr All Upfront",
			upfront:  10000.00,
			hourly:   0.00,
			years:    1,
			expected: 10000.00 / (365 * 24),
		},
		{
			name:     "1yr Partial Upfront",
			upfront:  5000.00,
			hourly:   0.50,
			years:    1,
			expected: (5000.00 / (365 * 24)) + 0.50,
		},
		{
			name:     "1yr No Upfront",
			upfront:  0.00,
			hourly:   1.20,
			years:    1,
			expected: 1.20,
		},
		{
			name:     "3yr All Upfront",
			upfront:  20000.00,
			hourly:   0.00,
			years:    3,
			expected: 20000.00 / (3 * 365 * 24),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeReservedRate(tt.upfront, tt.hourly, tt.years)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestGetTermYears(t *testing.T) {
	tests := []struct {
		termKey  string
		expected int
	}{
		{"1yr_AU", 1},
		{"1yr_PU", 1},
		{"1yr_NU", 1},
		{"3yr_AU", 3},
		{"3yr_PU", 3},
		{"3yr_NU", 3},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.termKey, func(t *testing.T) {
			result := GetTermYears(tt.termKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePriceList(t *testing.T) {
	// Sample AWS pricing JSON (matches actual AWS API structure)
	sampleJSON := `{
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
							"pricePerUnit": {
								"USD": "5.424"
							}
						}
					}
				}
			},
			"Reserved": {
				"offer2": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"dim1.2TG2D8R56U": {
							"unit": "Quantity",
							"description": "Upfront Fee",
							"pricePerUnit": {
								"USD": "10000"
							}
						},
						"dim1.6YS6EN2CT7": {
							"unit": "Hrs",
							"description": "Hourly charge",
							"pricePerUnit": {
								"USD": "0"
							}
						}
					}
				},
				"offer3": {
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "No Upfront",
						"OfferingClass": "standard"
					},
					"priceDimensions": {
						"dim2": {
							"unit": "Hrs",
							"pricePerUnit": {
								"USD": "1.2"
							}
						}
					}
				}
			}
		}
	}`

	parsed, err := ParsePriceList(sampleJSON)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	assert.Equal(t, 96, parsed.VCpu)
	assert.Equal(t, 384.0, parsed.MemoryGiB)
	assert.Equal(t, 5.424, parsed.OnDemandRate)

	assert.Equal(t, 2, len(parsed.ReservedRates))

	rate1yr := parsed.ReservedRates["1yr_AU"]
	assert.Equal(t, "1yr", rate1yr.Term)
	assert.Equal(t, "All Upfront", rate1yr.PurchaseType)
	assert.Equal(t, 10000.0, rate1yr.UpfrontPrice)
	assert.Equal(t, 0.0, rate1yr.HourlyPrice)

	rateNU := parsed.ReservedRates["1yr_NU"]
	assert.Equal(t, "No Upfront", rateNU.PurchaseType)
	assert.Equal(t, 1.2, rateNU.HourlyPrice)
}

func TestMapReservedTermKey(t *testing.T) {
	tests := []struct {
		term           string
		purchaseOption string
		offeringClass  string
		expected       string
	}{
		{"1yr", "All Upfront", "standard", "1yr_AU"},
		{"1yr", "Partial Upfront", "standard", "1yr_PU"},
		{"1yr", "No Upfront", "standard", "1yr_NU"},
		{"3yr", "All Upfront", "standard", "3yr_AU"},
		{"3yr", "Partial Upfront", "standard", "3yr_PU"},
		{"3yr", "No Upfront", "standard", "3yr_NU"},
		{"1yr", "All Upfront", "convertible", ""}, // Convertible should be filtered out
		{"invalid", "All Upfront", "standard", ""},
		{"1yr", "Invalid", "standard", ""},
	}

	for _, tt := range tests {
		t.Run(tt.term+"_"+tt.purchaseOption+"_"+tt.offeringClass, func(t *testing.T) {
			result := mapReservedTermKey(tt.term, tt.purchaseOption, tt.offeringClass)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterTypes(t *testing.T) {
	os := config.OsSpec{
		OperatingSystem: "Windows",
		PreInstalledSw:  "NA",
		LicenseModel:    "No License required",
		Label:           "Windows",
	}

	filters := BuildFilters("us-west-2", "c5.metal", os)

	for _, filter := range filters {
		assert.Equal(t, types.FilterTypeTermMatch, filter.Type)
	}
}
