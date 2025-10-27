# AWS Metal Prices

> A powerful Go CLI tool for retrieving, tracking, and comparing AWS EC2 bare metal instance pricing across regions and operating systems.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

## 📑 Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Commands](#commands)
- [Output Formats](#output-formats)
- [Use Cases](#use-cases)
- [How It Works](#how-it-works)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [Support](#support)

---

## Overview

AWS Metal Prices simplifies the complex task of gathering and comparing EC2 bare metal instance pricing. Whether you're optimizing costs, planning infrastructure, or evaluating Reserved Instance options, this tool provides accurate, up-to-date pricing data in formats you can actually use.

### Why Use This Tool?

- **Save Time**: No more manual price lookups across the AWS console
- **Compare Intelligently**: Easily compare On-Demand vs Reserved Instance pricing across multiple payment options
- **Track Changes**: Built-in diff functionality to monitor price changes over time
- **Automate Workflows**: JSON output enables integration with your existing automation
- **Stay Current**: Smart caching balances fresh data with minimal API calls

## Key Features

✅ **Comprehensive Pricing Data**
- On-Demand and Reserved Instance pricing (1yr, 3yr terms)
- All RI payment options: All Upfront, Partial Upfront, No Upfront
- Effective hourly rates calculated automatically for RI pricing

✅ **Flexible Configuration**
- Multi-region support
- Config-driven instance selection
- OS/licensing matrix (Linux, Windows - excludes SQL Server and macOS)
- Customizable output formats

✅ **Smart Caching**
- Local file-based cache with configurable TTL (default: 7 days)
- Reduces API calls and speeds up repeated queries
- Optional cache bypass for fresh data

✅ **Multiple Output Formats**
- **Markdown**: Beautiful, human-readable pricing tables
- **JSON**: Structured data for automation and integration

✅ **Powerful Utilities**
- **Validate**: Ensure pricing data completeness
- **Diff**: Compare pricing catalogs to track changes
- **Print**: Query and filter cached pricing data
- **Export**: Split by instance type for focused analysis

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/sjramblings/awsmetalprices.git
cd awsmetalprices

# Build the binary
make build

# Or install to GOPATH/bin
make install
```

### Pre-built Binaries

Download pre-built binaries from the [Releases](https://github.com/sjramblings/awsmetalprices/releases) page.

## Prerequisites

### AWS Credentials

**Important:** The AWS Pricing API requires authentication, even though it returns public pricing data. You need valid AWS credentials to use this tool.

The tool uses the standard AWS SDK credential chain, which checks for credentials in the following order:

1. **Environment variables:**
   ```bash
   export AWS_ACCESS_KEY_ID="your-access-key"
   export AWS_SECRET_ACCESS_KEY="your-secret-key"
   export AWS_REGION="us-east-1"  # Optional, defaults to us-east-1 for pricing API
   ```

2. **AWS credentials file** (`~/.aws/credentials`):
   ```ini
   [default]
   aws_access_key_id = your-access-key
   aws_secret_access_key = your-secret-key
   ```

3. **AWS config file** (`~/.aws/config`):
   ```ini
   [default]
   region = us-east-1
   ```

4. **IAM role** (when running on EC2, ECS, Lambda, etc.)

**Required IAM Permissions:**

The AWS credentials need minimal permissions - only read access to the Pricing API:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "pricing:GetProducts",
        "pricing:DescribeServices"
      ],
      "Resource": "*"
    }
  ]
}
```

## Quick Start

### 1️⃣ Set Up AWS Credentials

The tool requires valid AWS credentials (even though pricing data is public). Use any standard method:

```bash
# Option 1: Environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# Option 2: AWS credentials file (~/.aws/credentials)
# [default]
# aws_access_key_id = your-access-key
# aws_secret_access_key = your-secret-key

# Option 3: IAM role (when running on AWS)
```

**Required IAM Permissions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["pricing:GetProducts", "pricing:DescribeServices"],
    "Resource": "*"
  }]
}
```

### 2️⃣ Configure Your Pricing Requirements

Edit `config.yaml` to specify what pricing data you need:

```yaml
regions:
  - "us-east-1"
  - "ap-southeast-2"

