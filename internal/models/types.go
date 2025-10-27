package models

import (
	"sort"
	"time"
)

// OsSpec defines operating system and licensing configuration
// This is duplicated from config package to avoid circular dependencies
type OsSpec struct {
	OperatingSystem string `json:"operatingSystem"`
	PreInstalledSw  string `json:"preInstalledSw"`
	LicenseModel    string `json:"licenseModel"`
	Label           string `json:"label"`
}

// PricePoint represents pricing data for a specific instance/region/OS combination
type PricePoint struct {
	Region          string             `json:"region"`
	InstanceType    string             `json:"instanceType"`
	VCpu            int                `json:"vCpu"`
	MemoryGiB       float64            `json:"memoryGiB"`
	OperatingSystem string             `json:"operatingSystem"`
	PreInstalledSw  string             `json:"preInstalledSw"`
	LicenseModel    string             `json:"licenseModel"`
	OsLabel         string             `json:"osLabel"`
	OnDemandUSDH    float64            `json:"onDemandUSDH"`
	ReservedUSDH    map[string]float64 `json:"reservedUSDH"`
	Notes           []string           `json:"notes,omitempty"`
}

// Catalog represents the complete pricing dataset
type Catalog struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Currency    string       `json:"currency"`
	Items       []PricePoint `json:"items"`
}

// AddNote appends a note to the price point
func (p *PricePoint) AddNote(note string) {
	if p.Notes == nil {
		p.Notes = make([]string, 0)
	}
	p.Notes = append(p.Notes, note)
}

// SortByInstance sorts catalog items by instance type, then region, then OS
func (c *Catalog) SortByInstance() {
	sort.Slice(c.Items, func(i, j int) bool {
		if c.Items[i].InstanceType != c.Items[j].InstanceType {
			return c.Items[i].InstanceType < c.Items[j].InstanceType
		}
		if c.Items[i].Region != c.Items[j].Region {
			return c.Items[i].Region < c.Items[j].Region
		}
		return c.Items[i].OsLabel < c.Items[j].OsLabel
	})
}

// FilterByRegion returns all price points for a specific region
func (c *Catalog) FilterByRegion(region string) []PricePoint {
	var filtered []PricePoint
	for _, item := range c.Items {
		if item.Region == region {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// FilterByOS returns all price points for a specific operating system label
func (c *Catalog) FilterByOS(osLabel string) []PricePoint {
	var filtered []PricePoint
	for _, item := range c.Items {
		if item.OsLabel == osLabel {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// FilterByInstance returns all price points for a specific instance type
func (c *Catalog) FilterByInstance(instanceType string) []PricePoint {
	var filtered []PricePoint
	for _, item := range c.Items {
		if item.InstanceType == instanceType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// Filter returns price points matching all provided criteria
func (c *Catalog) Filter(instanceType, region, osLabel string) []PricePoint {
	var filtered []PricePoint
	for _, item := range c.Items {
		if instanceType != "" && item.InstanceType != instanceType {
			continue
		}
		if region != "" && item.Region != region {
			continue
		}
		if osLabel != "" && item.OsLabel != osLabel {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}
