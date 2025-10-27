# Testing Documentation

## Overview

This document describes the comprehensive test suite designed to prevent the $0 Reserved Instance pricing bug from reoccurring.

## Test Coverage

- **Pricing Package**: 48.8% coverage
- **Models Package**: 100% coverage

## Regression Tests for $0 Pricing Bug

### Root Cause

The original bug had multiple causes:
1. Parser checked dimension key strings for "Upfront" but AWS uses cryptic codes like "2TG2D8R56U"
2. Parser didn't check the `unit` field which AWS uses to identify price types
3. "Convertible" Reserved Instances weren't filtered out (only "standard" should be included)
4. Raw upfront prices (e.g., $29,669) weren't being normalized to effective hourly rates (e.g., $3.39/hr)

### Test Files

#### 1. `internal/pricing/parser_regression_test.go`

**Purpose**: Prevent parser from returning $0 or unnormalized pricing values

**Key Tests**:
- `TestParseRealAWSResponse_NoZeroDollarValues`
  - Uses actual AWS Pricing API response structure
  - Verifies upfront prices are extracted: `assert.Equal(t, 29669.0, rate.UpfrontPrice)`
  - Verifies hourly prices are extracted: `assert.Equal(t, 1.728, rate.HourlyPrice)`
  - Confirms convertible offerings are filtered out

- `TestNormalizedPricing_NoZeroValues`
  - Tests normalization formula: `(upfront / termHours) + hourly`
  - Validates 1yr All Upfront: 29669 / 8760 = 3.3869
  - Validates 1yr Partial Upfront: (15137 / 8760) + 1.728 = 3.456
  - Validates 1yr No Upfront: 3.629
  - Critical assertions:
    ```go
    assert.NotEqual(t, 0.0, normalizedRate,
        "Normalized rate should NOT be $0 - this was the original bug!")
    assert.Less(t, normalizedRate, 50.0,
        "Normalized rate seems too high - possible missing division")
    ```

- `TestUnitFieldDetection`
  - Ensures parser correctly identifies `unit: "Quantity"` as upfront price
  - Ensures parser correctly identifies `unit: "Hrs"` as hourly price

- `TestOfferingClassFiltering`
  - Confirms only "standard" class RIs are included
  - Confirms "convertible" class RIs are filtered out

- `TestRegressionForLargeRawValues`
  - Detects if normalization is skipped
  - Validates rates are < $10/hour (not thousands)

#### 2. `internal/pricing/integration_test.go`

**Purpose**: End-to-end integration tests (requires `go test -tags integration`)

**Key Test**:
- `TestEndToEndPricing_NoZeroValues`
  - Full pipeline: AWS response → parsing → normalization → PricePoint creation
  - Validates no $0 values in final output
  - Validates no raw upfront values in final output
  - Validates expected rates match calculations
  - Validates all expected terms are present

#### 3. `internal/models/validation_test.go`

**Purpose**: Validate catalog.json output for pricing anomalies

**Key Function**: `ValidateCatalogPricing(catalog *Catalog) []string`

Detects:
- **$0 values** (the original bug)
  ```go
  if rate == 0.0 {
      errors = append(errors,
          formatError(item, termKey, "$0 value detected - this was the bug!"))
  }
  ```

- **Unnormalized upfront prices** (>$100/hr)
  ```go
  if rate > 100.0 {
      errors = append(errors,
          formatError(item, termKey,
              "suspiciously high rate (>$100/hr) - may be unnormalized upfront price"))
  }
  ```

- **Suspiciously low rates** (<$0.50/hr)
- **RI rates higher than On-Demand** (should be cheaper)
- **Missing or incomplete RI data**

**Key Tests**:
- `TestCatalogValidation_DetectsZeroValues` - Confirms validation catches $0 bugs
- `TestCatalogValidation_DetectsUnnormalizedValues` - Confirms validation catches raw upfront prices
- `TestCatalogValidation_FromFile` - Validates actual catalog.json file

## Running Tests

### Run all unit tests
```bash
go test ./...
```

### Run with verbose output
```bash
go test ./... -v
```

### Run integration tests
```bash
go test ./internal/pricing/... -tags integration -v
```

### Run with coverage
```bash
go test ./... -cover
```

### Validate actual catalog.json
```bash
go test ./internal/models/... -run TestCatalogValidation_FromFile -v
```

## Continuous Integration

All tests should be run in CI/CD pipeline before merging:

```yaml
# Example GitHub Actions workflow
- name: Run unit tests
  run: go test ./... -v

- name: Run integration tests
  run: go test ./internal/pricing/... -tags integration -v

- name: Generate coverage
  run: go test ./... -coverprofile=coverage.out

- name: Validate output
  run: |
    ./bin/awsmetalprices fetch --config config.yaml --out ./pricing
    go test ./internal/models/... -run TestCatalogValidation_FromFile
```

## What These Tests Protect Against

1. **$0 Reserved Instance pricing** - Multiple assertions ensure effective hourly rates are never zero
2. **Unnormalized upfront prices** - Tests verify rates are in reasonable hourly ranges ($0.50-$100)
3. **Missing unit field handling** - Tests ensure "Quantity" and "Hrs" units are correctly parsed
4. **Convertible RI leakage** - Tests confirm only "standard" class RIs are included
5. **Parsing regressions** - Tests use actual AWS response structure to catch API changes

## Future Maintenance

If AWS changes their Pricing API structure:
1. Tests will fail, alerting developers to the change
2. Debug logging in `client.go` captures raw responses to `.debug/` directory
3. Update parser logic in `internal/pricing/parser.go`
4. Update test data in regression tests to match new structure
5. Re-run all tests to verify fix

## Test Results

All tests passing as of 2025-10-26:

```
=== Pricing Package ===
✅ TestParseRealAWSResponse_NoZeroDollarValues
✅ TestNormalizedPricing_NoZeroValues
✅ TestUnitFieldDetection
✅ TestOfferingClassFiltering
✅ TestRegressionForLargeRawValues
✅ TestEndToEndPricing_NoZeroValues (integration)

=== Models Package ===
✅ TestCatalogValidation_NoZeroDollarValues
✅ TestCatalogValidation_DetectsZeroValues
✅ TestCatalogValidation_DetectsUnnormalizedValues
✅ TestCatalogValidation_FromFile

Coverage: 48.8% (pricing), 100% (models)
```

Actual catalog.json validation: **PASSED** (no errors found in 36 items, 216 RI values)
