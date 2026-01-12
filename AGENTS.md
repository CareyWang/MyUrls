# AGENTS.md

## Build Commands
```bash
make default      # Development build
make all         # Cross-platform builds
make sync_data   # Build data sync tool
make fmt         # Format code
make clean       # Clean build artifacts
```

## Test Commands
```bash
go test ./...                    # Run all tests
go test ./internal/service/...   # Run specific package
SKIP_REDIS_TESTS=1 go test ./... # Skip Redis tests
go test -run TestFunctionName    # Run single test function
```

## Code Style
- **Naming**: PascalCase for exported types/functions, camelCase for variables
- **Imports**: Standard lib → Third-party → Internal (github.com/CareyWang/MyUrls)
- **Comments**: Chinese for business logic, English for technical details
- **Architecture**: Layered (cmd/, internal/{config,handler,service,storage,model,utils})
- **Testing**: Table-driven tests with testify/assert, co-located *_test.go files
- **Error Handling**: Service layer returns empty string on error, handlers return proper HTTP codes
- **Configuration**: Multi-source (flags → env vars → TOML → defaults) with MYURLS_ prefix