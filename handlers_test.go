package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestShortToLongHandler(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	server, mappings := newTestMappings(t)
	require.NoError(t, server.Set("custom", "https://example.com"))
	router := gin.New()
	router.GET("/:shortKey", ShortToLongHandler(mappings))
	get := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}

	response := get("/custom")
	require.Equal(t, http.StatusFound, response.Code)
	require.Equal(t, "https://example.com", response.Header().Get("Location"))

	response = get("/missing")
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Contains(t, response.Body.String(), "exists or expired")

	server.SetError("ERR storage unavailable")
	response = get("/custom")
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.NotContains(t, response.Body.String(), "exists or expired")
	require.NotContains(t, response.Body.String(), "storage unavailable")
}

func postShort(t *testing.T, creator *shortURLCreator, contentType string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/short", bytes.NewReader(body))
	request.Header.Set("Content-Type", contentType)
	router := gin.New()
	router.POST("/short", LongToShortHandler(creator))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func postShortJSON(t *testing.T, creator *shortURLCreator, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(params)
	require.NoError(t, err)
	return postShort(t, creator, "application/json", body)
}

func TestLongToShortHandlerAcceptsSubmissionFormats(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	const longURL = "https://example.com/path?a=1"
	encoded := base64.StdEncoding.EncodeToString([]byte(longURL))
	urlEncoded := func(longURLInput string) (string, []byte) {
		return "application/x-www-form-urlencoded", []byte(url.Values{"longUrl": {longURLInput}, "shortKey": {"custom"}}.Encode())
	}
	jsonEncoded := func(longURLInput string) (string, []byte) {
		body, err := json.Marshal(map[string]string{"longUrl": longURLInput, "shortKey": "custom"})
		require.NoError(t, err)
		return "application/json", body
	}
	// The built-in frontend posts a FormData with a btoa() long URL.
	multipartEncoded := func(longURLInput string) (string, []byte) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("longUrl", longURLInput))
		require.NoError(t, writer.WriteField("shortKey", "custom"))
		require.NoError(t, writer.Close())
		return writer.FormDataContentType(), body.Bytes()
	}

	for _, tc := range []struct {
		name   string
		encode func(string) (string, []byte)
		input  string
	}{
		{"plain JSON", jsonEncoded, longURL},
		{"base64 JSON", jsonEncoded, encoded},
		{"plain form", urlEncoded, longURL},
		{"base64 form", urlEncoded, encoded},
		{"built-in frontend", multipartEncoded, encoded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, creator := newTestCreator(t)
			contentType, body := tc.encode(tc.input)
			response := postShort(t, creator, contentType, body)

			require.Equal(t, http.StatusOK, response.Code)
			require.JSONEq(t, `{"Code": 1, "ShortUrl": "https://s.example.net/custom"}`, response.Body.String())
			requireMapping(t, creator.mappings, "custom", longURL)
		})
	}
}

func TestLongToShortHandlerBoundsRequestBody(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	// The longest allowed long URL, with four-byte runes, still fits once encoded.
	prefix := "https://example.com/"
	longest := prefix + strings.Repeat("😀", maxLongURLLength-len(prefix))
	for name, input := range map[string]string{"plain": longest, "base64": encodeBase64(longest)} {
		t.Run(name, func(t *testing.T) {
			_, creator := newTestCreator(t)
			body := []byte(url.Values{"longUrl": {input}}.Encode())
			response := postShort(t, creator, "application/x-www-form-urlencoded", body)
			require.Contains(t, response.Body.String(), `"ShortUrl"`)
		})
	}

	// A larger body is rejected while reading, before the length rule runs.
	_, creator := newTestCreator(t)
	response := postShortJSON(t, creator, map[string]string{"longUrl": prefix + strings.Repeat("a", maxShortRequestBytes)})
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"Message":"invalid parameters"`)
}

func TestLongToShortHandlerMapsFailures(t *testing.T) {
	previousLogger := logger
	logger = zap.NewNop().Sugar()
	t.Cleanup(func() { logger = previousLogger })

	const longURL = "https://example.com"
	for _, tc := range []struct {
		name        string
		params      map[string]string
		arrange     func(*testing.T, *miniredis.Miniredis, *shortURLCreator)
		wantCode    int
		wantMessage string
	}{
		{
			name:        "missing long URL",
			params:      map[string]string{"shortKey": "custom"},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "invalid parameters",
		},
		{
			name:        "invalid long URL",
			params:      map[string]string{"longUrl": "example.com"},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "invalid long URL, only http and https URLs are supported",
		},
		{
			name:        "long URL over limit",
			params:      map[string]string{"longUrl": longURL + "/" + strings.Repeat("a", maxLongURLLength)},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "long URL is too long, at most 8192 characters are allowed",
		},
		{
			name:        "self-referencing long URL",
			params:      map[string]string{"longUrl": "https://s.example.net/abc"},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "long URL must not point to this service",
		},
		{
			name:        "invalid short key",
			params:      map[string]string{"longUrl": longURL, "shortKey": "logo.png"},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "invalid short key, only letters, digits, hyphens, and underscores are allowed",
		},
		{
			name:        "short key over limit",
			params:      map[string]string{"longUrl": longURL, "shortKey": strings.Repeat("a", maxShortKeyLength+1)},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "short key is too long, at most 20 characters are allowed",
		},
		{
			name:        "reserved short key",
			params:      map[string]string{"longUrl": longURL, "shortKey": "short"},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "short key already exists, please use another one or leave it empty to generate automatically",
		},
		{
			name:   "short key conflict",
			params: map[string]string{"longUrl": longURL, "shortKey": "custom"},
			arrange: func(t *testing.T, server *miniredis.Miniredis, _ *shortURLCreator) {
				require.NoError(t, server.Set("custom", "original"))
			},
			wantCode:    ResponseCodeParamsCheckError,
			wantMessage: "short key already exists, please use another one or leave it empty to generate automatically",
		},
		{
			name:   "retry exhaustion",
			params: map[string]string{"longUrl": longURL},
			arrange: func(t *testing.T, server *miniredis.Miniredis, creator *shortURLCreator) {
				require.NoError(t, server.Set("autoKey", "original"))
				creator.mappings.generateKey = func() (string, error) { return "autoKey", nil }
			},
			wantCode:    ResponseCodeServerError,
			wantMessage: "failed to create short URL",
		},
		{
			name:   "generation error",
			params: map[string]string{"longUrl": longURL},
			arrange: func(_ *testing.T, _ *miniredis.Miniredis, creator *shortURLCreator) {
				creator.mappings.generateKey = func() (string, error) {
					return "", errors.New("random source unavailable")
				}
			},
			wantCode:    ResponseCodeServerError,
			wantMessage: "failed to create short URL",
		},
		{
			name:   "storage error",
			params: map[string]string{"longUrl": longURL, "shortKey": "custom"},
			arrange: func(_ *testing.T, server *miniredis.Miniredis, _ *shortURLCreator) {
				server.SetError("ERR storage unavailable")
			},
			wantCode:    ResponseCodeServerError,
			wantMessage: "failed to create short URL",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, creator := newTestCreator(t)
			if tc.arrange != nil {
				tc.arrange(t, server, creator)
			}
			response := postShortJSON(t, creator, tc.params)

			require.Equal(t, http.StatusOK, response.Code)
			expected, err := json.Marshal(Response{Code: tc.wantCode, Msg: tc.wantMessage})
			require.NoError(t, err)
			require.JSONEq(t, string(expected), response.Body.String())
		})
	}
}
