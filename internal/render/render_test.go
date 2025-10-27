package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sjramblings/awsmetalprices/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCatalog() *models.Catalog {
	return &models.Catalog{
		GeneratedAt: time.Date(2025, 10, 20, 0, 0, 0, 0, time.UTC),
		Currency:    "USD",
		Items: []models.PricePoint{
			{
				Region:       "us-east-1",
				InstanceType: "m5.metal",
				VCpu:         96,
				MemoryGiB:    384.0,
				OsLabel:      "Linux",
				OnDemandUSDH: 5.424,
				ReservedUSDH: map[string]float64{
					"1yr_AU": 3.1234,
					"1yr_PU": 3.2222,
					"1yr_NU": 3.3333,
					"3yr_AU": 2.4444,
					"3yr_PU": 2.5555,
					"3yr_NU": 2.6666,
				},
			},
			{
				Region:       "us-east-1",
				InstanceType: "m5.metal",
				VCpu:         96,
				MemoryGiB:    384.0,
				OsLabel:      "Windows",
				OnDemandUSDH: 6.789,
				ReservedUSDH: map[string]float64{
					"1yr_AU": 4.1111,
					"1yr_PU": 4.2222,
					"1yr_NU": 4.3333,
					"3yr_AU": 3.4444,
					"3yr_PU": 3.5555,
					"3yr_NU": 3.6666,
				},
			},
			{
				Region:       "us-west-2",
				InstanceType: "c5.metal",
				VCpu:         96,
				MemoryGiB:    192.0,
				OsLabel:      "Linux",
				OnDemandUSDH: 4.08,
				ReservedUSDH: map[string]float64{
					"1yr_AU": 2.5,
					"1yr_PU": 2.6,
					"1yr_NU": 2.7,
					"3yr_AU": 1.8,
					"3yr_PU": 1.9,
					"3yr_NU": 2.0,
				},
			},
		},
	}
}

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		value    float64
		decimals int
		expected float64
	}{
		{3.14159, 2, 3.14},
		{3.14159, 4, 3.1416},
		{5.424, 2, 5.42},
		{5.424, 0, 5.0},
		{2.5555, 3, 2.556},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := roundFloat(tt.value, tt.decimals)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderJSON(t *testing.T) {
	tmpDir := t.TempDir()
	catalog := createTestCatalog()

	err := RenderJSON(catalog, tmpDir, 4)
	require.NoError(t, err)

	// Verify file was created
	filename := filepath.Join(tmpDir, "catalog.json")
	_, err = os.Stat(filename)
	require.NoError(t, err)

	// Read and verify content
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "\"generatedAt\"")
	assert.Contains(t, content, "\"currency\": \"USD\"")
	assert.Contains(t, content, "\"instanceType\": \"m5.metal\"")
	assert.Contains(t, content, "5.424")
}

func TestRenderMarkdown_Combined(t *testing.T) {
	tmpDir := t.TempDir()
	catalog := createTestCatalog()

	err := RenderMarkdown(catalog, tmpDir, 4, false, true)
	require.NoError(t, err)

	// Verify file was created
	filename := filepath.Join(tmpDir, "pricing.md")
	_, err = os.Stat(filename)
	require.NoError(t, err)

	// Read and verify content
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# EC2 Metal Pricing")
	assert.Contains(t, content, "2025-10-20")
	assert.Contains(t, content, "## m5.metal")
	assert.Contains(t, content, "### Linux")
	assert.Contains(t, content, "### Windows")
	assert.Contains(t, content, "## c5.metal")
	assert.Contains(t, content, "| Region | vCPU | Mem (GiB) | On-Demand |")
	assert.Contains(t, content, "| us-east-1 |")
	assert.Contains(t, content, "_Notes_:")
}

func TestRenderMarkdown_Split(t *testing.T) {
	tmpDir := t.TempDir()
	catalog := createTestCatalog()

	err := RenderMarkdown(catalog, tmpDir, 4, true, false)
	require.NoError(t, err)

	// Verify separate files were created
	m5File := filepath.Join(tmpDir, "m5_metal.md")
	c5File := filepath.Join(tmpDir, "c5_metal.md")

	_, err = os.Stat(m5File)
	require.NoError(t, err)

	_, err = os.Stat(c5File)
	require.NoError(t, err)

	// Read m5.metal file
	data, err := os.ReadFile(m5File)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# m5.metal")
	assert.Contains(t, content, "## Linux")
	assert.Contains(t, content, "## Windows")
	assert.NotContains(t, content, "c5.metal")
	assert.NotContains(t, content, "_Notes_:") // showNotes is false
}

func TestFormatRIColumn(t *testing.T) {
	tests := []struct {
		name     string
		prices   map[string]float64
		term     string
		expected string
	}{
		{
			name: "1yr with all values",
			prices: map[string]float64{
				"1yr_AU": 3.1234,
				"1yr_PU": 3.2222,
				"1yr_NU": 3.3333,
			},
			term:     "1yr",
			expected: "3.1234 / 3.2222 / 3.3333",
		},
		{
			name: "3yr with all values",
			prices: map[string]float64{
				"3yr_AU": 2.4444,
				"3yr_PU": 2.5555,
				"3yr_NU": 2.6666,
			},
			term:     "3yr",
			expected: "2.4444 / 2.5555 / 2.6666",
		},
		{
			name:     "missing values",
			prices:   map[string]float64{},
			term:     "1yr",
			expected: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRIColumn(tt.prices, tt.term)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGroupByInstance(t *testing.T) {
	catalog := createTestCatalog()
	groups := groupByInstance(catalog.Items)

	assert.Equal(t, 2, len(groups))
	assert.Equal(t, 2, len(groups["m5.metal"]))
	assert.Equal(t, 1, len(groups["c5.metal"]))
}

func TestGroupByOS(t *testing.T) {
	catalog := createTestCatalog()
	groups := groupByOS(catalog.Items)

	assert.Equal(t, 2, len(groups))
	assert.Equal(t, 2, len(groups["Linux"]))
	assert.Equal(t, 1, len(groups["Windows"]))
}

func TestRenderTable(t *testing.T) {
	items := []models.PricePoint{
		{
			Region:       "us-east-1",
			VCpu:         96,
			MemoryGiB:    384.0,
			OnDemandUSDH: 5.424,
			ReservedUSDH: map[string]float64{
				"1yr_AU": 3.0,
				"1yr_PU": 3.5,
				"1yr_NU": 4.0,
				"3yr_AU": 2.0,
				"3yr_PU": 2.5,
				"3yr_NU": 3.0,
			},
		},
	}

	table := renderTable(items)

	assert.Contains(t, table, "| Region | vCPU | Mem (GiB) | On-Demand |")
	assert.Contains(t, table, "| us-east-1 | 96 | 384.0 | 5.4240 |")
	assert.Contains(t, table, "3.0000 / 3.5000 / 4.0000")
	assert.Contains(t, table, "2.0000 / 2.5000 / 3.0000")

	// Count header separator lines
	assert.True(t, strings.Contains(table, "|--------|"))
}

func TestRoundCatalog(t *testing.T) {
	catalog := createTestCatalog()
	rounded := roundCatalog(catalog, 2)

	assert.Equal(t, len(catalog.Items), len(rounded.Items))
	assert.Equal(t, 5.42, rounded.Items[0].OnDemandUSDH)
	assert.Equal(t, 3.12, rounded.Items[0].ReservedUSDH["1yr_AU"])
}
