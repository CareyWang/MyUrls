## internal/service

### OVERVIEW
Business logic layer for URL mapping operations. Provides core URL shortening,
redirection, and TTL management. Delegates persistence to storage driver.

### WHERE TO LOOK
- `url.go` - Core service functions: ShortToLong, LongToShort, Renew, CheckKeyExists
- `url_test.go` - Service-level tests with storage sqlite setup

### CONVENTIONS
- Functions accept context.Context as first parameter
- ShortToLong returns empty string on error (repo error handling convention)
- Other functions return error directly
- Renew returns nil if TTL < 0 (key absent or permanent), no-op for nonexistent keys
- Uses storage.GetDriver() for persistence access
- Tests: testify/assert for assertions, require for setup, cleanup with defer

### ANTI-PATTERNS
- Don't handle storage errors in service layer - propagate to handlers
- Don't add business logic to storage calls - keep pure delegation
- Don't check ShortToLong result for nil - empty string indicates error
- Don't assume Renew modifies TTL on nonexistent keys - check TTL first
- Don't duplicate storage logic - service layer orchestrates only
