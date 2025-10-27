package pricing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/sjramblings/awsmetalprices/internal/cache"
	"github.com/sjramblings/awsmetalprices/internal/config"
	"github.com/sjramblings/awsmetalprices/internal/models"
)

// Client wraps the AWS Pricing API client
type Client struct {
	awsClient    *pricing.Client
	config       *config.Config
	cacheManager *cache.Manager
	noCache      bool
}

// NewClient creates a new pricing API client
func NewClient(cfg *config.Config, cacheManager *cache.Manager, noCache bool) (*Client, error) {
	// Load AWS SDK configuration using default credential chain
	// The Pricing API requires authentication even though it returns public data
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"), // Pricing API is only available in us-east-1
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{
		awsClient:    pricing.NewFromConfig(awsCfg),
		config:       cfg,
		cacheManager: cacheManager,
		noCache:      noCache,
	}, nil
}

// FetchPricing retrieves pricing data for all configured instances/regions/OS combinations
func (c *Client) FetchPricing(ctx context.Context) (*models.Catalog, error) {
	catalog := &models.Catalog{
		GeneratedAt: time.Now(),
		Currency:    c.config.Currency,
		Items:       make([]models.PricePoint, 0),
	}

	instances := c.config.GetAllInstances()
	date := c.config.GetDate()

	for _, region := range c.config.Regions {
		for _, instanceType := range instances {
			for _, os := range c.config.OsMatrix {
				// Check cache first
				if !c.noCache && c.cacheManager != nil {
					if cachedPoint, found := c.cacheManager.Get(date, region, instanceType, os.Label); found {
						catalog.Items = append(catalog.Items, *cachedPoint)
						continue
					}
				}

				// Fetch from AWS API
				pricePoint, err := c.fetchSinglePrice(ctx, region, instanceType, os)
				if err != nil {
					// Log error but continue with other instances
					fmt.Printf("Warning: failed to fetch pricing for %s/%s/%s: %v\n", region, instanceType, os.Label, err)
					continue
				}

				if pricePoint != nil {
					catalog.Items = append(catalog.Items, *pricePoint)

					// Cache the result
					if !c.noCache && c.cacheManager != nil {
						if err := c.cacheManager.Set(date, region, instanceType, os.Label, pricePoint); err != nil {
							fmt.Printf("Warning: failed to cache result: %v\n", err)
						}
					}
				}
			}
		}
	}

	catalog.SortByInstance()
	return catalog, nil
}

// fetchSinglePrice retrieves pricing for a single instance/region/OS combination
func (c *Client) fetchSinglePrice(ctx context.Context, region, instanceType string, os config.OsSpec) (*models.PricePoint, error) {
	filters := BuildFilters(region, instanceType, os)

	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
		MaxResults:  aws.Int32(10),
	}

	result, err := c.awsClient.GetProducts(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("AWS API error: %w", err)
	}

	if len(result.PriceList) == 0 {
		return nil, fmt.Errorf("no pricing data found")
	}

	// DEBUG: Log raw AWS response to help diagnose parsing issues
	debugLogAWSResponse(region, instanceType, os.Label, result.PriceList[0])

	// Parse the first result (there should typically only be one)
	parsed, err := ParsePriceList(result.PriceList[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse pricing data: %w", err)
	}

	// Build price point
	pricePoint := &models.PricePoint{
		Region:          region,
		InstanceType:    instanceType,
		VCpu:            parsed.VCpu,
		MemoryGiB:       parsed.MemoryGiB,
		OperatingSystem: os.OperatingSystem,
		PreInstalledSw:  os.PreInstalledSw,
		LicenseModel:    os.LicenseModel,
		OsLabel:         os.Label,
		OnDemandUSDH:    parsed.OnDemandRate,
		ReservedUSDH:    make(map[string]float64),
		Notes:           make([]string, 0),
	}

	// Normalize reserved pricing
	for termKey, rate := range parsed.ReservedRates {
		years := GetTermYears(termKey)
		if years > 0 {
			effectiveRate := NormalizeReservedRate(rate.UpfrontPrice, rate.HourlyPrice, years)
			pricePoint.ReservedUSDH[termKey] = effectiveRate
		}
	}

	// Add note if any RI data is missing
	expectedTerms := len(c.config.Reserved.Terms) * len(c.config.Reserved.Purchases)
	if len(pricePoint.ReservedUSDH) < expectedTerms {
		pricePoint.AddNote("Some Reserved Instance pricing may be unavailable")
	}

	return pricePoint, nil
}

// debugLogAWSResponse writes the raw AWS API response to a debug file for inspection
func debugLogAWSResponse(region, instanceType, osLabel, rawJSON string) {
	debugDir := ".debug"
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create debug directory: %v\n", err)
		return
	}

	// Sanitize filename
	safeInstance := strings.ReplaceAll(instanceType, ".", "_")
	safeOS := strings.ReplaceAll(osLabel, " ", "_")
	filename := filepath.Join(debugDir, fmt.Sprintf("%s_%s_%s_raw.json", region, safeInstance, safeOS))

	if err := os.WriteFile(filename, []byte(rawJSON), 0644); err != nil {
		fmt.Printf("Warning: failed to write debug file: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Raw AWS response saved to %s\n", filename)
	}
}