instances:
  explicit:
    - "m7i.metal-24xl"
    - "c7i.metal-24xl"
    - "m5.metal"

osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"

reserved:
  terms: ["1yr", "3yr"]
  purchases: ["All Upfront", "Partial Upfront", "No Upfront"]
```

### 3️⃣ Fetch Pricing Data

```bash
# Fetch pricing data
./bin/awsmetalprices fetch --config config.yaml --out ./pricing

# Fetch with fresh data (bypass cache)
./bin/awsmetalprices fetch --config config.yaml --out ./pricing --no-cache
```

### 4️⃣ View Your Results

**Markdown Output** (`pricing/pricing.md`):
```markdown
## m7i.metal-24xl - Linux

| Region | vCPU | Mem (GiB) | On-Demand | 1yr RI (A/P/N) | 3yr RI (A/P/N) |
|--------|------|-----------|-----------|----------------|----------------|
| us-east-1 | 96 | 384 | $3.4567 | $3.12/$3.22/$3.33 | $2.44/$2.56/$2.67 |
```

**JSON Output** (`pricing/catalog.json`):
```json
{
  "generatedAt": "2025-10-27T00:00:00Z",
  "currency": "USD",
  "items": [{
    "region": "us-east-1",
    "instanceType": "m7i.metal-24xl",
    "vCpu": 96,
    "memoryGiB": 384,
    "operatingSystem": "Linux",
    "onDemandUSDH": 3.4567,
    "reservedUSDH": {
      "1yr_AU": 3.1234,
      "1yr_PU": 3.2222,
      "1yr_NU": 3.3333
    }
  }]
}
```

## Configuration

The tool uses a YAML configuration file to define what pricing data to fetch. See `config.yaml` for a complete example.

### Key Configuration Options

| Field | Description | Default |
|-------|-------------|---------|
| `date` | Pricing date ("auto" for current date or specific date like "2025-10-20") | `auto` |
| `regions` | List of AWS regions to query | Required |
| `instances.explicit` | Specific instance types to include | Required |
| `excludePatterns` | Regex patterns to exclude instances (e.g., "^mac.*") | `[]` |
| `osMatrix` | Operating system and licensing configurations | Required |
| `cache.ttlDays` | Cache time-to-live in days | `7` |
| `render.roundDecimals` | Number of decimal places for prices | `4` |
| `render.splitByInstance` | Create separate files per instance type | `false` |

### Operating System Matrix

Configure which OS/licensing combinations to fetch:

```yaml
osMatrix:
  - operatingSystem: "Linux"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Linux"

  - operatingSystem: "Windows"
    preInstalledSw: "NA"
    licenseModel: "No License required"
    label: "Windows"
```

### Reserved Instance Terms

Configure which RI terms to include:

```yaml
reserved:
  classes: ["standard"]
  terms: ["1yr", "3yr"]
  purchases: ["All Upfront", "Partial Upfront", "No Upfront"]
```

## Commands

### 📥 fetch - Retrieve Pricing Data

Fetches current pricing from AWS and exports to Markdown and JSON formats.

```bash
# Basic usage
awsmetalprices fetch --config config.yaml --out ./pricing

# Force fresh data (bypass cache)
awsmetalprices fetch --config config.yaml --out ./pricing --no-cache
```

**Options:**
| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to configuration file | `config.yaml` |
| `--out` | Output directory for pricing data | `./pricing` |
| `--no-cache` | Bypass cache and fetch fresh data | `false` |

**Outputs:**
- `pricing/pricing.md` - Human-readable Markdown tables
- `pricing/catalog.json` - Machine-readable JSON data

---

### ✅ validate - Verify Data Completeness

Validates that a pricing catalog contains all expected data based on your configuration.

```bash
awsmetalprices validate --catalog pricing/catalog.json
```

**Validation Checks:**
- ✓ All configured instances are present
- ✓ All configured regions are present
- ✓ All OS configurations are present
- ✓ All items have complete attributes (vCPU, memory, pricing)
- ✓ No $0.00 pricing (common API parsing bug)

**Exit Codes:**
- `0` - Validation passed
- `1` - Validation failed (with detailed error messages)

---

### 🔍 print - Query Pricing Data

Query and filter cached pricing data with flexible filtering options.

```bash
# Print all pricing for a specific instance
awsmetalprices print --catalog pricing/catalog.json \
  --instance m5.metal

