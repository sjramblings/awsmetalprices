package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
date: "2025-10-20"
regions:
  - "us-east-1"
instances:
  explicit:
    - "m5.metal"
osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "2025-10-20", cfg.Date)
		assert.Equal(t, []string{"us-east-1"}, cfg.Regions)
		assert.Equal(t, []string{"m5.metal"}, cfg.Instances.Explicit)
		assert.Equal(t, 1, len(cfg.OsMatrix))
	})

	t.Run("applies defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
regions:
  - "us-east-1"
instances:
  explicit:
    - "m5.metal"
osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, "auto", cfg.Date)
		assert.Equal(t, "USD", cfg.Currency)
		assert.Equal(t, "Shared", cfg.Tenancy)
		assert.Equal(t, ".cache", cfg.Cache.Dir)
		assert.Equal(t, 7, cfg.Cache.TTLDays)
		assert.Equal(t, 4, cfg.Render.RoundDecimals)
	})

	t.Run("missing required fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
date: "2025-10-20"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
invalid: yaml: content:
  - this is broken
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(configPath)
		assert.Error(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/config.yaml")
		assert.Error(t, err)
	})
}

func TestExcludePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
regions:
  - "us-east-1"
instances:
  explicit:
    - "m5.metal"
    - "mac1.metal"
excludePatterns:
  - "^mac.*"
osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.False(t, cfg.IsExcluded("m5.metal"))
	assert.True(t, cfg.IsExcluded("mac1.metal"))
	assert.True(t, cfg.IsExcluded("mac2.metal"))
}

func TestGetAllInstances(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
regions:
  - "us-east-1"
instances:
  explicit:
    - "m5.metal"
    - "mac1.metal"
    - "c5.metal"
excludePatterns:
  - "^mac.*"
osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	instances := cfg.GetAllInstances()
	assert.Equal(t, 2, len(instances))
	assert.Contains(t, instances, "m5.metal")
	assert.Contains(t, instances, "c5.metal")
	assert.NotContains(t, instances, "mac1.metal")
}

func TestGetDate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Run("auto date", func(t *testing.T) {
		configContent := `
date: "auto"
regions:
  - "us-east-1"
instances:
  explicit:
    - "m5.metal"
osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		date := cfg.GetDate()
		assert.NotEmpty(t, date)
		assert.NotEqual(t, "auto", date)
	})

	t.Run("fixed date", func(t *testing.T) {
		configContent := `
date: "2025-10-20"
regions:
  - "us-east-1"
instances:
  explicit:
    - "m5.metal"
osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		date := cfg.GetDate()
		assert.Equal(t, "2025-10-20", date)
	})
}
