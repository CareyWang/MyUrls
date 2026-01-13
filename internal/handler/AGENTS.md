# internal/handler

## OVERVIEW
HTTP request handling layer using Gin framework. Validates requests, calls service layer, formats responses. Two main handlers: URL routing and cache management.

## WHERE TO LOOK
- **url.go**: GET /:shortKey (redirect) and POST /short (create)
- **cache.go**: Cache clear endpoint with auth checks

## CONVENTIONS
**Response Patterns:**
- Errors: `model.Response` with Code/Msg structure
- Success: Legacy format (Code=1) for short creation, 301 redirect for retrieval
- Status codes: 404 for missing URLs, 200 for API responses, 403/401 for auth failures

**Defaults:**
- TTL: 365 days (`time.Hour * 24 * 365`)
- Short key length: 7 characters
- Auto-generate key if empty

**Auth Checks (cache.go):**
- Requires BOTH localhost IP AND valid token
- IP check: 127.0.0.1, ::1, localhost, or loopback
- Token: Query parameter `token` matches configured cacheToken
- Returns 403 on IP fail, 401 on token fail

**Request Binding:**
- Use `c.ShouldBind()` with struct tags for validation
- Base64 decode support for longUrl parameter

## ANTI-PATTERNS
- Don't bypass service layer for business logic
- Don't return 500 on validation errors (use 200 with error response)
- Don't skip localhost checks on cache clear endpoint