# Filter by multiple criteria
awsmetalprices print --catalog pricing/catalog.json \
  --instance m7i.metal-24xl \
  --os Linux \
  --region us-east-1 \
  --format table

# Output as JSON for scripting
awsmetalprices print --catalog pricing/catalog.json \
  --instance c7i.metal-24xl \
  --format json
```

**Options:**
| Flag | Description | Required |
|------|-------------|----------|
| `--catalog` | Path to catalog JSON file | Yes |
| `--instance` | Filter by instance type | No |
| `--os` | Filter by OS label (e.g., "Linux", "Windows") | No |
| `--region` | Filter by region (e.g., "us-east-1") | No |
| `--format` | Output format: `table` or `json` | No (default: `table`) |

---

### 📊 diff - Compare Pricing Changes

Compare two pricing catalogs to identify changes, perfect for tracking price updates over time.

```bash
# Compare catalogs from different dates
awsmetalprices diff \
  --old pricing/catalog-2025-10-10.json \
  --new pricing/catalog.json

# Output differences as JSON for automation
awsmetalprices diff \
  --old pricing/catalog-2025-10-10.json \
  --new pricing/catalog.json \
  --format json
```

**Options:**
| Flag | Description | Required |
|------|-------------|----------|
| `--old` | Path to older catalog | Yes |
| `--new` | Path to newer catalog | Yes |
| `--format` | Output format: `table` or `json` | No (default: `table`) |

**Reports:**
- ➕ **Added items** - New instance/region/OS combinations
- ➖ **Removed items** - Discontinued offerings
- 📈 **Price increases** - With delta and percentage
- 📉 **Price decreases** - With delta and percentage
- ➡️ **Unchanged items** - For reference

## Output Formats

### Markdown

Human-readable pricing tables grouped by instance type and OS:

```markdown
# EC2 Metal Pricing — Linux & Windows — 2025-10-20

## m7i.metal-24xl

### Linux
| Region | vCPU | Mem (GiB) | On-Demand | 1yr RI (A / P / N) | 3yr RI (A / P / N) |
|--------|------|-----------|------------|--------------------|--------------------|
| us-east-1 | 96 | 384 | 3.4567 | 3.1234 / 3.2222 / 3.3333 | 2.4444 / 2.5555 / 2.6666 |

### Windows
| Region | vCPU | Mem (GiB) | On-Demand | 1yr RI (A / P / N) | 3yr RI (A / P / N) |
|--------|------|-----------|------------|--------------------|--------------------|
| us-east-1 | 96 | 384 | 4.6789 | 4.1111 / 4.2222 / 4.3333 | 3.4444 / 3.5555 / 3.6666 |
```

### JSON

Machine-readable structured data for automation:

```json
{
  "generatedAt": "2025-10-20T00:00:00Z",
  "currency": "USD",
  "items": [
    {
      "region": "us-east-1",
      "instanceType": "m7i.metal-24xl",
      "vCpu": 96,
      "memoryGiB": 384,
      "operatingSystem": "Windows",
      "preInstalledSw": "NA",
      "licenseModel": "No License required",
      "osLabel": "Windows",
      "onDemandUSDH": 4.6789,
      "reservedUSDH": {
        "1yr_AU": 4.1111,
        "1yr_PU": 4.2222,
        "1yr_NU": 4.3333,
        "3yr_AU": 3.4444,
        "3yr_PU": 3.5555,
        "3yr_NU": 3.6666
      }
    }
  ]
}
```

## Caching

The tool implements intelligent local file-based caching to minimize AWS API calls and improve performance.

### 🗄️ Cache Structure

```
.cache/
└── 2025-10-27/                    # Date-based organization
    ├── us-east-1/                 # Region
    │   ├── m5.metal__Linux.json
    │   ├── m5.metal__Windows.json
    │   └── c7i.metal-24xl__Linux.json
    └── ap-southeast-2/
        └── i4i.metal__Linux.json
