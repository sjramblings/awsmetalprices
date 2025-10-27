package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sjramblings/awsmetalprices/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	manager := NewManager(".cache", 7)

	key := manager.GenerateKey("2025-10-20", "us-east-1", "m5.metal", "Linux")
	expected := filepath.Join(".cache", "2025-10-20", "us-east-1", "m5_metal__Linux.json")

	assert.Equal(t, expected, key)
}

func TestGenerateKey_WithSpaces(t *testing.T) {
	manager := NewManager(".cache", 7)

	key := manager.GenerateKey("2025-10-20", "us-east-1", "m7i.metal-24xl", "Windows Server")
	expected := filepath.Join(".cache", "2025-10-20", "us-east-1", "m7i_metal-24xl__Windows_Server.json")

	assert.Equal(t, expected, key)
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, 7)

	pricePoint := &models.PricePoint{
		Region:       "us-east-1",
		InstanceType: "m5.metal",
		VCpu:         96,
		MemoryGiB:    384.0,
		OsLabel:      "Linux",
		OnDemandUSDH: 5.424,
		ReservedUSDH: map[string]float64{
			"1yr_AU": 3.123,
			"1yr_PU": 3.456,
		},
	}

	// Set the cache entry
	err := manager.Set("2025-10-20", "us-east-1", "m5.metal", "Linux", pricePoint)
	require.NoError(t, err)

	// Get the cache entry
	retrieved, found := manager.Get("2025-10-20", "us-east-1", "m5.metal", "Linux")
	assert.True(t, found)
	assert.NotNil(t, retrieved)
	assert.Equal(t, pricePoint.Region, retrieved.Region)
	assert.Equal(t, pricePoint.InstanceType, retrieved.InstanceType)
	assert.Equal(t, pricePoint.VCpu, retrieved.VCpu)
	assert.Equal(t, pricePoint.OnDemandUSDH, retrieved.OnDemandUSDH)
	assert.Equal(t, 2, len(retrieved.ReservedUSDH))
}

func TestGet_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, 7)

	retrieved, found := manager.Get("2025-10-20", "us-east-1", "nonexistent", "Linux")
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestIsExpired(t *testing.T) {
	manager := NewManager(".cache", 7)

	t.Run("not expired", func(t *testing.T) {
		modTime := time.Now().Add(-3 * 24 * time.Hour) // 3 days ago
		assert.False(t, manager.IsExpired(modTime))
	})

	t.Run("expired", func(t *testing.T) {
		modTime := time.Now().Add(-10 * 24 * time.Hour) // 10 days ago
		assert.True(t, manager.IsExpired(modTime))
	})

	t.Run("exactly at TTL", func(t *testing.T) {
		modTime := time.Now().Add(-7*24*time.Hour - 1*time.Second) // Just over 7 days
		assert.True(t, manager.IsExpired(modTime))
	})
}

func TestGet_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, 1) // 1 day TTL

	pricePoint := &models.PricePoint{
		Region:       "us-east-1",
		InstanceType: "m5.metal",
		OsLabel:      "Linux",
	}

	// Set the cache entry
	err := manager.Set("2025-10-20", "us-east-1", "m5.metal", "Linux", pricePoint)
	require.NoError(t, err)

	// Modify the file timestamp to simulate an old cache entry
	key := manager.GenerateKey("2025-10-20", "us-east-1", "m5.metal", "Linux")
	oldTime := time.Now().Add(-3 * 24 * time.Hour) // 3 days ago
	err = os.Chtimes(key, oldTime, oldTime)
	require.NoError(t, err)

	// Try to get the expired entry
	retrieved, found := manager.Get("2025-10-20", "us-east-1", "m5.metal", "Linux")
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir, 7)

	pricePoint := &models.PricePoint{
		Region:       "us-east-1",
		InstanceType: "m5.metal",
		OsLabel:      "Linux",
	}

	// Create some cache entries
	err := manager.Set("2025-10-20", "us-east-1", "m5.metal", "Linux", pricePoint)
	require.NoError(t, err)
	err = manager.Set("2025-10-20", "us-west-2", "c5.metal", "Windows", pricePoint)
	require.NoError(t, err)

	// Verify cache directory exists
	_, err = os.Stat(tmpDir)
	require.NoError(t, err)

	// Clear the cache
	err = manager.Clear()
	require.NoError(t, err)

	// Verify cache directory is removed
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err))
}

func TestSet_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	manager := NewManager(cacheDir, 7)

	pricePoint := &models.PricePoint{
		Region:       "us-east-1",
		InstanceType: "m5.metal",
		OsLabel:      "Linux",
	}

	// Set should create all necessary directories
	err := manager.Set("2025-10-20", "us-east-1", "m5.metal", "Linux", pricePoint)
	require.NoError(t, err)

	// Verify directory structure was created
	expectedDir := filepath.Join(cacheDir, "2025-10-20", "us-east-1")
	_, err = os.Stat(expectedDir)
	require.NoError(t, err)
}
