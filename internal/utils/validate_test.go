package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateShortKey(t *testing.T) {
	valid32Chars := "0123456789abcdefghijklmnopqrstuv"

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "empty key is allowed for auto generation", key: "", wantErr: false},
		{name: "single digit", key: "0", wantErr: false},
		{name: "single lowercase", key: "a", wantErr: false},
		{name: "single uppercase", key: "Z", wantErr: false},
		{name: "mixed alphanumeric", key: "aZ09bY18", wantErr: false},
		{name: "max length 32", key: valid32Chars, wantErr: false},
		{name: "too long 33", key: valid32Chars + "w", wantErr: true},
		{name: "underscore not allowed", key: "abc_def", wantErr: true},
		{name: "hyphen not allowed", key: "abc-def", wantErr: true},
		{name: "slash not allowed", key: "abc/def", wantErr: true},
		{name: "space not allowed", key: "abc def", wantErr: true},
		{name: "chinese not allowed", key: "短链接", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShortKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLongURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "valid http", url: "http://example.com", wantErr: false},
		{name: "valid https with path and query", url: "https://example.com/path?a=1&b=2", wantErr: false},
		{name: "empty string", url: "", wantErr: true},
		{name: "javascript scheme", url: "javascript:alert(1)", wantErr: true},
		{name: "data scheme", url: "data:text/html,<script>alert(1)</script>", wantErr: true},
		{name: "no scheme", url: "example.com", wantErr: true},
		{name: "no host", url: "https://", wantErr: true},
		{name: "too long", url: "https://example.com/" + strings.Repeat("a", 10000), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLongURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