```

### ⚙️ Cache Configuration

```yaml
# In config.yaml
cache:
  dir: ".cache"      # Cache directory location
  ttlDays: 7         # Time-to-live in days
```

### 📋 Cache Behavior

| Scenario | Behavior |
|----------|----------|
| **First fetch** | Queries AWS API, stores response in cache |
| **Within TTL** | Returns cached data (fast, no API calls) |
| **After TTL** | Queries AWS API, updates cache |
| **With `--no-cache`** | Always queries AWS API, updates cache |

### 🔧 Cache Management

```bash
# Clear all cached data
make clean

# Or manually remove cache
rm -rf .cache/

# Bypass cache for single fetch
awsmetalprices fetch --config config.yaml --out ./pricing --no-cache

# Check cache size
du -sh .cache/
```

### 💡 Cache Best Practices

- **Development**: Use `--no-cache` to test with fresh data
- **Production**: Let cache TTL manage freshness (default 7 days is reasonable for pricing data)
- **CI/CD**: Use `--no-cache` to ensure latest pricing
- **Cost Optimization**: Cache reduces API calls (though AWS Pricing API has no direct costs)

### 🔍 Cache Inspection

Cache files are raw AWS API responses in JSON format. You can inspect them directly:

```bash
# View cached response
cat .cache/2025-10-27/us-east-1/m5.metal__Linux.json | jq .

# Check cache age
ls -lh .cache/2025-10-27/us-east-1/
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile targets)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install to GOPATH/bin
make install
```

### Testing

The project includes comprehensive test coverage to ensure reliability and prevent regressions.

#### Quick Testing

```bash
# Run all tests with coverage report
make test

# Run specific test suites
make test-unit          # Unit tests only
make test-integration   # Integration tests with AWS API
make test-regression    # $0 pricing bug regression tests
make test-validate-catalog  # End-to-end validation

# Verbose test output for debugging
make test-verbose

# Generate coverage report
make test-coverage
```

#### Test Organization

```
internal/
├── cache/
│   └── cache_test.go           # Cache TTL and file operations
├── config/
│   └── config_test.go          # Configuration parsing and validation
├── models/
│   ├── types_test.go           # Data structure validation
│   └── validation_test.go      # Pricing validation logic
├── pricing/
│   ├── pricing_test.go         # API client tests
│   ├── parser_test.go          # AWS response parsing
│   ├── parser_regression_test.go  # $0 pricing bug prevention
│   └── integration_test.go     # End-to-end API tests
└── render/
    └── render_test.go          # Markdown and JSON output
```

#### 🛡️ Regression Testing

This project includes **extensive regression tests** to prevent the $0 Reserved Instance pricing bug from reoccurring:

- **Parser Unit Tests**: Validate AWS API response parsing with real response structures
- **Integration Tests**: End-to-end tests with actual AWS API calls
- **Validation Tests**: Ensure no $0.00 pricing in output
- **Coverage Tracking**: Monitor test coverage for critical paths

See [`docs/TESTING.md`](docs/TESTING.md) for detailed documentation.

**Current Test Coverage:**
- 🟢 **Models Package**: 100%
- 🟡 **Pricing Package**: 48.8% (targeting 80%+)
- 🟡 **Overall**: Growing with each PR

### Code Quality

```bash
# Format code with gofmt
make fmt

# Run golangci-lint
make lint

# Run all quality checks (fmt + lint + test)
make check

# Install linter (if not already installed)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### Quality Standards

