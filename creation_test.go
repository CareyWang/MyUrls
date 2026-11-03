package main

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestCreator(t testing.TB) (*miniredis.Miniredis, *shortURLCreator) {
	t.Helper()
	server, mappings := newTestMappings(t)
	return server, newShortURLCreator(mappings, "https", "s.example.net")
}

func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestCreatorRejectsInvalidInput(t *testing.T) {
	validLongURL := "https://example.com"
	for _, tc := range []struct {
		name     string
		longURL  string
		shortKey string
		wantErr  error
	}{
		{"plain text decodable as base64", "example/path", "", errInvalidLongURL},
		{"missing scheme", "example.com", "", errInvalidLongURL},
		{"javascript scheme", "javascript:alert(1)", "", errInvalidLongURL},
		{"base64 javascript scheme", encodeBase64("javascript:alert(1)"), "", errInvalidLongURL},
		{"missing host", "https://", "", errInvalidLongURL},
		{"long URL over limit", "https://example.com/" + strings.Repeat("a", maxLongURLLength), "", errLongURLTooLong},
		{"base64 long URL over limit", encodeBase64("https://example.com/" + strings.Repeat("a", maxLongURLLength)), "", errLongURLTooLong},
		{"short key shadows static file", validLongURL, "logo.png", errInvalidShortKey},
		{"short key with slash", validLongURL, "a/b", errInvalidShortKey},
		{"short key with query", validLongURL, "q?x", errInvalidShortKey},
		{"short key over limit", validLongURL, strings.Repeat("a", maxShortKeyLength+1), errShortKeyTooLong},
		{"short key names a route", validLongURL, "short", errShortKeyTaken},
		{"short key reserved for api", validLongURL, "api", errShortKeyTaken},
		{"short key reserved for admin", validLongURL, "admin", errShortKeyTaken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, creator := newTestCreator(t)
			shortURL, err := creator.Create(context.Background(), tc.longURL, tc.shortKey)
			require.ErrorIs(t, err, tc.wantErr)
			require.Empty(t, shortURL)
			require.Empty(t, server.Keys())
		})
	}
}

func TestCreatorRejectsLongURLsPointingToItself(t *testing.T) {
	for _, tc := range []struct {
		name    string
		proto   string
		domain  string
		longURL string
		wantErr error
	}{
		{"same host", "https", "s.example.net", "https://s.example.net/abc", errSelfReferencingURL},
		{"base64", "https", "s.example.net", encodeBase64("https://s.example.net/abc"), errSelfReferencingURL},
		{"host case", "https", "s.example.net", "https://S.Example.NET/abc", errSelfReferencingURL},
		{"trailing dot", "https", "s.example.net", "https://s.example.net./abc", errSelfReferencingURL},
		{"other scheme", "https", "s.example.net", "http://s.example.net/abc", errSelfReferencingURL},
		{"default port", "https", "s.example.net", "https://s.example.net:443/abc", errSelfReferencingURL},
		{"user info", "https", "s.example.net", "https://user@s.example.net/abc", errSelfReferencingURL},
		{"configured port", "http", "localhost:8080", "http://localhost:8080/abc", errSelfReferencingURL},
		{"other port", "https", "s.example.net", "https://s.example.net:8443/abc", nil},
		{"other configured port", "http", "localhost:8080", "http://localhost:9090/abc", nil},
		{"subdomain", "https", "s.example.net", "https://a.s.example.net/abc", nil},
		{"domain in path", "https", "s.example.net", "https://example.com/s.example.net", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, mappings := newTestMappings(t)
			creator := newShortURLCreator(mappings, tc.proto, tc.domain)
			_, err := creator.Create(context.Background(), tc.longURL, "custom")
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr != nil {
				require.Empty(t, server.Keys())
			} else {
				requireMapping(t, mappings, "custom", tc.longURL)
			}
		})
	}
}

func TestCreatorStoresNormalizedLongURL(t *testing.T) {
	maxLengthURL := "https://example.com/"
	maxLengthURL += strings.Repeat("字", maxLongURLLength-len(maxLengthURL))
	for _, tc := range []struct {
		name    string
		input   string
		wantURL string
	}{
		{"plain", "https://example.com/path?a=1", "https://example.com/path?a=1"},
		{"http", "http://example.com", "http://example.com"},
		{"base64", encodeBase64("https://example.com/path?a=1"), "https://example.com/path?a=1"},
		{"at length limit", maxLengthURL, maxLengthURL},
		{"base64 at length limit", encodeBase64(maxLengthURL), maxLengthURL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, creator := newTestCreator(t)
			_, err := creator.Create(context.Background(), tc.input, "custom")
			require.NoError(t, err)
			requireMapping(t, creator.mappings, "custom", tc.wantURL)
		})
	}
}

func TestCreatorBuildsShortURL(t *testing.T) {
	maxLengthKey := strings.Repeat("a", maxShortKeyLength)
	for _, tc := range []struct {
		name     string
		shortKey string
		wantKey  string
	}{
		{"custom key", "my-custom_Key1", "my-custom_Key1"},
		{"key at length limit", maxLengthKey, maxLengthKey},
		{"reserved names are case-sensitive", "Short", "Short"},
		{"generated key", "", "autoKey"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, mappings := newTestMappings(t)
			mappings.generateKey = func() (string, error) { return "autoKey", nil }
			creator := newShortURLCreator(mappings, "http", "s.example.net:8080")
			shortURL, err := creator.Create(context.Background(), "https://example.com", tc.shortKey)
			require.NoError(t, err)
			require.Equal(t, "http://s.example.net:8080/"+tc.wantKey, shortURL)
			requireMapping(t, mappings, tc.wantKey, "https://example.com")
		})
	}
}

func TestCreatorReportsShortKeyConflict(t *testing.T) {
	server, creator := newTestCreator(t)
	require.NoError(t, server.Set("custom", "https://example.com/original"))
	shortURL, err := creator.Create(context.Background(), "https://example.com/new", "custom")
	require.ErrorIs(t, err, errShortKeyTaken)
	require.Empty(t, shortURL)
	requireMapping(t, creator.mappings, "custom", "https://example.com/original")
}

func TestCreationRulesDoNotAffectExistingMappings(t *testing.T) {
	server, creator := newTestCreator(t)
	require.NoError(t, server.Set("short", "javascript:alert(1)"))
	requireMapping(t, creator.mappings, "short", "javascript:alert(1)")
}

func TestRegisteredRoutesCannotBeShortKeys(t *testing.T) {
	_, creator := newTestCreator(t)
	router := gin.New()
	registerRoutes(router, creator.mappings, creator)

	routes := router.Routes()
	require.NotEmpty(t, routes)
	for _, route := range routes {
		segment, _, _ := strings.Cut(strings.TrimPrefix(route.Path, "/"), "/")
		if segment == "" || segment[0] == ':' || segment[0] == '*' {
			continue
		}
		require.Errorf(t, validateShortKey(segment), "route %s %s can be shadowed by a short key", route.Method, route.Path)
	}
}
