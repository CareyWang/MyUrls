package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