- **Code Formatting**: All code must pass `gofmt`
- **Linting**: Zero linter warnings required
- **Test Coverage**: New code should include tests
- **Documentation**: Public functions must have GoDoc comments
- **Error Handling**: All errors must be properly handled

### Project Structure

```
aws-metal-prices/
├── cmd/
│   └── awsmetalprices/     # Main entry point
├── internal/
│   ├── cache/              # Caching logic
│   ├── cli/                # CLI commands
│   ├── config/             # Configuration management
│   ├── models/             # Data models
│   ├── pricing/            # AWS Pricing API client
│   └── render/             # Output renderers
├── testdata/               # Test fixtures
├── config.yaml             # Default configuration
├── Makefile                # Build automation
└── README.md               # This file
```

## Use Cases

### 💰 Cost Optimization
Compare On-Demand vs Reserved Instance pricing to calculate potential savings across different commitment terms and payment options.

```bash
# Generate pricing report for cost analysis
awsmetalprices fetch --config config.yaml --out ./reports/$(date +%Y-%m-%d)
```

### 📊 Infrastructure Planning
Evaluate pricing across multiple regions to determine optimal deployment locations for your workloads.

```bash
# Compare pricing between US and APAC regions
awsmetalprices print --catalog pricing/catalog.json --region us-east-1 --format table
awsmetalprices print --catalog pricing/catalog.json --region ap-southeast-2 --format table
```

### 🔔 Price Change Monitoring
Track pricing changes over time to understand trends and make informed purchasing decisions.

```bash
# Weekly price monitoring workflow
awsmetalprices fetch --config config.yaml --out ./pricing
awsmetalprices diff --old ./pricing/archive/2025-10-20.json --new ./pricing/catalog.json
```

### 🤖 Automation & Integration
Export pricing data as JSON for integration with your existing infrastructure-as-code, cost management, or reporting systems.

```bash
# Automated pricing updates in CI/CD
awsmetalprices fetch --config config.yaml --out ./pricing --no-cache
awsmetalprices validate --catalog ./pricing/catalog.json
# Parse JSON and update your systems
```

---

## How It Works

### Architecture Overview

```
┌─────────────────┐
│  config.yaml    │
│  (User Config)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐      ┌─────────────────┐
│  CLI Commands   │─────▶│ Cache Layer  │─────▶│  AWS Pricing    │
│ (fetch/validate)│      │   (Local)    │      │      API        │
└────────┬────────┘      └──────────────┘      └─────────────────┘
         │
         │                                      ┌─────────────────┐
         │                                      │   pricing.md    │
         └─────────────────────────────────────▶│  catalog.json   │
                                                └─────────────────┘
```

### Workflow Steps

1. **📋 Configuration Loading**
   - Reads `config.yaml` to determine instances, regions, and OS configurations
   - Validates configuration parameters

2. **🔍 Cache Check**
   - Checks local `.cache/` directory for existing pricing data
   - Respects TTL setting (default: 7 days)
   - Bypassed with `--no-cache` flag

3. **☁️ AWS API Query** (on cache miss)
   - Authenticates using AWS SDK credential chain
   - Queries AWS Pricing API in us-east-1 (only region for pricing data)
   - Applies filters per instance/region/OS combination

4. **📐 Data Parsing & Normalization**
   - Extracts instance attributes (vCPU, memory, etc.)
   - Parses On-Demand hourly pricing
   - Calculates effective hourly rates for Reserved Instances

5. **💾 Caching**
   - Stores raw API responses in `.cache/` directory
   - Organizes by date, region, and instance type

6. **📄 Output Rendering**
   - Generates human-readable Markdown tables
   - Creates machine-readable JSON catalog
   - Optionally splits output by instance type

### Reserved Instance Calculation

The tool automatically calculates **effective hourly rates** for Reserved Instances by combining upfront and recurring costs:

```
termHours = termYears × 365 × 24
effectiveHourly = (upfrontCost / termHours) + recurringHourly
```

