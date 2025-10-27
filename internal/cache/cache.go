package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sjramblings/awsmetalprices/internal/models"
)

// Manager handles caching of pricing data
type Manager struct {
	dir     string
	ttlDays int
}

// NewManager creates a new cache manager
func NewManager(dir string, ttlDays int) *Manager {
	return &Manager{
		dir:     dir,
		ttlDays: ttlDays,
	}
}

// GenerateKey creates a cache key path for a specific price point
func (m *Manager) GenerateKey(date, region, instanceType, osLabel string) string {
	// Sanitize instance type and OS label for filesystem
	sanitizedInstance := strings.ReplaceAll(instanceType, ".", "_")
	sanitizedOS := strings.ReplaceAll(osLabel, " ", "_")

	filename := fmt.Sprintf("%s__%s.json", sanitizedInstance, sanitizedOS)
	return filepath.Join(m.dir, date, region, filename)
}

// Get retrieves a cached price point if it exists and is not expired
func (m *Manager) Get(date, region, instanceType, osLabel string) (*models.PricePoint, bool) {
	key := m.GenerateKey(date, region, instanceType, osLabel)

	// Check if file exists
	info, err := os.Stat(key)
	if err != nil {
		return nil, false
	}

	// Check if expired
	if m.IsExpired(info.ModTime()) {
		return nil, false
	}

	// Read and unmarshal
	data, err := os.ReadFile(key)
	if err != nil {
		return nil, false
	}

	var pricePoint models.PricePoint
	if err := json.Unmarshal(data, &pricePoint); err != nil {
		return nil, false
	}

	return &pricePoint, true
}

// Set stores a price point in the cache
func (m *Manager) Set(date, region, instanceType, osLabel string, pricePoint *models.PricePoint) error {
	key := m.GenerateKey(date, region, instanceType, osLabel)

	// Create directory structure
	dir := filepath.Dir(key)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(pricePoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal price point: %w", err)
	}

	// Write to file
	if err := os.WriteFile(key, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// IsExpired checks if a cache entry has exceeded the TTL
func (m *Manager) IsExpired(modTime time.Time) bool {
	expiryTime := modTime.Add(time.Duration(m.ttlDays) * 24 * time.Hour)
	return time.Now().After(expiryTime)
}

// Clear removes all cached data
func (m *Manager) Clear() error {
	if err := os.RemoveAll(m.dir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}
