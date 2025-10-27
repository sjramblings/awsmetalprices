package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricePoint_AddNote(t *testing.T) {
	pp := &PricePoint{
		Region:       "us-east-1",
		InstanceType: "m5.metal",
	}

	pp.AddNote("First note")
	assert.Equal(t, 1, len(pp.Notes))
	assert.Equal(t, "First note", pp.Notes[0])

	pp.AddNote("Second note")
	assert.Equal(t, 2, len(pp.Notes))
	assert.Equal(t, "Second note", pp.Notes[1])
}

func TestCatalog_SortByInstance(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{InstanceType: "m5.metal", Region: "us-west-2", OsLabel: "Windows"},
			{InstanceType: "c5.metal", Region: "us-east-1", OsLabel: "Linux"},
			{InstanceType: "m5.metal", Region: "us-east-1", OsLabel: "Linux"},
			{InstanceType: "m5.metal", Region: "us-east-1", OsLabel: "Windows"},
		},
	}

	catalog.SortByInstance()

	// Should be sorted by instance, then region, then OS
	assert.Equal(t, "c5.metal", catalog.Items[0].InstanceType)
	assert.Equal(t, "m5.metal", catalog.Items[1].InstanceType)
	assert.Equal(t, "us-east-1", catalog.Items[1].Region)
	assert.Equal(t, "Linux", catalog.Items[1].OsLabel)
	assert.Equal(t, "Windows", catalog.Items[2].OsLabel)
}

func TestCatalog_FilterByRegion(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{Region: "us-east-1", InstanceType: "m5.metal"},
			{Region: "us-west-2", InstanceType: "c5.metal"},
			{Region: "us-east-1", InstanceType: "c5.metal"},
		},
	}

	filtered := catalog.FilterByRegion("us-east-1")
	assert.Equal(t, 2, len(filtered))
	for _, item := range filtered {
		assert.Equal(t, "us-east-1", item.Region)
	}
}

func TestCatalog_FilterByOS(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{OsLabel: "Linux", InstanceType: "m5.metal"},
			{OsLabel: "Windows", InstanceType: "c5.metal"},
			{OsLabel: "Linux", InstanceType: "c5.metal"},
		},
	}

	filtered := catalog.FilterByOS("Linux")
	assert.Equal(t, 2, len(filtered))
	for _, item := range filtered {
		assert.Equal(t, "Linux", item.OsLabel)
	}
}

func TestCatalog_FilterByInstance(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{InstanceType: "m5.metal", Region: "us-east-1"},
			{InstanceType: "c5.metal", Region: "us-west-2"},
			{InstanceType: "m5.metal", Region: "us-west-2"},
		},
	}

	filtered := catalog.FilterByInstance("m5.metal")
	assert.Equal(t, 2, len(filtered))
	for _, item := range filtered {
		assert.Equal(t, "m5.metal", item.InstanceType)
	}
}

func TestCatalog_Filter(t *testing.T) {
	catalog := &Catalog{
		Items: []PricePoint{
			{InstanceType: "m5.metal", Region: "us-east-1", OsLabel: "Linux"},
			{InstanceType: "m5.metal", Region: "us-east-1", OsLabel: "Windows"},
			{InstanceType: "m5.metal", Region: "us-west-2", OsLabel: "Linux"},
			{InstanceType: "c5.metal", Region: "us-east-1", OsLabel: "Linux"},
		},
	}

	t.Run("filter by instance only", func(t *testing.T) {
		filtered := catalog.Filter("m5.metal", "", "")
		assert.Equal(t, 3, len(filtered))
	})

	t.Run("filter by region only", func(t *testing.T) {
		filtered := catalog.Filter("", "us-east-1", "")
		assert.Equal(t, 3, len(filtered))
	})

	t.Run("filter by OS only", func(t *testing.T) {
		filtered := catalog.Filter("", "", "Linux")
		assert.Equal(t, 3, len(filtered))
	})

	t.Run("filter by all criteria", func(t *testing.T) {
		filtered := catalog.Filter("m5.metal", "us-east-1", "Linux")
		assert.Equal(t, 1, len(filtered))
		assert.Equal(t, "m5.metal", filtered[0].InstanceType)
		assert.Equal(t, "us-east-1", filtered[0].Region)
		assert.Equal(t, "Linux", filtered[0].OsLabel)
	})

	t.Run("filter with no matches", func(t *testing.T) {
		filtered := catalog.Filter("nonexistent", "", "")
		assert.Equal(t, 0, len(filtered))
	})
}

func TestJSON_Marshaling(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	catalog := &Catalog{
		GeneratedAt: now,
		Currency:    "USD",
		Items: []PricePoint{
			{
				Region:          "us-east-1",
				InstanceType:    "m5.metal",
				VCpu:            96,
				MemoryGiB:       384.0,
				OperatingSystem: "Linux",
				PreInstalledSw:  "NA",
				LicenseModel:    "No License required",
				OsLabel:         "Linux",
				OnDemandUSDH:    5.424,
				ReservedUSDH: map[string]float64{
					"1yr_AU": 3.123,
					"1yr_PU": 3.456,
				},
				Notes: []string{"Test note"},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(catalog)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Catalog
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, catalog.Currency, decoded.Currency)
	assert.Equal(t, 1, len(decoded.Items))
	assert.Equal(t, catalog.Items[0].InstanceType, decoded.Items[0].InstanceType)
	assert.Equal(t, catalog.Items[0].VCpu, decoded.Items[0].VCpu)
	assert.Equal(t, catalog.Items[0].OnDemandUSDH, decoded.Items[0].OnDemandUSDH)
	assert.Equal(t, 2, len(decoded.Items[0].ReservedUSDH))
}
