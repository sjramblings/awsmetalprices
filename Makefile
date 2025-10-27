.PHONY: build test fetch validate print diff clean install test-coverage test-integration test-regression test-validate-catalog

# Build the binary
build:
	go build -o bin/awsmetalprices ./cmd/awsmetalprices

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests (requires build tag)
test-integration:
	go test ./internal/pricing/... -tags integration -v

# Run regression tests for $0 pricing bug
test-regression:
	@echo "Running regression tests to prevent $0 pricing bug..."
	go test ./internal/pricing/... -run TestParseRealAWSResponse_NoZeroDollarValues -v
	go test ./internal/pricing/... -run TestNormalizedPricing_NoZeroValues -v
	go test ./internal/pricing/... -run TestUnitFieldDetection -v
	go test ./internal/pricing/... -run TestOfferingClassFiltering -v
	go test ./internal/pricing/... -run TestRegressionForLargeRawValues -v
	go test ./internal/models/... -run TestCatalogValidation -v
	@echo "All regression tests passed!"

# Validate actual catalog.json output
test-validate-catalog:
	@echo "Validating actual catalog.json file..."
	go test ./internal/models/... -run TestCatalogValidation_FromFile -v
	@echo "Catalog validation passed!"

# Fetch pricing data
fetch:
	./bin/awsmetalprices fetch --config config.yaml --out ./pricing

# Validate catalog
validate:
	./bin/awsmetalprices validate --catalog pricing/catalog.json

# Print pricing for a specific instance
print:
	./bin/awsmetalprices print --catalog pricing/catalog.json

# Compare two catalogs
diff:
	@echo "Usage: make diff OLD=<path> NEW=<path>"
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Error: Please specify OLD and NEW catalog paths"; \
		exit 1; \
	fi
	./bin/awsmetalprices diff --old $(OLD) --new $(NEW)

# Clean build artifacts and cache
clean:
	rm -rf bin/
	rm -rf .cache/
	rm -rf pricing/
	rm -f coverage.out coverage.html

# Install the binary to GOPATH/bin
install:
	go install ./cmd/awsmetalprices

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Run all checks (fmt, lint, test)
check: fmt lint test
	@echo "All checks passed!"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o bin/awsmetalprices-linux-amd64 ./cmd/awsmetalprices
	GOOS=darwin GOARCH=amd64 go build -o bin/awsmetalprices-darwin-amd64 ./cmd/awsmetalprices
	GOOS=darwin GOARCH=arm64 go build -o bin/awsmetalprices-darwin-arm64 ./cmd/awsmetalprices
	GOOS=windows GOARCH=amd64 go build -o bin/awsmetalprices-windows-amd64.exe ./cmd/awsmetalprices
	@echo "Binaries built in bin/"

# Display help
help:
	@echo "Available targets:"
	@echo "  build                  - Build the awsmetalprices binary"
	@echo "  test                   - Run all tests"
	@echo "  test-verbose           - Run tests with verbose output"
	@echo "  test-coverage          - Generate test coverage report"
	@echo "  test-integration       - Run integration tests with build tag"
	@echo "  test-regression        - Run regression tests for \$$0 pricing bug"
	@echo "  test-validate-catalog  - Validate actual catalog.json output"
	@echo "  fetch                  - Fetch pricing data using default config"
	@echo "  validate               - Validate the generated catalog"
	@echo "  print                  - Print pricing data from catalog"
	@echo "  diff                   - Compare two catalogs (usage: make diff OLD=path NEW=path)"
	@echo "  clean                  - Remove build artifacts and cache"
	@echo "  install                - Install binary to GOPATH/bin"
	@echo "  fmt                    - Format Go code"
	@echo "  lint                   - Run linter"
	@echo "  check                  - Run fmt, lint, and test"
	@echo "  build-all              - Build for multiple platforms"
	@echo "  help                   - Display this help message"