**Example: 1-Year All Upfront RI**
```
Upfront Cost:     $10,000
Recurring Hourly: $0.00
Term Duration:    8,760 hours (1 year)

Effective Rate:   $10,000 ÷ 8,760 = $1.1416/hour
```

**Example: 3-Year Partial Upfront RI**
```
Upfront Cost:     $15,000
Recurring Hourly: $0.50
Term Duration:    26,280 hours (3 years)

Effective Rate:   ($15,000 ÷ 26,280) + $0.50 = $1.0708/hour
```

This normalization enables direct cost comparison across all pricing models.

### AWS Pricing API Details

- **Authentication Required**: Even though pricing is public data, AWS requires valid credentials
- **Credential Chain**: Supports environment variables, `~/.aws/` files, and IAM roles
- **API Region**: All queries go to `us-east-1` (the only region serving pricing data)
- **Minimal Permissions**: Only needs `pricing:GetProducts` and `pricing:DescribeServices`
- **Rate Limiting**: Built-in retry logic with exponential backoff

## Supported Regions

The tool supports all major AWS regions. Region codes are mapped to AWS location labels:

- `us-east-1` → US East (N. Virginia)
- `us-west-2` → US West (Oregon)
- `eu-central-1` → EU (Frankfurt)
- `ap-southeast-2` → Asia Pacific (Sydney)
- And more...

## Supported Metal Instances

The tool supports all AWS EC2 bare metal instance types. Here are some common families:

### 🖥️ General Purpose
Perfect for balanced workloads requiring CPU, memory, and networking resources.
- **M5 Family**: `m5.metal`, `m5d.metal`, `m5n.metal`, `m5dn.metal`, `m5zn.metal`
- **M6 Family**: `m6i.metal`, `m6id.metal`, `m6in.metal`, `m6idn.metal`
- **M7 Family**: `m7i.metal-24xl`, `m7i.metal-48xl`, `m7g.metal`

### ⚡ Compute Optimized
Ideal for compute-intensive workloads like batch processing, scientific modeling, and gaming.
- **C5 Family**: `c5.metal`, `c5d.metal`, `c5n.metal`
- **C6 Family**: `c6i.metal`, `c6id.metal`, `c6in.metal`, `c6gn.metal`
- **C7 Family**: `c7i.metal-24xl`, `c7i.metal-48xl`, `c7g.metal`, `c7gn.metal`

### 💾 Memory Optimized
Designed for memory-intensive workloads like in-memory databases and real-time big data analytics.
- **R5 Family**: `r5.metal`, `r5d.metal`, `r5n.metal`, `r5dn.metal`, `r5b.metal`
- **R6 Family**: `r6i.metal`, `r6id.metal`, `r6in.metal`, `r6idn.metal`
- **R7 Family**: `r7i.metal-24xl`, `r7i.metal-48xl`, `r7iz.metal-16xl`, `r7g.metal`
- **X2 Family**: `x2idn.metal`, `x2iedn.metal`, `x2iezn.metal`

### 💽 Storage Optimized
Built for high sequential read/write access to large datasets.
- **I3 Family**: `i3.metal`, `i3en.metal`
- **I4 Family**: `i4i.metal`, `i4g.metal`
- **I7 Family**: `i7ie.metal-24xl`, `i7ie.metal-48xl`
- **D3 Family**: `d3.metal`, `d3en.metal`

### 🎮 Accelerated Computing
For GPU-intensive workloads like ML training, graphics rendering, and video transcoding.
- **P4 Family**: `p4d.metal`
- **P5 Family**: `p5.metal`
- **G4 Family**: `g4dn.metal`
- **G5 Family**: `g5.metal`

### 📊 High Performance Computing
- **Hpc7 Family**: `hpc7g.metal`, `hpc7a.metal`

> **Note**: Instance availability varies by region. Use the `fetch` command to discover what's available in your target regions.

## Limitations

⚠️ **Current Scope & Known Limitations:**

