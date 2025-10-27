package pricing

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ProductAttributes contains EC2 instance attributes
type ProductAttributes struct {
	VCpu   string `json:"vcpu"`
	Memory string `json:"memory"`
}

// PriceDimension represents a single pricing entry
type PriceDimension struct {
	Unit         interface{}       `json:"unit"`        // Can be string like "Hrs" or "Quantity"
	Description  string            `json:"description"` // Human-readable description
	PricePerUnit map[string]string `json:"pricePerUnit"`
}

// OnDemandTerm represents on-demand pricing terms
type OnDemandTerm struct {
	PriceDimensions map[string]PriceDimension `json:"priceDimensions"`
}

// ReservedTerm represents reserved instance pricing terms
type ReservedTerm struct {
	TermAttributes struct {
		LeaseContractLength string `json:"LeaseContractLength"`
		PurchaseOption      string `json:"PurchaseOption"`
		OfferingClass       string `json:"OfferingClass"`
	} `json:"termAttributes"`
	PriceDimensions map[string]PriceDimension `json:"priceDimensions"`
}

// Product represents the full AWS Pricing API product structure
type Product struct {
	Product struct {
		Attributes ProductAttributes `json:"attributes"`
	} `json:"product"`
	Terms struct {
		OnDemand map[string]OnDemandTerm `json:"OnDemand"`
		Reserved map[string]ReservedTerm `json:"Reserved"`
	} `json:"terms"`
}

// ParsedPricing contains extracted pricing information
type ParsedPricing struct {
	VCpu          int
	MemoryGiB     float64
	OnDemandRate  float64
	ReservedRates map[string]ReservedRate
}

// ReservedRate contains RI pricing details
type ReservedRate struct {
	Term          string
	PurchaseType  string
	OfferingClass string
	UpfrontPrice  float64
	HourlyPrice   float64
}

// ParsePriceList parses AWS Pricing API response and extracts pricing data
func ParsePriceList(priceListJSON string) (*ParsedPricing, error) {
	var product Product
	if err := json.Unmarshal([]byte(priceListJSON), &product); err != nil {
		return nil, fmt.Errorf("failed to parse price list JSON: %w", err)
	}

	parsed := &ParsedPricing{
		ReservedRates: make(map[string]ReservedRate),
	}

	// Parse attributes
	var err error
	parsed.VCpu, err = strconv.Atoi(product.Product.Attributes.VCpu)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vCPU: %w", err)
	}

	// Parse memory (format: "384 GiB")
	memoryStr := strings.TrimSuffix(product.Product.Attributes.Memory, " GiB")
	memoryStr = strings.TrimSpace(memoryStr)
	parsed.MemoryGiB, err = strconv.ParseFloat(memoryStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory: %w", err)
	}

	// Parse on-demand pricing
	for _, term := range product.Terms.OnDemand {
		for _, dimension := range term.PriceDimensions {
			if priceStr, ok := dimension.PricePerUnit["USD"]; ok {
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse on-demand price: %w", err)
				}
				parsed.OnDemandRate = price
				break
			}
		}
	}

	// Parse reserved pricing
	for _, term := range product.Terms.Reserved {
		var upfront, hourly float64

		for _, dimension := range term.PriceDimensions {
			if priceStr, ok := dimension.PricePerUnit["USD"]; ok {
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					continue
				}

				// Check the unit field to determine price type
				// AWS uses "Quantity" for upfront fees and "Hrs" for hourly charges
				unit, hasUnit := dimension.Unit.(string)
				if hasUnit {
					switch unit {
					case "Quantity":
						upfront = price
					case "Hrs":
						hourly = price
					}
				}
			}
		}

		termKey := mapReservedTermKey(
			term.TermAttributes.LeaseContractLength,
			term.TermAttributes.PurchaseOption,
			term.TermAttributes.OfferingClass,
		)

		if termKey != "" {
			parsed.ReservedRates[termKey] = ReservedRate{
				Term:          term.TermAttributes.LeaseContractLength,
				PurchaseType:  term.TermAttributes.PurchaseOption,
				OfferingClass: term.TermAttributes.OfferingClass,
				UpfrontPrice:  upfront,
				HourlyPrice:   hourly,
			}
		}
	}

	return parsed, nil
}

// mapReservedTermKey creates a standardized key for reserved instance terms
// Only returns keys for "standard" offering class (filters out "convertible")
func mapReservedTermKey(term, purchaseOption, offeringClass string) string {
	// Only process "standard" class RIs (as per config.yaml)
	if offeringClass != "standard" {
		return ""
	}

	var termPrefix string
	switch term {
	case "1yr":
		termPrefix = "1yr"
	case "3yr":
		termPrefix = "3yr"
	default:
		return ""
	}

	var purchaseSuffix string
	switch purchaseOption {
	case "All Upfront":
		purchaseSuffix = "AU"
	case "Partial Upfront":
		purchaseSuffix = "PU"
	case "No Upfront":
		purchaseSuffix = "NU"
	default:
		return ""
	}

	return termPrefix + "_" + purchaseSuffix
}
