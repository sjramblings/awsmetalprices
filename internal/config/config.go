package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Date            string          `yaml:"date"`
	Regions         []string        `yaml:"regions"`
	Instances       InstancesConfig `yaml:"instances"`
	ExcludePatterns []string        `yaml:"excludePatterns"`
	Tenancy         string          `yaml:"tenancy"`
	OsMatrix        []OsSpec        `yaml:"osMatrix"`
	Currency        string          `yaml:"currency"`
	Reserved        ReservedConfig  `yaml:"reserved"`
	Cache           CacheConfig     `yaml:"cache"`
	Render          RenderConfig    `yaml:"render"`

	// Compiled exclude patterns
	excludeRegexes []*regexp.Regexp
}

// InstancesConfig defines instance selection
type InstancesConfig struct {
	Explicit []string `yaml:"explicit"`
	Patterns []string `yaml:"patterns"`
}

// OsSpec defines operating system and licensing configuration
type OsSpec struct {
	OperatingSystem string `yaml:"operatingSystem"`
	PreInstalledSw  string `yaml:"preInstalledSw"`
	LicenseModel    string `yaml:"licenseModel"`
	Label           string `yaml:"label"`
}

// ReservedConfig defines Reserved Instance terms
type ReservedConfig struct {
	Classes   []string `yaml:"classes"`
	Terms     []string `yaml:"terms"`
	Purchases []string `yaml:"purchases"`
}

// CacheConfig defines caching behavior
type CacheConfig struct {
	Dir     string `yaml:"dir"`
	TTLDays int    `yaml:"ttlDays"`
}

// RenderConfig defines output formatting options
type RenderConfig struct {
	SplitByInstance bool `yaml:"splitByInstance"`
	ShowNotes       bool `yaml:"showNotes"`
	RoundDecimals   int  `yaml:"roundDecimals"`
}

// LoadConfig loads and validates configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply defaults
	if cfg.Date == "" {
		cfg.Date = "auto"
	}
	if cfg.Currency == "" {
		cfg.Currency = "USD"
	}
	if cfg.Tenancy == "" {
		cfg.Tenancy = "Shared"
	}
	if cfg.Cache.Dir == "" {
		cfg.Cache.Dir = ".cache"
	}
	if cfg.Cache.TTLDays == 0 {
		cfg.Cache.TTLDays = 7
	}
	if cfg.Render.RoundDecimals == 0 {
		cfg.Render.RoundDecimals = 4
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Compile exclude patterns
	if err := cfg.compileExcludePatterns(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate ensures required configuration fields are present
func (c *Config) validate() error {
	if len(c.Regions) == 0 {
		return fmt.Errorf("at least one region must be specified")
	}

	if len(c.Instances.Explicit) == 0 && len(c.Instances.Patterns) == 0 {
		return fmt.Errorf("at least one instance (explicit or pattern) must be specified")
	}

	if len(c.OsMatrix) == 0 {
		return fmt.Errorf("at least one OS entry must be specified in osMatrix")
	}

	return nil
}

// compileExcludePatterns compiles regex patterns for instance exclusion
func (c *Config) compileExcludePatterns() error {
	c.excludeRegexes = make([]*regexp.Regexp, 0, len(c.ExcludePatterns))
	for _, pattern := range c.ExcludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
		c.excludeRegexes = append(c.excludeRegexes, re)
	}
	return nil
}

// IsExcluded checks if an instance type matches any exclude pattern
func (c *Config) IsExcluded(instanceType string) bool {
	for _, re := range c.excludeRegexes {
		if re.MatchString(instanceType) {
			return true
		}
	}
	return false
}

// GetDate returns the effective date for pricing queries
func (c *Config) GetDate() string {
	if c.Date == "auto" {
		return time.Now().Format("2006-01-02")
	}
	return c.Date
}

// GetAllInstances returns the complete list of instance types to query
func (c *Config) GetAllInstances() []string {
	instances := make([]string, 0, len(c.Instances.Explicit))
	for _, inst := range c.Instances.Explicit {
		if !c.IsExcluded(inst) {
			instances = append(instances, inst)
		}
	}
	return instances
}