| Limitation | Details | Workaround |
|------------|---------|------------|
| **Pricing Type** | Only public list pricing | Does not include Enterprise Discount Programs, private offers, or volume discounts |
| **Spot Pricing** | Not included | Spot prices are highly dynamic and require different API endpoints |
| **macOS Instances** | Excluded by default | Can be included by removing `^mac.*` from `excludePatterns` in config |
| **SQL Server** | Not supported | Only includes "No License required" options for simplicity |
| **RI Availability** | Not all instances have RI pricing | Some newer instance types may lack Reserved Instance options |
| **Savings Plans** | Not included | Future enhancement - see below |

## Future Enhancements

We're planning several enhancements to make the tool even more powerful:

### 🌍 Multi-Currency Support
```yaml
# Future config option
currency: "AUD"
fxRate: 1.52  # Pinned exchange rate for consistency
```
Convert pricing to other currencies with configurable exchange rates for cost planning in your local currency.

### 🔍 Instance Family Discovery
```yaml
# Future config option
instances:
  patterns:
    - "m7i.*"      # Auto-expand to all m7i instances
    - "c7[a-z].*"  # Regex pattern matching
```
Automatically discover and fetch pricing for entire instance families.

### 💰 Savings Plans Integration
```yaml
# Future config option
savingsPlans:
  enabled: true
  commitmentTypes: ["Compute", "EC2Instance"]
  terms: ["1yr", "3yr"]
```
Show Savings Plans pricing alongside Reserved Instance options for complete cost comparison.

### 🔌 MCP Server Mode
```bash
# Future command
awsmetalprices serve --port 8080
```
Run as a local HTTP server for real-time querying from scripts and dashboards.

### 📊 Web UI Dashboard
Interactive web interface for browsing, filtering, and visualizing pricing data with:
- Interactive pricing tables
- Cost comparison charts
- Price trend graphs
- Export functionality

### 🔔 Price Change Alerts
```yaml
# Future config option
alerts:
  webhook: "https://your-webhook-url.com"
  threshold: 5  # Alert on 5% price change
```
Automated notifications when prices change beyond configured thresholds.

### 📈 Historical Price Tracking
Built-in database to track pricing over time and visualize trends:
```bash
awsmetalprices history --instance m5.metal --region us-east-1 --days 90
```

