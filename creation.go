package main

import (
	"context"
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxLongURLLength  = 8192
	maxShortKeyLength = 20
)

// creationInputError is a rejected creation input. Its text is shown to the caller.
type creationInputError string

func (e creationInputError) Error() string {
	return string(e)
}

const (
	errInvalidLongURL     creationInputError = "invalid long URL, only http and https URLs are supported"
	errLongURLTooLong     creationInputError = "long URL is too long, at most 8192 characters are allowed"
	errSelfReferencingURL creationInputError = "long URL must not point to this service"
	errInvalidShortKey    creationInputError = "invalid short key, only letters, digits, hyphens, and underscores are allowed"
	errShortKeyTooLong    creationInputError = "short key is too long, at most 20 characters are allowed"
	errShortKeyTaken      creationInputError = "short key already exists, please use another one or leave it empty to generate automatically"
)

// A dot-free short key cannot shadow a static file such as /logo.png.
var shortKeyPattern = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)

// reservedShortKeys cannot be requested because they name routes, current or planned.
// TestRegisteredRoutesCannotBeShortKeys fails when a new route is missing here.
var reservedShortKeys = map[string]bool{
	"short": true,
	"api":   true,
	"admin": true,
}

// shortURLCreator applies the creation rules and builds short URLs.
type shortURLCreator struct {
	mappings *shortURLMappings
	proto    string
	domain   string
}

func newShortURLCreator(mappings *shortURLMappings, proto, domain string) *shortURLCreator {
	return &shortURLCreator{mappings: mappings, proto: proto, domain: domain}
}

// Create validates the input and returns the short URL of a new mapping.
// The input may be a plain or legacy base64 long URL. An empty shortKey generates a key.
// Rejected input returns a creationInputError without touching the storage.
func (c *shortURLCreator) Create(ctx context.Context, longURLInput, shortKey string) (string, error) {
	longURL, err := c.normalizeLongURL(longURLInput)
	if err != nil {
		return "", err
	}
	if shortKey != "" {
		if err := validateShortKey(shortKey); err != nil {
			return "", err
		}
	}

	shortKey, err = c.mappings.Create(ctx, longURL, shortKey)
	if errors.Is(err, errShortKeyExists) {
		return "", errShortKeyTaken
	}
	if err != nil {
		return "", err
	}
	return c.proto + "://" + c.domain + "/" + shortKey, nil
}

// normalizeLongURL returns the long URL from plain or legacy base64 input.
func (c *shortURLCreator) normalizeLongURL(input string) (string, error) {
	longURL := input
	parsed, ok := parseHTTPURL(longURL)
	if !ok {
		// 兼容以前的实现，这里如果是 base64 编码的字符串，进行解码
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return "", errInvalidLongURL
		}
		longURL = string(decoded)
		if parsed, ok = parseHTTPURL(longURL); !ok {
			return "", errInvalidLongURL
		}
	}

	if utf8.RuneCountInString(longURL) > maxLongURLLength {
		return "", errLongURLTooLong
	}
	if canonicalAuthority(parsed.Scheme, parsed.Host) == canonicalAuthority(c.proto, c.domain) {
		return "", errSelfReferencingURL
	}
	return longURL, nil
}

func parseHTTPURL(rawURL string) (*url.URL, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, false
	}
	return parsed, (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

// canonicalAuthority makes equivalent hosts comparable, ignoring case, a trailing
// dot, and the scheme's default port.
func canonicalAuthority(scheme, host string) string {
	authority := url.URL{Host: host}
	port := authority.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	return strings.TrimSuffix(strings.ToLower(authority.Hostname()), ".") + ":" + port
}

// validateShortKey checks a user-specified short key. A reserved key reads as taken,
// because to the caller it is simply a key that cannot be claimed.
func validateShortKey(shortKey string) error {
	if !shortKeyPattern.MatchString(shortKey) {
		return errInvalidShortKey
	}
	if len(shortKey) > maxShortKeyLength {
		return errShortKeyTooLong
	}
	if reservedShortKeys[shortKey] {
		return errShortKeyTaken
	}
	return nil
}