**Want to contribute?** Check out our [Contributing](#contributing) section!

## Troubleshooting

### ❌ No Pricing Data Found

**Symptoms:**
- Empty output files
- "No items found" messages
- Missing instance types in results

**Solutions:**
```bash
# 1. Verify instance type is available in the region
#    Check AWS documentation for regional availability

# 2. Bypass cache to fetch fresh data
awsmetalprices fetch --config config.yaml --out ./pricing --no-cache

# 3. Validate your configuration
cat config.yaml  # Check for typos in instance names

# 4. Test with a known-good instance
#    Try m5.metal or c5.metal which are widely available
```

### 🗂️ Cache Issues

**Symptoms:**
- Stale pricing data
- Unexpected pricing values
- Cache read/write errors

**Solutions:**
```bash
# Clear all cached data
make clean

# Or manually remove cache directory
rm -rf .cache/

# Check cache directory permissions
ls -la .cache/

# Verify disk space
df -h .

# Fetch with cache bypass
awsmetalprices fetch --config config.yaml --out ./pricing --no-cache
```

### ☁️ AWS API Errors

**Symptoms:**
- "Access Denied" errors
- Connection timeouts
- Rate limiting messages

**Solutions:**
```bash
# 1. Verify AWS credentials are configured
aws sts get-caller-identity

# 2. Check IAM permissions
#    Ensure your user/role has pricing:GetProducts and pricing:DescribeServices

# 3. Test AWS Pricing API connectivity
aws pricing describe-services --service-code AmazonEC2 --region us-east-1

# 4. Check for rate limiting
#    The tool includes automatic retry logic with exponential backoff
#    If rate limited, wait a few minutes and try again

# 5. Verify internet connectivity
ping pricing.us-east-1.amazonaws.com
```

### 🐛 Validation Failures

**Symptoms:**
- `awsmetalprices validate` reports missing data
- $0.00 pricing in output

**Solutions:**
```bash
# 1. Re-fetch with fresh data
awsmetalprices fetch --config config.yaml --out ./pricing --no-cache

# 2. Check if instance type supports Reserved Instances
#    Not all metal instances have RI pricing available

# 3. Verify OS/licensing combinations
#    Ensure your osMatrix matches AWS offerings

# 4. Run validation with detailed output
awsmetalprices validate --catalog pricing/catalog.json
```

### 📝 Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `NoCredentialProviders` | AWS credentials not found | Configure AWS credentials (see Quick Start) |
| `AccessDenied` | Insufficient IAM permissions | Add `pricing:GetProducts` permission |
| `UnmarshalError` | Corrupted cache file | Run `make clean` to clear cache |
| `instance not found` | Instance unavailable in region | Check AWS regional availability |
| `$0.00 pricing` | API parsing issue | Report as bug (should be caught by tests) |

## Contributing

We welcome contributions! Whether it's bug fixes, new features, documentation improvements, or test coverage, your help is appreciated.

### Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
   ```bash
   git clone https://github.com/YOUR_USERNAME/aws-metal-prices.git
   cd aws-metal-prices
   ```

3. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make your changes** with tests
   ```bash
   # Write your code
   # Add or update tests
   ```

5. **Run quality checks**
   ```bash
   make check  # Runs fmt, lint, and all tests
   ```

6. **Commit and push**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   git push origin feature/your-feature-name
   ```

7. **Submit a Pull Request** on GitHub

### Development Guidelines

- **Code Style**: Run `make fmt` before committing
- **Testing**: Add tests for new functionality (run `make test`)
- **Linting**: Ensure `make lint` passes
- **Documentation**: Update README.md for user-facing changes
- **Commits**: Use clear, descriptive commit messages

### Testing Your Changes

```bash
# Run all tests
make test

# Run specific test suites
make test-integration      # Integration tests with AWS API
make test-regression       # $0 pricing bug regression tests
make test-validate-catalog # Catalog validation tests

# Check test coverage
make test-coverage
```

### Areas for Contribution

- 🐛 **Bug Fixes**: Check [open issues](https://github.com/sjramblings/aws-metal-prices/issues)
- ✨ **New Features**: See "Future Enhancements" section
- 📚 **Documentation**: Improve examples, add use cases
- 🧪 **Testing**: Increase test coverage
- 🌍 **Localization**: Add support for more currencies/regions

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

### Getting Help

- 📖 **Documentation**: You're reading it! Check the sections above
- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/sjramblings/aws-metal-prices/issues)
- 💬 **Questions**: [GitHub Discussions](https://github.com/sjramblings/aws-metal-prices/discussions)
- 💡 **Feature Requests**: [GitHub Issues](https://github.com/sjramblings/aws-metal-prices/issues) with `enhancement` label

### Reporting Issues

When reporting bugs, please include:

1. **Version information**: `awsmetalprices --version`
2. **Configuration**: Your `config.yaml` (remove sensitive data)
3. **Command executed**: The exact command you ran
4. **Error output**: Complete error messages and stack traces
5. **Environment**: OS, Go version, AWS region

### Security Issues

If you discover a security vulnerability, please email security@example.com instead of creating a public issue.

## Acknowledgments

- **AWS Pricing API** - For providing comprehensive public pricing data
- **Go Community** - For excellent libraries including:
  - [Cobra](https://github.com/spf13/cobra) - CLI framework
  - [Viper](https://github.com/spf13/viper) - Configuration management
  - [AWS SDK for Go](https://github.com/aws/aws-sdk-go) - AWS API client
- **Contributors** - Thank you to everyone who has contributed to this project!

---

<div align="center">

**[⬆ back to top](#aws-metal-prices)**

Made with ☕ and Go

**Star this repo if you find it useful!** ⭐

</div>
